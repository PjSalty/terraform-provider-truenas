package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &ISCSITargetResource{}
	_ resource.ResourceWithImportState = &ISCSITargetResource{}
	_ resource.ResourceWithModifyPlan  = &ISCSITargetResource{}
)

// ISCSITargetResource manages an iSCSI target.
type ISCSITargetResource struct {
	client *client.Client
}

// ISCSITargetResourceModel describes the resource data model.
type ISCSITargetResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Alias    types.String `tfsdk:"alias"`
	Mode     types.String `tfsdk:"mode"`
	Groups   types.List   `tfsdk:"groups"`
	Timeouts timeouts.

		// ISCSITargetGroupModel describes a target group in the data model.
		Value `tfsdk:"timeouts"`
}

type ISCSITargetGroupModel struct {
	Portal     types.Int64  `tfsdk:"portal"`
	Initiator  types.Int64  `tfsdk:"initiator"`
	AuthMethod types.String `tfsdk:"auth_method"`
	Auth       types.Int64  `tfsdk:"auth"`
}

func NewISCSITargetResource() resource.Resource {
	return &ISCSITargetResource{}
}

func (r *ISCSITargetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_target"
}

var iscsiTargetGroupAttrTypes = map[string]attr.Type{
	"portal":      types.Int64Type,
	"initiator":   types.Int64Type,
	"auth_method": types.StringType,
	"auth":        types.Int64Type,
}

func (r *ISCSITargetResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an iSCSI target on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the iSCSI target.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The iSCSI target name (IQN suffix). Lowercase alphanumeric with dots, hyphens, colons.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9][a-z0-9.\-:]*$`),
						"iSCSI target name must be lowercase alphanumeric, may contain dots, hyphens, and colons",
					),
				},
			},
			"alias": schema.StringAttribute{
				Description: "An optional alias for the target.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 120),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "The target mode (ISCSI, FC, BOTH).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ISCSI"),
				Validators: []validator.String{
					stringvalidator.OneOf("ISCSI", "FC", "BOTH"),
				},
			},
			"groups": schema.ListNestedAttribute{
				Description: "Target groups linking portals and initiators.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"portal": schema.Int64Attribute{
							Description: "Portal ID.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"initiator": schema.Int64Attribute{
							Description: "Initiator group ID.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"auth_method": schema.StringAttribute{
							Description: "Authentication method (NONE, CHAP, CHAP_MUTUAL).",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("NONE"),
							Validators: []validator.String{
								stringvalidator.OneOf("NONE", "CHAP", "CHAP_MUTUAL"),
							},
						},
						"auth": schema.Int64Attribute{
							Description: "Auth group ID (0 for none).",
							Optional:    true,
							Computed:    true,
							Validators: []validator.Int64{
								int64validator.AtLeast(0),
							},
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *ISCSITargetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ISCSITargetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSITarget start")

	var plan ISCSITargetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ISCSITargetCreateRequest{
		Name: plan.Name.ValueString(),
		Mode: plan.Mode.ValueString(),
	}

	if !plan.Alias.IsNull() {
		createReq.Alias = plan.Alias.ValueString()
	}

	if !plan.Groups.IsNull() {
		var groups []ISCSITargetGroupModel
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
		for _, g := range groups {
			createReq.Groups = append(createReq.Groups, client.ISCSITargetGroup{
				Portal:     int(g.Portal.ValueInt64()),
				Initiator:  int(g.Initiator.ValueInt64()),
				AuthMethod: g.AuthMethod.ValueString(),
				Auth:       int(g.Auth.ValueInt64()),
			})
		}
	}

	tflog.Debug(ctx, "Creating iSCSI target", map[string]interface{}{"name": plan.Name.ValueString()})

	target, err := r.client.CreateISCSITarget(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating iSCSI Target",
			fmt.Sprintf("Could not create iSCSI target %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, target, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ISCSITarget success")
}

func (r *ISCSITargetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSITarget start")

	var state ISCSITargetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI target ID: %s", err))
		return
	}

	target, err := r.client.GetISCSITarget(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Target",
			fmt.Sprintf("Could not read iSCSI target %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, target, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ISCSITarget success")
}

func (r *ISCSITargetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSITarget start")

	var plan ISCSITargetResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ISCSITargetResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI target ID: %s", err))
		return
	}

	updateReq := &client.ISCSITargetUpdateRequest{
		Name: plan.Name.ValueString(),
		Mode: plan.Mode.ValueString(),
	}

	if !plan.Alias.IsNull() {
		updateReq.Alias = plan.Alias.ValueString()
	}

	if !plan.Groups.IsNull() {
		var groups []ISCSITargetGroupModel
		resp.Diagnostics.Append(plan.Groups.ElementsAs(ctx, &groups, false)...)
		for _, g := range groups {
			updateReq.Groups = append(updateReq.Groups, client.ISCSITargetGroup{
				Portal:     int(g.Portal.ValueInt64()),
				Initiator:  int(g.Initiator.ValueInt64()),
				AuthMethod: g.AuthMethod.ValueString(),
				Auth:       int(g.Auth.ValueInt64()),
			})
		}
	}

	target, err := r.client.UpdateISCSITarget(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating iSCSI Target",
			fmt.Sprintf("Could not update iSCSI target %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, target, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ISCSITarget success")
}

func (r *ISCSITargetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSITarget start")

	var state ISCSITargetResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI target ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting iSCSI target", map[string]interface{}{"id": id})

	err = r.client.DeleteISCSITarget(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "iSCSI target already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting iSCSI Target",
			fmt.Sprintf("Could not delete iSCSI target %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ISCSITarget success")
}

// ModifyPlan enforces iSCSI target cross-attribute constraints at plan time.
//
// Groups with CHAP authentication must reference an auth group ID > 0:
//
//   - auth_method=CHAP or CHAP_MUTUAL requires auth > 0 (auth=0 means "no
//     auth group" which is incompatible with requesting CHAP).
func (r *ISCSITargetResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_iscsi_target")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config ISCSITargetResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if config.Groups.IsNull() || config.Groups.IsUnknown() {
		return
	}

	var groups []ISCSITargetGroupModel
	resp.Diagnostics.Append(config.Groups.ElementsAs(ctx, &groups, false)...)

	for i, g := range groups {
		method := g.AuthMethod.ValueString()
		if method != "CHAP" && method != "CHAP_MUTUAL" {
			continue
		}
		if g.Auth.IsNull() || g.Auth.IsUnknown() || g.Auth.ValueInt64() <= 0 {
			resp.Diagnostics.AddAttributeError(
				path.Root("groups").AtListIndex(i).AtName("auth"),
				"Missing auth group",
				fmt.Sprintf("groups[%d] uses auth_method=%s but auth is unset or 0. "+
					"CHAP authentication requires referencing a configured iSCSI auth group ID.", i, method),
			)
		}
	}
}

func (r *ISCSITargetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("iSCSI target ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ISCSITargetResource) mapResponseToModel(_ context.Context, target *client.ISCSITarget, model *ISCSITargetResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(target.ID))
	model.Name = types.StringValue(target.Name)
	model.Alias = types.StringValue(target.Alias)
	model.Mode = types.StringValue(target.Mode)

	if len(target.Groups) > 0 {
		groupObjects := make([]attr.Value, 0, len(target.Groups))
		for _, g := range target.Groups {
			obj, diags := types.ObjectValue(iscsiTargetGroupAttrTypes, map[string]attr.Value{
				"portal":      types.Int64Value(int64(g.Portal)),
				"initiator":   types.Int64Value(int64(g.Initiator)),
				"auth_method": types.StringValue(g.AuthMethod),
				"auth":        types.Int64Value(int64(g.Auth)),
			})
			if !diags.HasError() {
				groupObjects = append(groupObjects, obj)
			}
		}
		groupList, diags := types.ListValue(types.ObjectType{AttrTypes: iscsiTargetGroupAttrTypes}, groupObjects)
		if !diags.HasError() {
			model.Groups = groupList
		}
	} else {
		model.Groups = types.ListNull(types.ObjectType{AttrTypes: iscsiTargetGroupAttrTypes})
	}
}
