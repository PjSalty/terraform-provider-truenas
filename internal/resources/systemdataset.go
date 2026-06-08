package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &SystemDatasetResource{}
	_ resource.ResourceWithImportState = &SystemDatasetResource{}
)

// SystemDatasetResource manages the TrueNAS system dataset pool assignment.
// This is a singleton — only one system dataset configuration exists.
// PUT /systemdataset is asynchronous and the client layer waits for the
// job to complete.
//
// Delete resets the pool to null, which lets TrueNAS pick a default (boot
// pool) — this mirrors the behavior of the other singleton resources in
// this provider (e.g. ssh_config).
type SystemDatasetResource struct {
	client *client.Client
}

// SystemDatasetResourceModel describes the resource data model.
type SystemDatasetResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Pool     types.String   `tfsdk:"pool"`
	PoolSet  types.Bool     `tfsdk:"pool_set"`
	UUID     types.String   `tfsdk:"uuid"`
	Basename types.String   `tfsdk:"basename"`
	Path     types.String   `tfsdk:"path"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewSystemDatasetResource() resource.Resource {
	return &SystemDatasetResource{}
}

func (r *SystemDatasetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_systemdataset"
}

func (r *SystemDatasetResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the TrueNAS system dataset pool assignment. This is a singleton — " +
			"only one instance of this resource can exist per TrueNAS system. " +
			"Default timeouts: 20m for create/update (system dataset move requires service restarts), 10m for delete.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always 'systemdataset' (singleton).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool": schema.StringAttribute{
				Description: "The name of the pool hosting the system dataset. Set to an empty " +
					"string to let TrueNAS select the default (boot pool).",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 50),
				},
			},
			"pool_set": schema.BoolAttribute{
				Description: "Whether a pool has been explicitly set for the system dataset.",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "UUID of the system dataset.",
				Computed:    true,
			},
			"basename": schema.StringAttribute{
				Description: "Base name of the system dataset.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "Filesystem path to the system dataset.",
				Computed:    true,
			},
		},
	}
}

func (r *SystemDatasetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SystemDatasetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SystemDataset start")

	var plan SystemDatasetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating system dataset resource (updating singleton)",
		map[string]interface{}{"pool": plan.Pool.ValueString()})

	updateReq := buildSystemDatasetUpdate(&plan)
	cfg, err := r.client.UpdateSystemDataset(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating System Dataset",
			fmt.Sprintf("Could not update system dataset: %s", err),
		)
		return
	}

	r.mapResponseToModel(cfg, &plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SystemDataset success")
}

func (r *SystemDatasetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SystemDataset start")

	var state SystemDatasetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetSystemDataset(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading System Dataset",
			fmt.Sprintf("Could not read system dataset: %s", err),
		)
		return
	}

	r.mapResponseToModel(cfg, &state)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SystemDataset success")
}

func (r *SystemDatasetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SystemDataset start")

	var plan SystemDatasetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := buildSystemDatasetUpdate(&plan)
	cfg, err := r.client.UpdateSystemDataset(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating System Dataset",
			fmt.Sprintf("Could not update system dataset: %s", err),
		)
		return
	}

	r.mapResponseToModel(cfg, &plan)
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SystemDataset success")
}

func (r *SystemDatasetResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SystemDataset start")

	tflog.Debug(ctx, "Deleting system dataset resource (resetting to default boot-pool)")

	// Reset by sending null pool; TrueNAS will fall back to the boot pool.
	var nullPool *string
	_, err := r.client.UpdateSystemDataset(ctx, &client.SystemDatasetUpdateRequest{
		Pool: nullPool,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting System Dataset",
			fmt.Sprintf("Could not reset system dataset to default: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete SystemDataset success")
}

func (r *SystemDatasetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildSystemDatasetUpdate(plan *SystemDatasetResourceModel) *client.SystemDatasetUpdateRequest {
	updateReq := &client.SystemDatasetUpdateRequest{}
	if !plan.Pool.IsNull() && !plan.Pool.IsUnknown() {
		v := plan.Pool.ValueString()
		if v == "" {
			// Empty string means "reset to default" — send explicit null.
			updateReq.Pool = nil
		} else {
			updateReq.Pool = &v
		}
	}
	return updateReq
}

func (r *SystemDatasetResource) mapResponseToModel(cfg *client.SystemDataset, model *SystemDatasetResourceModel) {
	model.ID = types.StringValue("systemdataset")
	model.Pool = types.StringValue(cfg.Pool)
	model.PoolSet = types.BoolValue(cfg.PoolSet)
	model.UUID = types.StringValue(cfg.UUID)
	model.Basename = types.StringValue(cfg.Basename)
	model.Path = types.StringValue(cfg.Path)
}
