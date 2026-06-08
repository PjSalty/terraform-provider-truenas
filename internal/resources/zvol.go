package resources

import (
	"context"
	"fmt"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &ZvolResource{}
	_ resource.ResourceWithImportState = &ZvolResource{}
	_ resource.ResourceWithModifyPlan  = &ZvolResource{}
)

// ZvolResource manages a TrueNAS ZFS volume (zvol).
type ZvolResource struct {
	client *client.Client
}

// ZvolResourceModel describes the resource data model.
type ZvolResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Name          types.String   `tfsdk:"name"`
	Pool          types.String   `tfsdk:"pool"`
	Volsize       types.Int64    `tfsdk:"volsize"`
	Volblocksize  types.String   `tfsdk:"volblocksize"`
	Deduplication types.String   `tfsdk:"deduplication"`
	Compression   types.String   `tfsdk:"compression"`
	Comments      types.String   `tfsdk:"comments"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewZvolResource() resource.Resource {
	return &ZvolResource{}
}

func (r *ZvolResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zvol"
}

func (r *ZvolResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a ZFS volume (zvol) on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The full zvol path (e.g., tank/myvol).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the zvol (without pool prefix).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"pool": schema.StringAttribute{
				Description: "The pool to create the zvol in.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"volsize": schema.Int64Attribute{
				Description: "The size of the zvol in bytes.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"volblocksize": schema.StringAttribute{
				Description: "The block size of the zvol (e.g., 4K, 8K, 16K, 32K, 64K, 128K).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("16K"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("512", "1K", "2K", "4K", "8K", "16K", "32K", "64K", "128K"),
				},
			},
			"deduplication": schema.StringAttribute{
				Description: "Deduplication setting (ON, OFF, VERIFY).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ON", "OFF", "VERIFY"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"compression": schema.StringAttribute{
				Description: "Compression algorithm (OFF, LZ4, GZIP, ZSTD, ZLE, LZJB).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("OFF", "LZ4", "GZIP", "GZIP-1", "GZIP-9", "ZSTD", "ZSTD-FAST", "ZLE", "LZJB", "INHERIT"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comments": schema.StringAttribute{
				Description: "User-provided comments for the zvol.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ZvolResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ZvolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Zvol start")

	var plan ZvolResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fullName := plan.Pool.ValueString() + "/" + plan.Name.ValueString()

	createReq := &client.ZvolCreateRequest{
		Name:    fullName,
		Volsize: plan.Volsize.ValueInt64(),
	}

	if !plan.Volblocksize.IsNull() && !plan.Volblocksize.IsUnknown() {
		createReq.Volblocksize = plan.Volblocksize.ValueString()
	}
	if !plan.Deduplication.IsNull() {
		createReq.Deduplication = plan.Deduplication.ValueString()
	}
	if !plan.Compression.IsNull() {
		createReq.Compression = plan.Compression.ValueString()
	}
	if !plan.Comments.IsNull() {
		createReq.Comments = plan.Comments.ValueString()
	}

	tflog.Debug(ctx, "Creating zvol", map[string]interface{}{
		"name": fullName,
	})

	dataset, err := r.client.CreateZvol(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Zvol",
			fmt.Sprintf("Could not create zvol %q: %s", fullName, err),
		)
		return
	}

	r.mapResponseToModel(dataset, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Zvol success")
}

func (r *ZvolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Zvol start")

	var state ZvolResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataset, err := r.client.GetZvol(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Zvol",
			fmt.Sprintf("Could not read zvol %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(dataset, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Zvol success")
}

func (r *ZvolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Zvol start")

	var plan ZvolResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ZvolResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &client.ZvolUpdateRequest{}

	if !plan.Volsize.IsNull() {
		updateReq.Volsize = plan.Volsize.ValueInt64()
	}
	if !plan.Deduplication.IsNull() {
		updateReq.Deduplication = plan.Deduplication.ValueString()
	}
	if !plan.Compression.IsNull() {
		updateReq.Compression = plan.Compression.ValueString()
	}
	if !plan.Comments.IsNull() {
		updateReq.Comments = plan.Comments.ValueString()
	}

	dataset, err := r.client.UpdateZvol(ctx, state.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Zvol",
			fmt.Sprintf("Could not update zvol %q: %s", state.ID.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(dataset, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Zvol success")
}

func (r *ZvolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Zvol start")

	var state ZvolResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting zvol", map[string]interface{}{
		"id": state.ID.ValueString(),
	})

	err := r.client.DeleteZvol(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Zvol already deleted, removing from state", map[string]interface{}{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Zvol",
			fmt.Sprintf("Could not delete zvol %q: %s", state.ID.ValueString(), err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Zvol success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *ZvolResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_zvol")
}

func (r *ZvolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// mapResponseToModel maps the dataset API response to the zvol Terraform model.
func (r *ZvolResource) mapResponseToModel(dataset *client.DatasetResponse, model *ZvolResourceModel) {
	model.ID = types.StringValue(dataset.ID)

	// Parse pool and name from the full path
	parts := strings.SplitN(dataset.ID, "/", 2)
	if len(parts) >= 2 {
		model.Pool = types.StringValue(parts[0])
		model.Name = types.StringValue(parts[1])
	}

	// Zvol-specific properties.
	if v := dataset.GetVolsize(); v > 0 {
		model.Volsize = types.Int64Value(v)
	}
	if v := dataset.GetVolblocksize(); v != "" {
		model.Volblocksize = types.StringValue(v)
	}

	if dataset.Deduplication != nil {
		model.Deduplication = types.StringValue(dataset.Deduplication.Value)
	}
	if dataset.Compression != nil {
		model.Compression = types.StringValue(dataset.Compression.Value)
	}
	// SCALE 25.10+ moved comments from top-level `comments` (always null)
	// to `user_properties.comments`. GetComments() handles both shapes.
	model.Comments = types.StringValue(dataset.GetComments())
}
