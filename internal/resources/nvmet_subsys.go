package resources

import (
	"context"
	"fmt"
	"regexp"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &NVMetSubsysResource{}
	_ resource.ResourceWithImportState = &NVMetSubsysResource{}
)

// NVMetSubsysResource manages an NVMe-oF subsystem (target).
type NVMetSubsysResource struct {
	client *client.Client
}

// NVMetSubsysResourceModel describes the resource data model.
type NVMetSubsysResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	Subnqn       types.String   `tfsdk:"subnqn"`
	Serial       types.String   `tfsdk:"serial"`
	AllowAnyHost types.Bool     `tfsdk:"allow_any_host"`
	PiEnable     types.Bool     `tfsdk:"pi_enable"`
	QidMax       types.Int64    `tfsdk:"qid_max"`
	IeeeOui      types.String   `tfsdk:"ieee_oui"`
	Ana          types.Bool     `tfsdk:"ana"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetSubsysResource() resource.Resource {
	return &NVMetSubsysResource{}
}

func (r *NVMetSubsysResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_subsys"
}

func (r *NVMetSubsysResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NVMe-oF subsystem (target) on TrueNAS SCALE.",
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
				Description: "The numeric ID of the NVMe-oF subsystem.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Human-readable name for the subsystem. If subnqn is not provided, this name is appended to the global basenqn.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"subnqn": schema.StringAttribute{
				Description: "NVMe Qualified Name (NQN) for the subsystem. Auto-generated if not provided.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 223),
					stringvalidator.RegexMatches(regexp.MustCompile(`^nqn\.`), "must start with 'nqn.'"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"serial": schema.StringAttribute{
				Description: "Serial number assigned to the subsystem (computed).",
				Computed:    true,
			},
			"allow_any_host": schema.BoolAttribute{
				Description: "Allow any host to access the storage in this subsystem (no access control).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"pi_enable": schema.BoolAttribute{
				Description: "Enable Protection Information (PI) for data integrity checking.",
				Optional:    true,
			},
			"qid_max": schema.Int64Attribute{
				Description: "Maximum number of queue IDs allowed for this subsystem (1-65535).",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"ieee_oui": schema.StringAttribute{
				Description: "IEEE Organizationally Unique Identifier for the subsystem (6 hex chars).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(32),
					stringvalidator.RegexMatches(regexp.MustCompile(`^([0-9a-fA-F]*)$`), "must be hexadecimal characters only"),
				},
			},
			"ana": schema.BoolAttribute{
				Description: "Per-subsystem override of the global ANA setting. Leave unset to inherit global.",
				Optional:    true,
			},
		},
	}
}

func (r *NVMetSubsysResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetSubsysResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetSubsys start")

	var plan NVMetSubsysResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NVMetSubsysCreateRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Subnqn.IsNull() && !plan.Subnqn.IsUnknown() {
		v := plan.Subnqn.ValueString()
		createReq.Subnqn = &v
	}
	if !plan.AllowAnyHost.IsNull() && !plan.AllowAnyHost.IsUnknown() {
		v := plan.AllowAnyHost.ValueBool()
		createReq.AllowAnyHost = &v
	}
	if !plan.PiEnable.IsNull() && !plan.PiEnable.IsUnknown() {
		v := plan.PiEnable.ValueBool()
		createReq.PiEnable = &v
	}
	if !plan.QidMax.IsNull() && !plan.QidMax.IsUnknown() {
		v := int(plan.QidMax.ValueInt64())
		createReq.QidMax = &v
	}
	if !plan.IeeeOui.IsNull() && !plan.IeeeOui.IsUnknown() {
		v := plan.IeeeOui.ValueString()
		createReq.IeeeOui = &v
	}
	if !plan.Ana.IsNull() && !plan.Ana.IsUnknown() {
		v := plan.Ana.ValueBool()
		createReq.Ana = &v
	}

	tflog.Debug(ctx, "Creating nvmet_subsys", map[string]interface{}{"name": createReq.Name})

	subsys, err := r.client.CreateNVMetSubsys(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NVMe-oF Subsystem",
			fmt.Sprintf("Could not create NVMe-oF subsystem: %s", err),
		)
		return
	}

	r.mapResponseToModel(subsys, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetSubsys success")
}

func (r *NVMetSubsysResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetSubsys start")

	var state NVMetSubsysResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF subsystem ID: %s", err))
		return
	}

	subsys, err := r.client.GetNVMetSubsys(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Subsystem",
			fmt.Sprintf("Could not read NVMe-oF subsystem %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(subsys, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetSubsys success")
}

func (r *NVMetSubsysResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetSubsys start")

	var plan NVMetSubsysResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NVMetSubsysResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF subsystem ID: %s", err))
		return
	}

	updateReq := &client.NVMetSubsysUpdateRequest{}
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.Subnqn.IsNull() && !plan.Subnqn.IsUnknown() {
		v := plan.Subnqn.ValueString()
		updateReq.Subnqn = &v
	}
	if !plan.AllowAnyHost.IsNull() && !plan.AllowAnyHost.IsUnknown() {
		v := plan.AllowAnyHost.ValueBool()
		updateReq.AllowAnyHost = &v
	}
	if !plan.PiEnable.IsNull() && !plan.PiEnable.IsUnknown() {
		v := plan.PiEnable.ValueBool()
		updateReq.PiEnable = &v
	}
	if !plan.QidMax.IsNull() && !plan.QidMax.IsUnknown() {
		v := int(plan.QidMax.ValueInt64())
		updateReq.QidMax = &v
	}
	if !plan.IeeeOui.IsNull() && !plan.IeeeOui.IsUnknown() {
		v := plan.IeeeOui.ValueString()
		updateReq.IeeeOui = &v
	}
	if !plan.Ana.IsNull() && !plan.Ana.IsUnknown() {
		v := plan.Ana.ValueBool()
		updateReq.Ana = &v
	}

	subsys, err := r.client.UpdateNVMetSubsys(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NVMe-oF Subsystem",
			fmt.Sprintf("Could not update NVMe-oF subsystem %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(subsys, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetSubsys success")
}

func (r *NVMetSubsysResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetSubsys start")

	var state NVMetSubsysResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF subsystem ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting nvmet_subsys", map[string]interface{}{"id": id})

	if err := r.client.DeleteNVMetSubsys(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NVMe-oF subsystem already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NVMe-oF Subsystem",
			fmt.Sprintf("Could not delete NVMe-oF subsystem %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NVMetSubsys success")
}

func (r *NVMetSubsysResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NVMe-oF subsystem ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetSubsysResource) mapResponseToModel(subsys *client.NVMetSubsys, model *NVMetSubsysResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(subsys.ID))
	model.Name = types.StringValue(subsys.Name)
	if subsys.Subnqn != nil {
		model.Subnqn = types.StringValue(*subsys.Subnqn)
	} else {
		model.Subnqn = types.StringNull()
	}
	model.Serial = types.StringValue(subsys.Serial)
	model.AllowAnyHost = types.BoolValue(subsys.AllowAnyHost)
	if subsys.PiEnable != nil {
		model.PiEnable = types.BoolValue(*subsys.PiEnable)
	} else {
		model.PiEnable = types.BoolNull()
	}
	if subsys.QidMax != nil {
		model.QidMax = types.Int64Value(int64(*subsys.QidMax))
	} else {
		model.QidMax = types.Int64Null()
	}
	if subsys.IeeeOui != nil {
		model.IeeeOui = types.StringValue(*subsys.IeeeOui)
	} else {
		model.IeeeOui = types.StringNull()
	}
	if subsys.Ana != nil {
		model.Ana = types.BoolValue(*subsys.Ana)
	} else {
		model.Ana = types.BoolNull()
	}
}
