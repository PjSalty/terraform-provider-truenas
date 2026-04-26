package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &ISCSIInitiatorResource{}
	_ resource.ResourceWithImportState = &ISCSIInitiatorResource{}
)

// ISCSIInitiatorResource manages an iSCSI authorized initiator group.
type ISCSIInitiatorResource struct {
	client *client.Client
}

// ISCSIInitiatorResourceModel describes the resource data model.
type ISCSIInitiatorResourceModel struct {
	ID         types.String   `tfsdk:"id"`
	Initiators types.List     `tfsdk:"initiators"`
	Comment    types.String   `tfsdk:"comment"`
	Timeouts   timeouts.Value `tfsdk:"timeouts"`
}

func NewISCSIInitiatorResource() resource.Resource {
	return &ISCSIInitiatorResource{}
}

func (r *ISCSIInitiatorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_initiator"
}

func (r *ISCSIInitiatorResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an iSCSI authorized initiator group on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the initiator group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"initiators": schema.ListAttribute{
				Description: "List of initiator IQNs allowed to connect. Empty list allows all initiators.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment for the initiator group.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 120),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ISCSIInitiatorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ISCSIInitiatorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSIInitiator start")

	var plan ISCSIInitiatorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ISCSIInitiatorCreateRequest{}

	if !plan.Initiators.IsNull() && !plan.Initiators.IsUnknown() {
		var initiators []string
		resp.Diagnostics.Append(plan.Initiators.ElementsAs(ctx, &initiators, false)...)
		createReq.Initiators = initiators
	}

	if !plan.Comment.IsNull() && !plan.Comment.IsUnknown() {
		createReq.Comment = plan.Comment.ValueString()
	}

	tflog.Debug(ctx, "Creating iSCSI initiator group")

	initiator, err := r.client.CreateISCSIInitiator(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating iSCSI Initiator",
			fmt.Sprintf("Could not create iSCSI initiator: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, initiator, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ISCSIInitiator success")
}

func (r *ISCSIInitiatorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSIInitiator start")

	var state ISCSIInitiatorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI initiator ID: %s", err))
		return
	}

	initiator, err := r.client.GetISCSIInitiator(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Initiator",
			fmt.Sprintf("Could not read iSCSI initiator %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, initiator, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ISCSIInitiator success")
}

func (r *ISCSIInitiatorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSIInitiator start")

	var plan ISCSIInitiatorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ISCSIInitiatorResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI initiator ID: %s", err))
		return
	}

	updateReq := &client.ISCSIInitiatorUpdateRequest{}

	if !plan.Initiators.IsNull() {
		var initiators []string
		resp.Diagnostics.Append(plan.Initiators.ElementsAs(ctx, &initiators, false)...)
		updateReq.Initiators = initiators
	}

	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}

	initiator, err := r.client.UpdateISCSIInitiator(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating iSCSI Initiator",
			fmt.Sprintf("Could not update iSCSI initiator %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, initiator, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ISCSIInitiator success")
}

func (r *ISCSIInitiatorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSIInitiator start")

	var state ISCSIInitiatorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI initiator ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting iSCSI initiator", map[string]interface{}{"id": id})

	err = r.client.DeleteISCSIInitiator(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "iSCSI initiator already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting iSCSI Initiator",
			fmt.Sprintf("Could not delete iSCSI initiator %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ISCSIInitiator success")
}

func (r *ISCSIInitiatorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("iSCSI initiator ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ISCSIInitiatorResource) mapResponseToModel(ctx context.Context, initiator *client.ISCSIInitiator, model *ISCSIInitiatorResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(initiator.ID))
	model.Comment = types.StringValue(initiator.Comment)

	initiatorValues, diags := types.ListValueFrom(ctx, types.StringType, initiator.Initiators)
	if !diags.HasError() {
		model.Initiators = initiatorValues
	}
}
