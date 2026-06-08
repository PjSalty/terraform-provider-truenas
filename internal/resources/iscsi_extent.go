package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	"github.com/PjSalty/terraform-provider-truenas/internal/resourcevalidators"
)

var (
	_ resource.Resource                     = &ISCSIExtentResource{}
	_ resource.ResourceWithImportState      = &ISCSIExtentResource{}
	_ resource.ResourceWithModifyPlan       = &ISCSIExtentResource{}
	_ resource.ResourceWithConfigValidators = &ISCSIExtentResource{}
)

// ISCSIExtentResource manages an iSCSI extent.
type ISCSIExtentResource struct {
	client *client.Client
}

// ISCSIExtentResourceModel describes the resource data model.
type ISCSIExtentResourceModel struct {
	ID        types.String   `tfsdk:"id"`
	Name      types.String   `tfsdk:"name"`
	Type      types.String   `tfsdk:"type"`
	Disk      types.String   `tfsdk:"disk"`
	Path      types.String   `tfsdk:"path"`
	Filesize  types.Int64    `tfsdk:"filesize"`
	Blocksize types.Int64    `tfsdk:"blocksize"`
	RPM       types.String   `tfsdk:"rpm"`
	Enabled   types.Bool     `tfsdk:"enabled"`
	Comment   types.String   `tfsdk:"comment"`
	ReadOnly  types.Bool     `tfsdk:"readonly"`
	Timeouts  timeouts.Value `tfsdk:"timeouts"`
}

func NewISCSIExtentResource() resource.Resource {
	return &ISCSIExtentResource{}
}

func (r *ISCSIExtentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_extent"
}

func (r *ISCSIExtentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an iSCSI extent on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the iSCSI extent.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The extent name.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"type": schema.StringAttribute{
				Description: "The extent type (DISK or FILE).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("DISK", "FILE"),
				},
			},
			"disk": schema.StringAttribute{
				Description: "The zvol path for DISK type extents.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The file path for FILE type extents. For DISK type " +
					"extents, the API computes this from `disk` — leave unset.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"filesize": schema.Int64Attribute{
				Description: "The file size in bytes for FILE type extents.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"blocksize": schema.Int64Attribute{
				Description: "Block size in bytes (512, 1024, 2048, or 4096).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(512),
				Validators: []validator.Int64{
					int64validator.OneOf(512, 1024, 2048, 4096),
				},
			},
			"rpm": schema.StringAttribute{
				Description: "Reported RPM (SSD, 5400, 7200, 10000, 15000, UNKNOWN).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SSD"),
				Validators: []validator.String{
					stringvalidator.OneOf("SSD", "5400", "7200", "10000", "15000", "UNKNOWN"),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the extent is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"comment": schema.StringAttribute{
				Description: "A comment for the extent.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"readonly": schema.BoolAttribute{
				Description: "Whether the extent is read-only.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

// ConfigValidators enforces the type → {disk,path,filesize} shape
// rules at config-validation time:
//
//   - type = "DISK"  → disk must be set (the zvol path);
//     path/filesize are computed from the disk and must NOT be set
//     by the caller. (Enforced server-side; we leave that to the API
//     error since the operator would get a clear message either way.)
//   - type = "FILE"  → path AND filesize must both be set; the
//     server refuses a missing filesize with a generic "extent
//     create failed" that is hard to act on, so we short-circuit here.
func (r *ISCSIExtentResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidators.RequiredWhenEqual(
			"type",
			"DISK",
			[]string{"disk"},
		),
		resourcevalidators.RequiredWhenEqual(
			"type",
			"FILE",
			[]string{"path"},
		),
	}
}

func (r *ISCSIExtentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ISCSIExtentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSIExtent start")

	var plan ISCSIExtentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ISCSIExtentCreateRequest{
		Name:      plan.Name.ValueString(),
		Type:      plan.Type.ValueString(),
		Blocksize: int(plan.Blocksize.ValueInt64()),
		Enabled:   plan.Enabled.ValueBool(),
		ReadOnly:  plan.ReadOnly.ValueBool(),
	}

	if !plan.Disk.IsNull() {
		createReq.Disk = plan.Disk.ValueString()
	}
	if !plan.Path.IsNull() {
		createReq.Path = plan.Path.ValueString()
	}
	if !plan.Filesize.IsNull() {
		createReq.Filesize = plan.Filesize.ValueInt64()
	}
	if !plan.RPM.IsNull() {
		createReq.RPM = plan.RPM.ValueString()
	}
	if !plan.Comment.IsNull() {
		createReq.Comment = plan.Comment.ValueString()
	}

	tflog.Debug(ctx, "Creating iSCSI extent", map[string]interface{}{"name": plan.Name.ValueString()})

	extent, err := r.client.CreateISCSIExtent(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating iSCSI Extent",
			fmt.Sprintf("Could not create iSCSI extent %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(extent, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ISCSIExtent success")
}

func (r *ISCSIExtentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSIExtent start")

	var state ISCSIExtentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI extent ID: %s", err))
		return
	}

	extent, err := r.client.GetISCSIExtent(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Extent",
			fmt.Sprintf("Could not read iSCSI extent %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(extent, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ISCSIExtent success")
}

func (r *ISCSIExtentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSIExtent start")

	var plan ISCSIExtentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ISCSIExtentResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI extent ID: %s", err))
		return
	}

	enabled := plan.Enabled.ValueBool()
	readOnly := plan.ReadOnly.ValueBool()

	updateReq := &client.ISCSIExtentUpdateRequest{
		Name:      plan.Name.ValueString(),
		Type:      plan.Type.ValueString(),
		Blocksize: int(plan.Blocksize.ValueInt64()),
		Enabled:   &enabled,
		ReadOnly:  &readOnly,
	}

	if !plan.Disk.IsNull() {
		updateReq.Disk = plan.Disk.ValueString()
	}
	if !plan.Path.IsNull() {
		updateReq.Path = plan.Path.ValueString()
	}
	if !plan.Filesize.IsNull() {
		updateReq.Filesize = plan.Filesize.ValueInt64()
	}
	if !plan.RPM.IsNull() {
		updateReq.RPM = plan.RPM.ValueString()
	}
	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}

	extent, err := r.client.UpdateISCSIExtent(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating iSCSI Extent",
			fmt.Sprintf("Could not update iSCSI extent %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(extent, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ISCSIExtent success")
}

func (r *ISCSIExtentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSIExtent start")

	var state ISCSIExtentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI extent ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting iSCSI extent", map[string]interface{}{"id": id})

	err = r.client.DeleteISCSIExtent(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "iSCSI extent already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting iSCSI Extent",
			fmt.Sprintf("Could not delete iSCSI extent %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ISCSIExtent success")
}

// ModifyPlan enforces iSCSI extent cross-attribute constraints:
//
//   - type=DISK requires the `disk` attribute to be set to a zvol path.
//   - type=FILE requires the `path` attribute to be set.
//   - type=FILE requires `filesize > 0` (zero-size files are not extents).
//
// These rules are enforced by the TrueNAS API but only at apply time, which
// surfaces confusing 422 errors. Catching them at plan time improves UX.
func (r *ISCSIExtentResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_iscsi_extent")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config ISCSIExtentResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if config.Type.IsNull() || config.Type.IsUnknown() {
		return
	}
	extentType := config.Type.ValueString()

	diskSet := !config.Disk.IsNull() && !config.Disk.IsUnknown() && config.Disk.ValueString() != ""
	pathSet := !config.Path.IsNull() && !config.Path.IsUnknown() && config.Path.ValueString() != ""
	filesizeSet := !config.Filesize.IsNull() && !config.Filesize.IsUnknown() && config.Filesize.ValueInt64() > 0

	switch extentType {
	case "DISK":
		if !diskSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("disk"),
				"Missing disk",
				"type=DISK requires the `disk` attribute to be set to a zvol path (e.g. zvol/tank/iscsi1).",
			)
		}
	case "FILE":
		if !pathSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("path"),
				"Missing path",
				"type=FILE requires the `path` attribute to be set to a file path inside a TrueNAS dataset.",
			)
		}
		if !filesizeSet {
			resp.Diagnostics.AddAttributeError(
				path.Root("filesize"),
				"Missing filesize",
				"type=FILE requires `filesize` to be set to a positive number of bytes.",
			)
		}
	}
}

func (r *ISCSIExtentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("iSCSI extent ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ISCSIExtentResource) mapResponseToModel(extent *client.ISCSIExtent, model *ISCSIExtentResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(extent.ID))
	model.Name = types.StringValue(extent.Name)
	model.Type = types.StringValue(extent.Type)

	diskVal := extent.GetDisk()
	if diskVal == "" {
		model.Disk = types.StringNull()
	} else {
		model.Disk = types.StringValue(diskVal)
	}

	if extent.Path != "" {
		model.Path = types.StringValue(extent.Path)
	} else {
		model.Path = types.StringNull()
	}

	model.Filesize = types.Int64Value(extent.GetFilesize())
	model.Blocksize = types.Int64Value(int64(extent.Blocksize))
	model.RPM = types.StringValue(extent.RPM)
	model.Enabled = types.BoolValue(extent.Enabled)
	model.Comment = types.StringValue(extent.Comment)
	model.ReadOnly = types.BoolValue(extent.ReadOnly)
}
