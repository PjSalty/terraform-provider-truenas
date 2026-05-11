package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &NVMetNamespaceResource{}
	_ resource.ResourceWithImportState = &NVMetNamespaceResource{}
)

// NVMetNamespaceResource manages an NVMe-oF namespace (block device within a subsystem).
type NVMetNamespaceResource struct {
	client *client.Client
}

// NVMetNamespaceResourceModel describes the resource data model.
type NVMetNamespaceResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Nsid       types.Int64    `tfsdk:"nsid"`
	SubsysID   types.Int64    `tfsdk:"subsys_id"`
	DeviceType types.String   `tfsdk:"device_type"`
	DevicePath types.String   `tfsdk:"device_path"`
	Filesize   types.Int64    `tfsdk:"filesize"`
	Enabled    types.Bool     `tfsdk:"enabled"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetNamespaceResource() resource.Resource {
	return &NVMetNamespaceResource{}
}

func (r *NVMetNamespaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_namespace"
}

func (r *NVMetNamespaceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NVMe-oF namespace within a subsystem on TrueNAS SCALE. " +
			"A namespace exposes a ZVOL or file as a block device to connected hosts.",
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
				Description: "The numeric ID of the NVMe-oF namespace.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nsid": schema.Int64Attribute{
				Description: "Namespace ID (NSID), unique within the subsystem. Auto-assigned if not provided.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"subsys_id": schema.Int64Attribute{
				Description: "ID of the NVMe-oF subsystem to contain this namespace.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"device_type": schema.StringAttribute{
				Description: "Type of device backing the namespace: ZVOL or FILE.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("ZVOL", "FILE"),
				},
			},
			"device_path": schema.StringAttribute{
				Description: "Path to the device or file for the namespace.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
				},
			},
			"filesize": schema.Int64Attribute{
				Description: "Size of the backing file in bytes. Only used when device_type is FILE.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "If false, the namespace is not accessible.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NVMetNamespaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetNamespaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetNamespace start")

	var plan NVMetNamespaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NVMetNamespaceCreateRequest{
		DeviceType: plan.DeviceType.ValueString(),
		DevicePath: plan.DevicePath.ValueString(),
		SubsysID:   int(plan.SubsysID.ValueInt64()),
	}
	if !plan.Nsid.IsNull() && !plan.Nsid.IsUnknown() {
		v := int(plan.Nsid.ValueInt64())
		createReq.Nsid = &v
	}
	if !plan.Filesize.IsNull() && !plan.Filesize.IsUnknown() {
		v := plan.Filesize.ValueInt64()
		createReq.Filesize = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		createReq.Enabled = &v
	}

	tflog.Debug(ctx, "Creating nvmet_namespace", map[string]interface{}{
		"subsys_id":   createReq.SubsysID,
		"device_type": createReq.DeviceType,
	})

	ns, err := r.client.CreateNVMetNamespace(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NVMe-oF Namespace",
			fmt.Sprintf("Could not create NVMe-oF namespace: %s", err),
		)
		return
	}

	r.mapResponseToModel(ns, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetNamespace success")
}

func (r *NVMetNamespaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetNamespace start")

	var state NVMetNamespaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF namespace ID: %s", err))
		return
	}

	ns, err := r.client.GetNVMetNamespace(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Namespace",
			fmt.Sprintf("Could not read NVMe-oF namespace %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ns, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetNamespace success")
}

func (r *NVMetNamespaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetNamespace start")

	var plan NVMetNamespaceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NVMetNamespaceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF namespace ID: %s", err))
		return
	}

	updateReq := &client.NVMetNamespaceUpdateRequest{}
	if !plan.Nsid.IsNull() && !plan.Nsid.IsUnknown() {
		v := int(plan.Nsid.ValueInt64())
		updateReq.Nsid = &v
	}
	if !plan.DeviceType.IsNull() && !plan.DeviceType.IsUnknown() {
		v := plan.DeviceType.ValueString()
		updateReq.DeviceType = &v
	}
	if !plan.DevicePath.IsNull() && !plan.DevicePath.IsUnknown() {
		v := plan.DevicePath.ValueString()
		updateReq.DevicePath = &v
	}
	if !plan.Filesize.IsNull() && !plan.Filesize.IsUnknown() {
		v := plan.Filesize.ValueInt64()
		updateReq.Filesize = &v
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		v := plan.Enabled.ValueBool()
		updateReq.Enabled = &v
	}
	if !plan.SubsysID.IsNull() && !plan.SubsysID.IsUnknown() {
		v := int(plan.SubsysID.ValueInt64())
		updateReq.SubsysID = &v
	}

	ns, err := r.client.UpdateNVMetNamespace(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NVMe-oF Namespace",
			fmt.Sprintf("Could not update NVMe-oF namespace %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ns, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetNamespace success")
}

func (r *NVMetNamespaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetNamespace start")

	var state NVMetNamespaceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF namespace ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting nvmet_namespace", map[string]interface{}{"id": id})

	if err := r.client.DeleteNVMetNamespace(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NVMe-oF namespace already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NVMe-oF Namespace",
			fmt.Sprintf("Could not delete NVMe-oF namespace %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NVMetNamespace success")
}

func (r *NVMetNamespaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NVMe-oF namespace ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetNamespaceResource) mapResponseToModel(ns *client.NVMetNamespace, model *NVMetNamespaceResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(ns.ID))
	if ns.Nsid != nil {
		model.Nsid = types.Int64Value(int64(*ns.Nsid))
	} else {
		model.Nsid = types.Int64Null()
	}
	model.SubsysID = types.Int64Value(int64(ns.EffectiveSubsysID()))
	model.DeviceType = types.StringValue(ns.DeviceType)
	model.DevicePath = types.StringValue(ns.DevicePath)
	if ns.Filesize != nil {
		model.Filesize = types.Int64Value(*ns.Filesize)
	} else {
		model.Filesize = types.Int64Null()
	}
	model.Enabled = types.BoolValue(ns.Enabled)
}
