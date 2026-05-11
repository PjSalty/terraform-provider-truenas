package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &ISCSITargetExtentResource{}
	_ resource.ResourceWithImportState = &ISCSITargetExtentResource{}
)

// ISCSITargetExtentResource manages an iSCSI target-to-extent association.
type ISCSITargetExtentResource struct {
	client *client.Client
}

// ISCSITargetExtentResourceModel describes the resource data model.
type ISCSITargetExtentResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Target   types.Int64    `tfsdk:"target"`
	Extent   types.Int64    `tfsdk:"extent"`
	LunID    types.Int64    `tfsdk:"lunid"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewISCSITargetExtentResource() resource.Resource {
	return &ISCSITargetExtentResource{}
}

func (r *ISCSITargetExtentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_targetextent"
}

func (r *ISCSITargetExtentResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an iSCSI target-to-extent association on TrueNAS SCALE. " +
		"This links an iSCSI target to an extent, completing the iSCSI stack.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the target-extent association.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"target": schema.Int64Attribute{
				Description: "The iSCSI target ID.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"extent": schema.Int64Attribute{
				Description: "The iSCSI extent ID.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"lunid": schema.Int64Attribute{
				Description: "The LUN ID (0-1023). If not set, it is auto-assigned by TrueNAS.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(0, 1023),
				},
			},
		},
	}
}

func (r *ISCSITargetExtentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ISCSITargetExtentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSITargetExtent start")

	var plan ISCSITargetExtentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ISCSITargetExtentCreateRequest{
		Target: int(plan.Target.ValueInt64()),
		Extent: int(plan.Extent.ValueInt64()),
	}

	if !plan.LunID.IsNull() && !plan.LunID.IsUnknown() {
		v := int(plan.LunID.ValueInt64())
		createReq.LunID = &v
	}

	tflog.Debug(ctx, "Creating iSCSI target-extent association", map[string]interface{}{
		"target": plan.Target.ValueInt64(),
		"extent": plan.Extent.ValueInt64(),
	})

	te, err := r.client.CreateISCSITargetExtent(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating iSCSI Target-Extent",
			fmt.Sprintf("Could not create iSCSI target-extent association: %s", err),
		)
		return
	}

	r.mapResponseToModel(te, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ISCSITargetExtent success")
}

func (r *ISCSITargetExtentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSITargetExtent start")

	var state ISCSITargetExtentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI target-extent ID: %s", err))
		return
	}

	te, err := r.client.GetISCSITargetExtent(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Target-Extent",
			fmt.Sprintf("Could not read iSCSI target-extent %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(te, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ISCSITargetExtent success")
}

func (r *ISCSITargetExtentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSITargetExtent start")

	var plan ISCSITargetExtentResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ISCSITargetExtentResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI target-extent ID: %s", err))
		return
	}

	updateReq := &client.ISCSITargetExtentUpdateRequest{
		Target: int(plan.Target.ValueInt64()),
		Extent: int(plan.Extent.ValueInt64()),
	}

	if !plan.LunID.IsNull() && !plan.LunID.IsUnknown() {
		v := int(plan.LunID.ValueInt64())
		updateReq.LunID = &v
	}

	te, err := r.client.UpdateISCSITargetExtent(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating iSCSI Target-Extent",
			fmt.Sprintf("Could not update iSCSI target-extent %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(te, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ISCSITargetExtent success")
}

func (r *ISCSITargetExtentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSITargetExtent start")

	var state ISCSITargetExtentResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI target-extent ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting iSCSI target-extent", map[string]interface{}{"id": id})

	err = r.client.DeleteISCSITargetExtent(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "iSCSI target-extent mapping already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting iSCSI Target-Extent",
			fmt.Sprintf("Could not delete iSCSI target-extent %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ISCSITargetExtent success")
}

func (r *ISCSITargetExtentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("iSCSI target-extent ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ISCSITargetExtentResource) mapResponseToModel(te *client.ISCSITargetExtent, model *ISCSITargetExtentResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(te.ID))
	model.Target = types.Int64Value(int64(te.Target))
	model.Extent = types.Int64Value(int64(te.Extent))
	model.LunID = types.Int64Value(int64(te.LunID))
}
