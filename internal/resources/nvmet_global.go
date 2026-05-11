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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &NVMetGlobalResource{}
	_ resource.ResourceWithImportState = &NVMetGlobalResource{}
)

// NVMetGlobalResource manages the TrueNAS NVMe-oF global (singleton) config.
type NVMetGlobalResource struct {
	client *client.Client
}

// NVMetGlobalResourceModel describes the resource data model.
type NVMetGlobalResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Basenqn       types.String   `tfsdk:"basenqn"`
	Kernel        types.Bool     `tfsdk:"kernel"`
	Ana           types.Bool     `tfsdk:"ana"`
	Rdma          types.Bool     `tfsdk:"rdma"`
	XportReferral types.Bool     `tfsdk:"xport_referral"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetGlobalResource() resource.Resource {
	return &NVMetGlobalResource{}
}

func (r *NVMetGlobalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_global"
}

func (r *NVMetGlobalResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the NVMe-oF global configuration on TrueNAS SCALE. " +
			"This is a singleton resource — only one instance can exist. Delete " +
			"does not reset the remote configuration; it only removes the resource " +
			"from Terraform state.",
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
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"basenqn": schema.StringAttribute{
				Description: "NQN used as the prefix when creating subsystems without an explicit subnqn.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 223),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"kernel": schema.BoolAttribute{
				Description: "Use the kernel NVMe-oF backend.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"ana": schema.BoolAttribute{
				Description: "Enable Asymmetric Namespace Access (ANA).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"rdma": schema.BoolAttribute{
				Description: "Enable RDMA transport (Enterprise + RDMA-capable hardware only).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"xport_referral": schema.BoolAttribute{
				Description: "Generate cross-port referrals for ports on this TrueNAS.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NVMetGlobalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetGlobalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetGlobal start")

	var plan NVMetGlobalResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating nvmet_global (updating singleton)")

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateNVMetGlobal(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NVMe-oF Global",
			fmt.Sprintf("Could not update NVMe-oF global configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetGlobal success")
}

func (r *NVMetGlobalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetGlobal start")

	var state NVMetGlobalResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetNVMetGlobal(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Global",
			fmt.Sprintf("Could not read NVMe-oF global configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetGlobal success")
}

func (r *NVMetGlobalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetGlobal start")

	var plan NVMetGlobalResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateNVMetGlobal(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NVMe-oF Global",
			fmt.Sprintf("Could not update NVMe-oF global configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetGlobal success")
}

func (r *NVMetGlobalResource) Delete(ctx context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetGlobal start")

	// Singleton: there is no API to delete the global config. Removing the
	// resource from Terraform state is the only action.
	tflog.Debug(ctx, "Deleting nvmet_global resource (state-only removal)")
	tflog.Trace(ctx, "Delete NVMetGlobal success")
}

func (r *NVMetGlobalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetGlobalResource) buildUpdateRequest(plan *NVMetGlobalResourceModel) *client.NVMetGlobalUpdateRequest {
	updateReq := &client.NVMetGlobalUpdateRequest{}
	if !plan.Basenqn.IsNull() && !plan.Basenqn.IsUnknown() {
		v := plan.Basenqn.ValueString()
		updateReq.Basenqn = &v
	}
	if !plan.Kernel.IsNull() && !plan.Kernel.IsUnknown() {
		v := plan.Kernel.ValueBool()
		updateReq.Kernel = &v
	}
	if !plan.Ana.IsNull() && !plan.Ana.IsUnknown() {
		v := plan.Ana.ValueBool()
		updateReq.Ana = &v
	}
	if !plan.Rdma.IsNull() && !plan.Rdma.IsUnknown() {
		v := plan.Rdma.ValueBool()
		updateReq.Rdma = &v
	}
	if !plan.XportReferral.IsNull() && !plan.XportReferral.IsUnknown() {
		v := plan.XportReferral.ValueBool()
		updateReq.XportReferral = &v
	}
	return updateReq
}

func (r *NVMetGlobalResource) mapResponseToModel(config *client.NVMetGlobal, model *NVMetGlobalResourceModel) {
	model.ID = types.StringValue("1")
	model.Basenqn = types.StringValue(config.Basenqn)
	model.Kernel = types.BoolValue(config.Kernel)
	model.Ana = types.BoolValue(config.Ana)
	model.Rdma = types.BoolValue(config.Rdma)
	model.XportReferral = types.BoolValue(config.XportReferral)
}
