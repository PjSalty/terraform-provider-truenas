package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &NVMetPortSubsysResource{}
	_ resource.ResourceWithImportState = &NVMetPortSubsysResource{}
)

// NVMetPortSubsysResource manages an NVMe-oF port-to-subsystem association.
type NVMetPortSubsysResource struct {
	client *client.Client
}

// NVMetPortSubsysResourceModel describes the resource data model.
type NVMetPortSubsysResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	PortID   types.Int64    `tfsdk:"port_id"`
	SubsysID types.Int64    `tfsdk:"subsys_id"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetPortSubsysResource() resource.Resource {
	return &NVMetPortSubsysResource{}
}

func (r *NVMetPortSubsysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_port_subsys"
}

func (r *NVMetPortSubsysResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NVMe-oF port-to-subsystem association. " +
			"This makes a subsystem accessible via a given transport port. " +
			"Both port_id and subsys_id require replacement if changed.",
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
				Description: "The numeric ID of the port-subsys association.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port_id": schema.Int64Attribute{
				Description: "ID of the NVMe-oF port to associate.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"subsys_id": schema.Int64Attribute{
				Description: "ID of the NVMe-oF subsystem to expose on the port.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
		},
	}
}

func (r *NVMetPortSubsysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetPortSubsysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetPortSubsys start")

	var plan NVMetPortSubsysResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NVMetPortSubsysCreateRequest{
		PortID:   int(plan.PortID.ValueInt64()),
		SubsysID: int(plan.SubsysID.ValueInt64()),
	}

	tflog.Debug(ctx, "Creating nvmet_port_subsys", map[string]interface{}{
		"port_id":   createReq.PortID,
		"subsys_id": createReq.SubsysID,
	})

	ps, err := r.client.CreateNVMetPortSubsys(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NVMe-oF Port-Subsys Association",
			fmt.Sprintf("Could not create NVMe-oF port-subsys association: %s", err),
		)
		return
	}

	r.mapResponseToModel(ps, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetPortSubsys success")
}

func (r *NVMetPortSubsysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetPortSubsys start")

	var state NVMetPortSubsysResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF port-subsys ID: %s", err))
		return
	}

	ps, err := r.client.GetNVMetPortSubsys(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Port-Subsys Association",
			fmt.Sprintf("Could not read NVMe-oF port-subsys association %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ps, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetPortSubsys success")
}

func (r *NVMetPortSubsysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetPortSubsys start")

	// Both attributes are RequiresReplace, so Update only fires for computed-only drift.
	var plan NVMetPortSubsysResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetPortSubsys success")
}

func (r *NVMetPortSubsysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetPortSubsys start")

	var state NVMetPortSubsysResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF port-subsys ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting nvmet_port_subsys", map[string]interface{}{"id": id})

	if err := r.client.DeleteNVMetPortSubsys(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NVMe-oF port-subsys association already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NVMe-oF Port-Subsys Association",
			fmt.Sprintf("Could not delete NVMe-oF port-subsys association %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NVMetPortSubsys success")
}

func (r *NVMetPortSubsysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NVMe-oF port-subsys ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetPortSubsysResource) mapResponseToModel(ps *client.NVMetPortSubsys, model *NVMetPortSubsysResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(ps.ID))
	model.PortID = types.Int64Value(int64(ps.EffectivePortID()))
	model.SubsysID = types.Int64Value(int64(ps.EffectiveSubsysID()))
}
