package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &DatasetResource{}
	_ resource.ResourceWithImportState = &DatasetResource{}
	_ resource.ResourceWithModifyPlan  = &DatasetResource{}
)

// DatasetResource manages a TrueNAS ZFS dataset.
type DatasetResource struct {
	client *client.Client
}

// DatasetResourceModel describes the resource data model.
type DatasetResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Name          types.String   `tfsdk:"name"`
	Pool          types.String   `tfsdk:"pool"`
	ParentDataset types.String   `tfsdk:"parent_dataset"`
	Type          types.String   `tfsdk:"type"`
	Compression   types.String   `tfsdk:"compression"`
	Atime         types.String   `tfsdk:"atime"`
	Deduplication types.String   `tfsdk:"deduplication"`
	Quota         types.Int64    `tfsdk:"quota"`
	Refquota      types.Int64    `tfsdk:"refquota"`
	Comments      types.String   `tfsdk:"comments"`
	MountPoint    types.String   `tfsdk:"mount_point"`
	Sync          types.String   `tfsdk:"sync"`
	Snapdir       types.String   `tfsdk:"snapdir"`
	Copies        types.Int64    `tfsdk:"copies"`
	Readonly      types.String   `tfsdk:"readonly"`
	RecordSize    types.String   `tfsdk:"record_size"`
	ShareType     types.String   `tfsdk:"share_type"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewDatasetResource() resource.Resource {
	return &DatasetResource{}
}

func (r *DatasetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (r *DatasetResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a ZFS dataset on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The full dataset path (e.g., tank/mydata).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the dataset (without pool prefix).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-.:]*$`),
						"dataset name must start with alphanumeric and contain only alphanumeric, underscore, hyphen, period, or colon",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pool": schema.StringAttribute{
				Description: "The pool to create the dataset in.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-.]*$`),
						"pool name must start with a letter and contain only alphanumeric, underscore, hyphen, or period",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parent_dataset": schema.StringAttribute{
				Description: "Optional parent dataset path relative to pool (e.g., 'parent/child'). " +
					"The full dataset path will be pool/parent_dataset/name.",
				Optional: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(200),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The dataset type: FILESYSTEM or VOLUME.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("FILESYSTEM"),
				Validators: []validator.String{
					stringvalidator.OneOf("FILESYSTEM", "VOLUME"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"compression": schema.StringAttribute{
				Description: "Compression algorithm (OFF, LZ4, GZIP, ZSTD, ZLE, LZJB).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"OFF", "ON", "LZ4", "GZIP", "GZIP-1", "GZIP-2", "GZIP-3",
						"GZIP-4", "GZIP-5", "GZIP-6", "GZIP-7", "GZIP-8", "GZIP-9",
						"ZSTD", "ZSTD-FAST", "ZLE", "LZJB", "INHERIT",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"atime": schema.StringAttribute{
				Description: "Access time update behavior (ON, OFF).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ON", "OFF", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deduplication": schema.StringAttribute{
				Description: "Deduplication setting (ON, OFF, VERIFY).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ON", "OFF", "VERIFY", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"quota": schema.Int64Attribute{
				Description: "Dataset quota in bytes. 0 means no quota.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"refquota": schema.Int64Attribute{
				Description: "Dataset reference quota in bytes. 0 means no refquota.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"comments": schema.StringAttribute{
				Description: "User-provided comments for the dataset.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mount_point": schema.StringAttribute{
				Description: "The mount point of the dataset.",
				Computed:    true,
			},
			"sync": schema.StringAttribute{
				Description: "Sync write behavior (STANDARD, ALWAYS, DISABLED).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("STANDARD", "ALWAYS", "DISABLED", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"snapdir": schema.StringAttribute{
				Description: "Snapshot directory visibility (VISIBLE, HIDDEN).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("VISIBLE", "HIDDEN", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"copies": schema.Int64Attribute{
				Description: "Number of data copies (1, 2, or 3).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 3),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"readonly": schema.StringAttribute{
				Description: "Read-only setting (ON, OFF).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ON", "OFF", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"record_size": schema.StringAttribute{
				Description: "Record size (e.g., 128K, 1M). Valid values: 512, 1K, 2K, 4K, 8K, 16K, 32K, 64K, 128K, 256K, 512K, 1M.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("512", "1K", "2K", "4K", "8K", "16K", "32K", "64K", "128K", "256K", "512K", "1M", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"share_type": schema.StringAttribute{
				Description: "Share type preset (GENERIC, SMB).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("GENERIC", "SMB", "MULTIPROTOCOL", "NFS", "APPS"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *DatasetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

// buildFullName constructs the full dataset path from pool, parent_dataset, and name.
func buildFullName(pool, parent, name string) string {
	if parent != "" {
		return pool + "/" + parent + "/" + name
	}
	return pool + "/" + name
}

func (r *DatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Dataset start")

	var plan DatasetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	parent := ""
	if !plan.ParentDataset.IsNull() {
		parent = plan.ParentDataset.ValueString()
	}
	fullName := buildFullName(plan.Pool.ValueString(), parent, plan.Name.ValueString())

	createReq := &client.DatasetCreateRequest{
		Name: fullName,
	}

	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		createReq.Type = plan.Type.ValueString()
	}
	if !plan.Compression.IsNull() {
		createReq.Compression = plan.Compression.ValueString()
	}
	if !plan.Atime.IsNull() {
		createReq.Atime = plan.Atime.ValueString()
	}
	if !plan.Deduplication.IsNull() {
		createReq.Deduplication = plan.Deduplication.ValueString()
	}
	if !plan.Quota.IsNull() {
		createReq.Quota = plan.Quota.ValueInt64()
	}
	if !plan.Refquota.IsNull() {
		createReq.Refquota = plan.Refquota.ValueInt64()
	}
	if !plan.Comments.IsNull() {
		createReq.Comments = plan.Comments.ValueString()
	}
	if !plan.Sync.IsNull() {
		createReq.Sync = plan.Sync.ValueString()
	}
	if !plan.Snapdir.IsNull() {
		createReq.Snapdir = plan.Snapdir.ValueString()
	}
	if !plan.Copies.IsNull() {
		createReq.Copies = int(plan.Copies.ValueInt64())
	}
	if !plan.Readonly.IsNull() {
		createReq.Readonly = plan.Readonly.ValueString()
	}
	if !plan.RecordSize.IsNull() {
		createReq.RecordSize = plan.RecordSize.ValueString()
	}
	if !plan.ShareType.IsNull() {
		createReq.ShareType = plan.ShareType.ValueString()
	}

	tflog.Debug(ctx, "Creating dataset", map[string]interface{}{
		"name": fullName,
	})

	dataset, err := r.client.CreateDataset(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Dataset",
			fmt.Sprintf("Could not create dataset %q: %s", fullName, err),
		)
		return
	}

	r.mapResponseToModel(dataset, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Dataset success")
}

func (r *DatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Dataset start")

	var state DatasetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataset, err := r.client.GetDataset(ctx, state.ID.ValueString())
	if err != nil {
		// If the dataset no longer exists, remove from state
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Dataset",
			fmt.Sprintf("Could not read dataset %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(dataset, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Dataset success")
}

func (r *DatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Dataset start")

	var plan DatasetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DatasetResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.DatasetUpdateRequest{}

	if !plan.Compression.IsNull() {
		updateReq.Compression = plan.Compression.ValueString()
	}
	if !plan.Atime.IsNull() {
		updateReq.Atime = plan.Atime.ValueString()
	}
	if !plan.Deduplication.IsNull() {
		updateReq.Deduplication = plan.Deduplication.ValueString()
	}
	if !plan.Quota.IsNull() {
		updateReq.Quota = plan.Quota.ValueInt64()
	}
	if !plan.Refquota.IsNull() {
		updateReq.Refquota = plan.Refquota.ValueInt64()
	}
	if !plan.Comments.IsNull() {
		updateReq.Comments = plan.Comments.ValueString()
	}
	if !plan.Sync.IsNull() {
		updateReq.Sync = plan.Sync.ValueString()
	}
	if !plan.Snapdir.IsNull() {
		updateReq.Snapdir = plan.Snapdir.ValueString()
	}
	if !plan.Copies.IsNull() {
		updateReq.Copies = int(plan.Copies.ValueInt64())
	}
	if !plan.Readonly.IsNull() {
		updateReq.Readonly = plan.Readonly.ValueString()
	}
	if !plan.RecordSize.IsNull() {
		updateReq.RecordSize = plan.RecordSize.ValueString()
	}

	dataset, err := r.client.UpdateDataset(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Dataset",
			fmt.Sprintf("Could not update dataset %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(dataset, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Dataset success")
}

func (r *DatasetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Dataset start")

	var state DatasetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting dataset", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteDataset(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Dataset already deleted, removing from state", map[string]interface{}{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Dataset",
			fmt.Sprintf("Could not delete dataset %q: %s", state.ID.ValueString(), err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Dataset success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this dataset, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *DatasetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_dataset")
}

func (r *DatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToModel maps the API response to the Terraform resource model.
func (r *DatasetResource) mapResponseToModel(dataset *client.DatasetResponse, model *DatasetResourceModel) {
	model.ID = types.StringValue(dataset.ID)
	model.MountPoint = types.StringValue(dataset.MountPoint)

	// Parse pool and name from the full path
	parts := strings.SplitN(dataset.ID, "/", 2)
	if len(parts) >= 2 {
		model.Pool = types.StringValue(parts[0])
		// Determine name vs parent_dataset
		remaining := parts[1]
		lastSlash := strings.LastIndex(remaining, "/")
		if lastSlash >= 0 {
			model.ParentDataset = types.StringValue(remaining[:lastSlash])
			model.Name = types.StringValue(remaining[lastSlash+1:])
		} else {
			model.Name = types.StringValue(remaining)
			// No parent dataset in the ID path; leave model.ParentDataset as-is
			// (may be null for root-level datasets).
		}
	}

	model.Type = types.StringValue(dataset.Type)

	if dataset.Compression != nil {
		model.Compression = types.StringValue(dataset.Compression.Value)
	}
	if dataset.Atime != nil {
		model.Atime = types.StringValue(dataset.Atime.Value)
	}
	if dataset.Deduplication != nil {
		model.Deduplication = types.StringValue(dataset.Deduplication.Value)
	}
	if dataset.Quota != nil {
		if v, err := parseQuotaValue(dataset.Quota.Rawvalue); err == nil {
			model.Quota = types.Int64Value(v)
		}
	}
	if dataset.Refquota != nil {
		if v, err := parseQuotaValue(dataset.Refquota.Rawvalue); err == nil {
			model.Refquota = types.Int64Value(v)
		}
	}
	// SCALE 25.10+ moved comments from top-level `comments` (always null)
	// to `user_properties.comments`. GetComments() handles both shapes
	// transparently — see internal/client/dataset.go.
	model.Comments = types.StringValue(dataset.GetComments())
	if dataset.Sync != nil {
		model.Sync = types.StringValue(dataset.Sync.Value)
	}
	if dataset.Snapdir != nil {
		model.Snapdir = types.StringValue(dataset.Snapdir.Value)
	}
	if dataset.Copies != nil {
		if v, err := strconv.ParseInt(dataset.Copies.Value, 10, 64); err == nil {
			model.Copies = types.Int64Value(v)
		}
	}
	if dataset.Readonly != nil {
		model.Readonly = types.StringValue(dataset.Readonly.Value)
	}
	if dataset.RecordSize != nil {
		model.RecordSize = types.StringValue(dataset.RecordSize.Value)
	}
	// share_type is a create-time preset (applies ACL/case-sensitivity settings)
	// not a persistent ZFS property — the API returns "GENERIC" on read regardless
	// of what was requested at create. Preserve the user's value if already set in
	// the model; otherwise fall back to the API response or "GENERIC".
	if !model.ShareType.IsNull() && !model.ShareType.IsUnknown() {
		// keep user intent as-is
	} else if dataset.ShareType != nil {
		model.ShareType = types.StringValue(dataset.ShareType.Value)
	} else {
		model.ShareType = types.StringValue("GENERIC")
	}
}

// parseQuotaValue parses a raw quota string (may be "0" or byte count).
func parseQuotaValue(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}
