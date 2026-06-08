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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &ISCSIAuthResource{}
	_ resource.ResourceWithImportState = &ISCSIAuthResource{}
	_ resource.ResourceWithModifyPlan  = &ISCSIAuthResource{}
)

// ISCSIAuthResource manages an iSCSI CHAP credential set.
type ISCSIAuthResource struct {
	client *client.Client
}

type ISCSIAuthResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Tag           types.Int64    `tfsdk:"tag"`
	User          types.String   `tfsdk:"user"`
	Secret        types.String   `tfsdk:"secret"`
	Peeruser      types.String   `tfsdk:"peeruser"`
	Peersecret    types.String   `tfsdk:"peersecret"`
	DiscoveryAuth types.String   `tfsdk:"discovery_auth"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewISCSIAuthResource() resource.Resource {
	return &ISCSIAuthResource{}
}

func (r *ISCSIAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_auth"
}

func (r *ISCSIAuthResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an iSCSI CHAP authentication credential set.",
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
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tag": schema.Int64Attribute{
				Description: "Numeric tag used to associate this credential with iSCSI targets (0-65535).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 65535),
				},
			},
			"user": schema.StringAttribute{
				Description: "Username for iSCSI CHAP authentication (1-255 chars).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"secret": schema.StringAttribute{
				Description: "Password/secret for iSCSI CHAP authentication. Must be 12–16 characters per RFC 3720.",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(12, 16),
				},
			},
			"peeruser": schema.StringAttribute{
				Description: "Username for mutual CHAP authentication (optional).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"peersecret": schema.StringAttribute{
				Description: "Password/secret for mutual CHAP authentication. When set, must be 12–16 characters per RFC 3720.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.Any(
						stringvalidator.LengthBetween(0, 0),
						stringvalidator.LengthBetween(12, 16),
					),
				},
			},
			"discovery_auth": schema.StringAttribute{
				Description: "Authentication method for target discovery: NONE, CHAP, CHAP_MUTUAL.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("NONE"),
				Validators: []validator.String{
					stringvalidator.OneOf("NONE", "CHAP", "CHAP_MUTUAL"),
				},
			},
		},
	}
}

func (r *ISCSIAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ISCSIAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSIAuth start")

	var plan ISCSIAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ISCSIAuthCreateRequest{
		Tag:           int(plan.Tag.ValueInt64()),
		User:          plan.User.ValueString(),
		Secret:        plan.Secret.ValueString(),
		Peeruser:      plan.Peeruser.ValueString(),
		Peersecret:    plan.Peersecret.ValueString(),
		DiscoveryAuth: plan.DiscoveryAuth.ValueString(),
	}

	tflog.Debug(ctx, "Creating iSCSI auth", map[string]interface{}{"tag": createReq.Tag})

	a, err := r.client.CreateISCSIAuth(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating iSCSI Auth", fmt.Sprintf("Could not create iSCSI auth: %s", err))
		return
	}

	r.mapResponseToModel(a, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ISCSIAuth success")
}

func (r *ISCSIAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSIAuth start")

	var state ISCSIAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI auth ID: %s", err))
		return
	}

	a, err := r.client.GetISCSIAuth(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading iSCSI Auth", fmt.Sprintf("Could not read iSCSI auth %d: %s", id, err))
		return
	}

	r.mapResponseToModel(a, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ISCSIAuth success")
}

func (r *ISCSIAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSIAuth start")

	var plan ISCSIAuthResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ISCSIAuthResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI auth ID: %s", err))
		return
	}

	tag := int(plan.Tag.ValueInt64())
	user := plan.User.ValueString()
	secret := plan.Secret.ValueString()
	peeruser := plan.Peeruser.ValueString()
	peersecret := plan.Peersecret.ValueString()
	discoveryAuth := plan.DiscoveryAuth.ValueString()

	updateReq := &client.ISCSIAuthUpdateRequest{
		Tag:           &tag,
		User:          &user,
		Secret:        &secret,
		Peeruser:      &peeruser,
		Peersecret:    &peersecret,
		DiscoveryAuth: &discoveryAuth,
	}

	a, err := r.client.UpdateISCSIAuth(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating iSCSI Auth", fmt.Sprintf("Could not update iSCSI auth %d: %s", id, err))
		return
	}

	r.mapResponseToModel(a, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ISCSIAuth success")
}

func (r *ISCSIAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSIAuth start")

	var state ISCSIAuthResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI auth ID: %s", err))
		return
	}

	if err := r.client.DeleteISCSIAuth(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "iSCSI auth already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting iSCSI Auth", fmt.Sprintf("Could not delete iSCSI auth %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete ISCSIAuth success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *ISCSIAuthResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_iscsi_auth")
}

func (r *ISCSIAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("iSCSI auth ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("secret"), types.StringValue(""))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("peersecret"), types.StringValue(""))...)
}

func (r *ISCSIAuthResource) mapResponseToModel(a *client.ISCSIAuth, model *ISCSIAuthResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(a.ID))
	model.Tag = types.Int64Value(int64(a.Tag))
	model.User = types.StringValue(a.User)
	// Preserve user-supplied secrets (API redacts them).
	if model.Secret.IsNull() || model.Secret.IsUnknown() {
		model.Secret = types.StringValue("")
	}
	model.Peeruser = types.StringValue(a.Peeruser)
	if model.Peersecret.IsNull() || model.Peersecret.IsUnknown() {
		model.Peersecret = types.StringValue("")
	}
	if a.DiscoveryAuth != "" {
		model.DiscoveryAuth = types.StringValue(a.DiscoveryAuth)
	} else {
		model.DiscoveryAuth = types.StringValue("NONE")
	}
}
