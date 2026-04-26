package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &TunableResource{}
	_ resource.ResourceWithImportState = &TunableResource{}
)

// TunableResource manages a TrueNAS kernel tunable.
type TunableResource struct {
	client *client.Client
}

// TunableResourceModel describes the resource data model.
type TunableResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Type     types.String   `tfsdk:"type"`
	Var      types.String   `tfsdk:"var"`
	Value    types.String   `tfsdk:"value"`
	Comment  types.String   `tfsdk:"comment"`
	Enabled  types.Bool     `tfsdk:"enabled"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewTunableResource() resource.Resource {
	return &TunableResource{}
}

func (r *TunableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tunable"
}

func (r *TunableResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a kernel tunable (sysctl, udev, or ZFS parameter) on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the tunable.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "The type of tunable: SYSCTL, UDEV, or ZFS.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("SYSCTL", "UDEV", "ZFS"),
				},
			},
			"var": schema.StringAttribute{
				Description: "The variable name (e.g., net.ipv4.ip_forward).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"value": schema.StringAttribute{
				Description: "The value to set.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1024),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment describing the tunable.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the tunable is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *TunableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TunableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Tunable start")

	var plan TunableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.TunableCreateRequest{
		Type:    plan.Type.ValueString(),
		Var:     plan.Var.ValueString(),
		Value:   plan.Value.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() {
		createReq.Comment = plan.Comment.ValueString()
	}

	tflog.Debug(ctx, "Creating tunable", map[string]interface{}{
		"var": plan.Var.ValueString(),
	})

	tunable, err := r.client.CreateTunable(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Tunable",
			fmt.Sprintf("Could not create tunable %q: %s", plan.Var.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(tunable, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Tunable success")
}

func (r *TunableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Tunable start")

	var state TunableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse tunable ID: %s", err))
		return
	}

	tunable, err := r.client.GetTunable(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Tunable",
			fmt.Sprintf("Could not read tunable %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(tunable, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Tunable success")
}

func (r *TunableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Tunable start")

	var plan TunableResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state TunableResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse tunable ID: %s", err))
		return
	}

	enabled := plan.Enabled.ValueBool()

	updateReq := &client.TunableUpdateRequest{
		Value:   plan.Value.ValueString(),
		Enabled: &enabled,
	}

	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}

	tunable, err := r.client.UpdateTunable(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Tunable",
			fmt.Sprintf("Could not update tunable %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(tunable, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Tunable success")
}

func (r *TunableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Tunable start")

	var state TunableResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse tunable ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting tunable", map[string]interface{}{"id": id})

	err = r.client.DeleteTunable(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Tunable already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Tunable",
			fmt.Sprintf("Could not delete tunable %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Tunable success")
}

func (r *TunableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Tunable ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *TunableResource) mapResponseToModel(tunable *client.Tunable, model *TunableResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(tunable.ID))
	model.Type = types.StringValue(tunable.Type)
	model.Var = types.StringValue(tunable.Var)
	model.Value = types.StringValue(tunable.Value)
	model.Comment = types.StringValue(tunable.Comment)
	model.Enabled = types.BoolValue(tunable.Enabled)
}
