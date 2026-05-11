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
	_ resource.Resource                = &NVMetHostSubsysResource{}
	_ resource.ResourceWithImportState = &NVMetHostSubsysResource{}
)

// NVMetHostSubsysResource manages an NVMe-oF host-to-subsystem authorization.
type NVMetHostSubsysResource struct {
	client *client.Client
}

// NVMetHostSubsysResourceModel describes the resource data model.
type NVMetHostSubsysResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	HostID   types.Int64    `tfsdk:"host_id"`
	SubsysID types.Int64    `tfsdk:"subsys_id"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetHostSubsysResource() resource.Resource {
	return &NVMetHostSubsysResource{}
}

func (r *NVMetHostSubsysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_host_subsys"
}

func (r *NVMetHostSubsysResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NVMe-oF host-to-subsystem authorization. " +
			"This grants a host (initiator NQN) access to a subsystem (target). " +
			"Both host_id and subsys_id require replacement if changed.",
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
				Description: "The numeric ID of the host-subsys association.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host_id": schema.Int64Attribute{
				Description: "ID of the NVMe-oF host to authorize.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"subsys_id": schema.Int64Attribute{
				Description: "ID of the NVMe-oF subsystem to grant access to.",
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

func (r *NVMetHostSubsysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetHostSubsysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetHostSubsys start")

	var plan NVMetHostSubsysResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NVMetHostSubsysCreateRequest{
		HostID:   int(plan.HostID.ValueInt64()),
		SubsysID: int(plan.SubsysID.ValueInt64()),
	}

	tflog.Debug(ctx, "Creating nvmet_host_subsys", map[string]interface{}{
		"host_id":   createReq.HostID,
		"subsys_id": createReq.SubsysID,
	})

	hs, err := r.client.CreateNVMetHostSubsys(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NVMe-oF Host-Subsys Association",
			fmt.Sprintf("Could not create NVMe-oF host-subsys association: %s", err),
		)
		return
	}

	r.mapResponseToModel(hs, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetHostSubsys success")
}

func (r *NVMetHostSubsysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetHostSubsys start")

	var state NVMetHostSubsysResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF host-subsys ID: %s", err))
		return
	}

	hs, err := r.client.GetNVMetHostSubsys(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Host-Subsys Association",
			fmt.Sprintf("Could not read NVMe-oF host-subsys association %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(hs, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetHostSubsys success")
}

func (r *NVMetHostSubsysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetHostSubsys start")

	// Both attributes are RequiresReplace, so Update should only be called for
	// computed-only drift. Re-read from state and persist.
	var plan NVMetHostSubsysResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetHostSubsys success")
}

func (r *NVMetHostSubsysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetHostSubsys start")

	var state NVMetHostSubsysResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF host-subsys ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting nvmet_host_subsys", map[string]interface{}{"id": id})

	if err := r.client.DeleteNVMetHostSubsys(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NVMe-oF host-subsys association already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NVMe-oF Host-Subsys Association",
			fmt.Sprintf("Could not delete NVMe-oF host-subsys association %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NVMetHostSubsys success")
}

func (r *NVMetHostSubsysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NVMe-oF host-subsys ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetHostSubsysResource) mapResponseToModel(hs *client.NVMetHostSubsys, model *NVMetHostSubsysResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(hs.ID))
	model.HostID = types.Int64Value(int64(hs.EffectiveHostID()))
	model.SubsysID = types.Int64Value(int64(hs.EffectiveSubsysID()))
}
