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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	tnvalidators "github.com/PjSalty/terraform-provider-truenas/internal/validators"
)

var (
	_ resource.Resource                = &StaticRouteResource{}
	_ resource.ResourceWithImportState = &StaticRouteResource{}
)

// StaticRouteResource manages a TrueNAS static route.
type StaticRouteResource struct {
	client *client.Client
}

// StaticRouteResourceModel describes the resource data model.
type StaticRouteResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Destination types.String   `tfsdk:"destination"`
	Gateway     types.String   `tfsdk:"gateway"`
	Description types.String   `tfsdk:"description"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func NewStaticRouteResource() resource.Resource {
	return &StaticRouteResource{}
}

func (r *StaticRouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_static_route"
}

func (r *StaticRouteResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a static network route on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the static route.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"destination": schema.StringAttribute{
				Description: "The destination network in CIDR notation (e.g., 192.168.1.0/24).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					tnvalidators.IPOrCIDR(),
				},
			},
			"gateway": schema.StringAttribute{
				Description: "The gateway IP address for the route.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					tnvalidators.IPOrCIDR(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the static route.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
		},
	}
}

func (r *StaticRouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *StaticRouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create StaticRoute start")

	var plan StaticRouteResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.StaticRouteCreateRequest{
		Destination: plan.Destination.ValueString(),
		Gateway:     plan.Gateway.ValueString(),
	}

	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueString()
	}

	tflog.Debug(ctx, "Creating static route", map[string]interface{}{
		"destination": plan.Destination.ValueString(),
		"gateway":     plan.Gateway.ValueString(),
	})

	route, err := r.client.CreateStaticRoute(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Static Route",
			fmt.Sprintf("Could not create static route: %s", err),
		)
		return
	}

	r.mapResponseToModel(route, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create StaticRoute success")
}

func (r *StaticRouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read StaticRoute start")

	var state StaticRouteResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse static route ID %q: %s", state.ID.ValueString(), err))
		return
	}

	route, err := r.client.GetStaticRoute(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Static Route",
			fmt.Sprintf("Could not read static route %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(route, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read StaticRoute success")
}

func (r *StaticRouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update StaticRoute start")

	var plan StaticRouteResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state StaticRouteResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse static route ID: %s", err))
		return
	}

	updateReq := &client.StaticRouteUpdateRequest{
		Destination: plan.Destination.ValueString(),
		Gateway:     plan.Gateway.ValueString(),
	}

	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	route, err := r.client.UpdateStaticRoute(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Static Route",
			fmt.Sprintf("Could not update static route %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(route, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update StaticRoute success")
}

func (r *StaticRouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete StaticRoute start")

	var state StaticRouteResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse static route ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting static route", map[string]interface{}{
		"id": id,
	})

	err = r.client.DeleteStaticRoute(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Static route already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Static Route",
			fmt.Sprintf("Could not delete static route %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete StaticRoute success")
}

func (r *StaticRouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Static route ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *StaticRouteResource) mapResponseToModel(route *client.StaticRoute, model *StaticRouteResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(route.ID))
	model.Destination = types.StringValue(route.Destination)
	model.Gateway = types.StringValue(route.Gateway)
	model.Description = types.StringValue(route.Description)
}
