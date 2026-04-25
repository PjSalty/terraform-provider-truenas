package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	tnvalidators "github.com/PjSalty/terraform-provider-truenas/internal/validators"
)

var (
	_ resource.Resource                = &ISCSIPortalResource{}
	_ resource.ResourceWithImportState = &ISCSIPortalResource{}
	_ resource.ResourceWithModifyPlan  = &ISCSIPortalResource{}
)

// ISCSIPortalResource manages a TrueNAS iSCSI portal.
type ISCSIPortalResource struct {
	client *client.Client
}

// ISCSIPortalResourceModel describes the resource data model.
type ISCSIPortalResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Comment  types.String `tfsdk:"comment"`
	Listen   types.List   `tfsdk:"listen"`
	Tag      types.Int64  `tfsdk:"tag"`
	Timeouts timeouts.

		// ISCSIPortalListenModel describes a portal listen entry.
		Value `tfsdk:"timeouts"`
}

type ISCSIPortalListenModel struct {
	IP   types.String `tfsdk:"ip"`
	Port types.Int64  `tfsdk:"port"`
}

var iscsiPortalListenAttrTypes = map[string]attr.Type{
	"ip":   types.StringType,
	"port": types.Int64Type,
}

func NewISCSIPortalResource() resource.Resource {
	return &ISCSIPortalResource{}
}

func (r *ISCSIPortalResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_portal"
}

func (r *ISCSIPortalResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an iSCSI portal on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the iSCSI portal.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment for the portal.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"listen": schema.ListNestedAttribute{
				Description: "Listen addresses for the portal.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip": schema.StringAttribute{
							Description: "IP address to listen on (0.0.0.0 for all).",
							Required:    true,
							Validators: []validator.String{
								tnvalidators.IPOrCIDR(),
							},
						},
						"port": schema.Int64Attribute{
							Description: "Port reported by the server (always 3260 on " +
								"SCALE 25.10+). On 25.10+ the API no longer accepts " +
								"this field in create/update bodies — the provider " +
								"silently drops any user-supplied value.",
							Optional:           true,
							Computed:           true,
							DeprecationMessage: "The TrueNAS 25.10 API removed the per-listen port field. This attribute is now read-only and will be removed in a future major version.",
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"tag": schema.Int64Attribute{
				Description: "The portal group tag.",
				Computed:    true,
			},
		},
	}
}

func (r *ISCSIPortalResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ISCSIPortalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ISCSIPortal start")

	var plan ISCSIPortalResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ISCSIPortalCreateRequest{}

	if !plan.Comment.IsNull() {
		createReq.Comment = plan.Comment.ValueString()
	}

	var listens []ISCSIPortalListenModel
	resp.Diagnostics.Append(plan.Listen.ElementsAs(ctx, &listens, false)...)
	for _, l := range listens {
		listen := client.ISCSIPortalListen{
			IP: l.IP.ValueString(),
		}
		if !l.Port.IsNull() && !l.Port.IsUnknown() {
			listen.Port = int(l.Port.ValueInt64())
		}
		createReq.Listen = append(createReq.Listen, listen)
	}

	tflog.Debug(ctx, "Creating iSCSI portal")

	portal, err := r.client.CreateISCSIPortal(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating iSCSI Portal",
			fmt.Sprintf("Could not create iSCSI portal: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, portal, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ISCSIPortal success")
}

func (r *ISCSIPortalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ISCSIPortal start")

	var state ISCSIPortalResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI portal ID: %s", err))
		return
	}

	portal, err := r.client.GetISCSIPortal(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Portal",
			fmt.Sprintf("Could not read iSCSI portal %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, portal, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ISCSIPortal success")
}

func (r *ISCSIPortalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ISCSIPortal start")

	var plan ISCSIPortalResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ISCSIPortalResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI portal ID: %s", err))
		return
	}

	updateReq := &client.ISCSIPortalUpdateRequest{}

	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}

	var listens []ISCSIPortalListenModel
	resp.Diagnostics.Append(plan.Listen.ElementsAs(ctx, &listens, false)...)
	for _, l := range listens {
		listen := client.ISCSIPortalListen{
			IP: l.IP.ValueString(),
		}
		if !l.Port.IsNull() && !l.Port.IsUnknown() {
			listen.Port = int(l.Port.ValueInt64())
		}
		updateReq.Listen = append(updateReq.Listen, listen)
	}

	portal, err := r.client.UpdateISCSIPortal(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating iSCSI Portal",
			fmt.Sprintf("Could not update iSCSI portal %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, portal, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ISCSIPortal success")
}

func (r *ISCSIPortalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ISCSIPortal start")

	var state ISCSIPortalResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse iSCSI portal ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting iSCSI portal", map[string]interface{}{"id": id})

	err = r.client.DeleteISCSIPortal(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "iSCSI portal already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting iSCSI Portal",
			fmt.Sprintf("Could not delete iSCSI portal %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ISCSIPortal success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *ISCSIPortalResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_iscsi_portal")
}

func (r *ISCSIPortalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("iSCSI portal ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ISCSIPortalResource) mapResponseToModel(_ context.Context, portal *client.ISCSIPortal, model *ISCSIPortalResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(portal.ID))
	model.Comment = types.StringValue(portal.Comment)
	model.Tag = types.Int64Value(int64(portal.Tag))

	if len(portal.Listen) > 0 {
		listenObjects := make([]attr.Value, 0, len(portal.Listen))
		for _, l := range portal.Listen {
			obj, diags := types.ObjectValue(iscsiPortalListenAttrTypes, map[string]attr.Value{
				"ip":   types.StringValue(l.IP),
				"port": types.Int64Value(int64(l.Port)),
			})
			if !diags.HasError() {
				listenObjects = append(listenObjects, obj)
			}
		}
		listenList, diags := types.ListValue(types.ObjectType{AttrTypes: iscsiPortalListenAttrTypes}, listenObjects)
		if !diags.HasError() {
			model.Listen = listenList
		}
	}
}
