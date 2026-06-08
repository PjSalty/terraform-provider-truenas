package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &NVMetHostResource{}
	_ resource.ResourceWithImportState = &NVMetHostResource{}
	_ resource.ResourceWithModifyPlan  = &NVMetHostResource{}
)

// NVMetHostResource manages an NVMe-oF host (initiator NQN).
type NVMetHostResource struct {
	client *client.Client
}

// NVMetHostResourceModel describes the resource data model.
type NVMetHostResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Hostnqn       types.String   `tfsdk:"hostnqn"`
	DhchapKey     types.String   `tfsdk:"dhchap_key"`
	DhchapCtrlKey types.String   `tfsdk:"dhchap_ctrl_key"`
	DhchapDhgroup types.String   `tfsdk:"dhchap_dhgroup"`
	DhchapHash    types.String   `tfsdk:"dhchap_hash"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewNVMetHostResource() resource.Resource {
	return &NVMetHostResource{}
}

func (r *NVMetHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nvmet_host"
}

func (r *NVMetHostResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an NVMe-oF host (initiator NQN) on TrueNAS SCALE.",
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
				Description: "The numeric ID of the NVMe-oF host.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hostnqn": schema.StringAttribute{
				Description: "NQN of the host that will connect to this TrueNAS.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 223),
					stringvalidator.RegexMatches(regexp.MustCompile(`^nqn\.`), "must start with 'nqn.'"),
				},
			},
			"dhchap_key": schema.StringAttribute{
				Description: "Secret the host must present when connecting.",
				Optional:    true,
				Sensitive:   true,
			},
			"dhchap_ctrl_key": schema.StringAttribute{
				Description: "Secret TrueNAS will present to the host (bi-directional auth).",
				Optional:    true,
				Sensitive:   true,
			},
			"dhchap_dhgroup": schema.StringAttribute{
				Description: "Diffie-Hellman group used on top of CHAP (2048-BIT, 3072-BIT, 4096-BIT, 6144-BIT, 8192-BIT).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("2048-BIT", "3072-BIT", "4096-BIT", "6144-BIT", "8192-BIT"),
				},
			},
			"dhchap_hash": schema.StringAttribute{
				Description: "HMAC hash used for CHAP (SHA-256, SHA-384, SHA-512).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("SHA-256", "SHA-384", "SHA-512"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *NVMetHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NVMetHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NVMetHost start")

	var plan NVMetHostResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NVMetHostCreateRequest{
		Hostnqn: plan.Hostnqn.ValueString(),
	}
	if !plan.DhchapKey.IsNull() && !plan.DhchapKey.IsUnknown() {
		v := plan.DhchapKey.ValueString()
		createReq.DhchapKey = &v
	}
	if !plan.DhchapCtrlKey.IsNull() && !plan.DhchapCtrlKey.IsUnknown() {
		v := plan.DhchapCtrlKey.ValueString()
		createReq.DhchapCtrlKey = &v
	}
	if !plan.DhchapDhgroup.IsNull() && !plan.DhchapDhgroup.IsUnknown() {
		v := plan.DhchapDhgroup.ValueString()
		createReq.DhchapDhgroup = &v
	}
	if !plan.DhchapHash.IsNull() && !plan.DhchapHash.IsUnknown() {
		v := plan.DhchapHash.ValueString()
		createReq.DhchapHash = &v
	}

	tflog.Debug(ctx, "Creating nvmet_host", map[string]interface{}{"hostnqn": createReq.Hostnqn})

	host, err := r.client.CreateNVMetHost(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NVMe-oF Host",
			fmt.Sprintf("Could not create NVMe-oF host: %s", err),
		)
		return
	}

	r.mapResponseToModel(host, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NVMetHost success")
}

func (r *NVMetHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NVMetHost start")

	var state NVMetHostResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF host ID: %s", err))
		return
	}

	host, err := r.client.GetNVMetHost(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NVMe-oF Host",
			fmt.Sprintf("Could not read NVMe-oF host %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(host, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NVMetHost success")
}

func (r *NVMetHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NVMetHost start")

	var plan NVMetHostResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NVMetHostResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF host ID: %s", err))
		return
	}

	updateReq := &client.NVMetHostUpdateRequest{}
	if !plan.Hostnqn.IsNull() && !plan.Hostnqn.IsUnknown() {
		v := plan.Hostnqn.ValueString()
		updateReq.Hostnqn = &v
	}
	if !plan.DhchapKey.IsNull() && !plan.DhchapKey.IsUnknown() {
		v := plan.DhchapKey.ValueString()
		updateReq.DhchapKey = &v
	}
	if !plan.DhchapCtrlKey.IsNull() && !plan.DhchapCtrlKey.IsUnknown() {
		v := plan.DhchapCtrlKey.ValueString()
		updateReq.DhchapCtrlKey = &v
	}
	if !plan.DhchapDhgroup.IsNull() && !plan.DhchapDhgroup.IsUnknown() {
		v := plan.DhchapDhgroup.ValueString()
		updateReq.DhchapDhgroup = &v
	}
	if !plan.DhchapHash.IsNull() && !plan.DhchapHash.IsUnknown() {
		v := plan.DhchapHash.ValueString()
		updateReq.DhchapHash = &v
	}

	host, err := r.client.UpdateNVMetHost(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NVMe-oF Host",
			fmt.Sprintf("Could not update NVMe-oF host %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(host, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NVMetHost success")
}

func (r *NVMetHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NVMetHost start")

	var state NVMetHostResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NVMe-oF host ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting nvmet_host", map[string]interface{}{"id": id})

	if err := r.client.DeleteNVMetHost(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NVMe-oF host already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NVMe-oF Host",
			fmt.Sprintf("Could not delete NVMe-oF host %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NVMetHost success")
}

// ModifyPlan enforces cross-attribute constraints at plan time that the
// TrueNAS API would otherwise only surface at apply time:
//
//   - If `dhchap_key` is set (host-side CHAP secret), `dhchap_hash` must also
//     be set, because CHAP requires agreeing on a hash algorithm.
//   - If `dhchap_ctrl_key` is set (controller-side CHAP secret, bidirectional
//     auth), `dhchap_key` must also be set — bidirectional CHAP implies
//     unidirectional CHAP is already configured.
func (r *NVMetHostResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_nvmet_host")
	// Skip the rest on delete (no plan).
	if req.Plan.Raw.IsNull() {
		return
	}

	var config NVMetHostResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	keySet := !config.DhchapKey.IsNull() && !config.DhchapKey.IsUnknown() && config.DhchapKey.ValueString() != ""
	hashSet := !config.DhchapHash.IsNull() && !config.DhchapHash.IsUnknown() && config.DhchapHash.ValueString() != ""
	ctrlKeySet := !config.DhchapCtrlKey.IsNull() && !config.DhchapCtrlKey.IsUnknown() && config.DhchapCtrlKey.ValueString() != ""

	if keySet && !hashSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("dhchap_hash"),
			"Missing dhchap_hash",
			"When dhchap_key is set, dhchap_hash must also be set: CHAP requires "+
				"agreeing on a hash algorithm (SHA-256, SHA-384, or SHA-512).",
		)
	}

	if ctrlKeySet && !keySet {
		resp.Diagnostics.AddAttributeError(
			path.Root("dhchap_ctrl_key"),
			"Missing dhchap_key",
			"dhchap_ctrl_key enables bidirectional CHAP. Bidirectional CHAP "+
				"requires unidirectional CHAP to be configured first: dhchap_key must be set.",
		)
	}
}

func (r *NVMetHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NVMe-oF host ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NVMetHostResource) mapResponseToModel(host *client.NVMetHost, model *NVMetHostResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(host.ID))
	model.Hostnqn = types.StringValue(host.Hostnqn)
	if host.DhchapKey != nil {
		model.DhchapKey = types.StringValue(*host.DhchapKey)
	} else {
		model.DhchapKey = types.StringNull()
	}
	if host.DhchapCtrlKey != nil {
		model.DhchapCtrlKey = types.StringValue(*host.DhchapCtrlKey)
	} else {
		model.DhchapCtrlKey = types.StringNull()
	}
	if host.DhchapDhgroup != nil {
		model.DhchapDhgroup = types.StringValue(*host.DhchapDhgroup)
	} else {
		model.DhchapDhgroup = types.StringNull()
	}
	if host.DhchapHash != nil {
		model.DhchapHash = types.StringValue(*host.DhchapHash)
	} else {
		model.DhchapHash = types.StringNull()
	}
}
