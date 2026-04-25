package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &NFSShareResource{}
	_ resource.ResourceWithImportState = &NFSShareResource{}
	_ resource.ResourceWithModifyPlan  = &NFSShareResource{}
)

// NFSShareResource manages a TrueNAS NFS share.
type NFSShareResource struct {
	client *client.Client
}

// NFSShareResourceModel describes the resource data model.
type NFSShareResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Path         types.String   `tfsdk:"path"`
	Comment      types.String   `tfsdk:"comment"`
	Hosts        types.List     `tfsdk:"hosts"`
	Networks     types.List     `tfsdk:"networks"`
	ReadOnly     types.Bool     `tfsdk:"readonly"`
	MaprootUser  types.String   `tfsdk:"maproot_user"`
	MaprootGroup types.String   `tfsdk:"maproot_group"`
	MapallUser   types.String   `tfsdk:"mapall_user"`
	MapallGroup  types.String   `tfsdk:"mapall_group"`
	Security     types.List     `tfsdk:"security"`
	Enabled      types.Bool     `tfsdk:"enabled"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func NewNFSShareResource() resource.Resource {
	return &NFSShareResource{}
}

func (r *NFSShareResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_nfs"
}

func (r *NFSShareResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an NFS share on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the NFS share.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path to share (e.g., /mnt/tank/data). Must start with /mnt/.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(5, 1023),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^/mnt/`),
						"NFS share path must start with /mnt/",
					),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment describing the share.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"hosts": schema.ListAttribute{
				Description: "List of allowed hostnames or IP addresses. Empty means all hosts.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"networks": schema.ListAttribute{
				Description: "List of allowed networks in CIDR notation.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"readonly": schema.BoolAttribute{
				Description: "Whether the share is read-only.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"maproot_user": schema.StringAttribute{
				Description: "Map root user to this user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"maproot_group": schema.StringAttribute{
				Description: "Map root group to this group.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mapall_user": schema.StringAttribute{
				Description: "Map all users to this user.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mapall_group": schema.StringAttribute{
				Description: "Map all groups to this group.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"security": schema.ListAttribute{
				Description: "Security mechanisms (SYS, KRB5, KRB5I, KRB5P).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("SYS", "KRB5", "KRB5I", "KRB5P")),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the share is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
	}
}

func (r *NFSShareResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *NFSShareResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create NFSShare start")

	var plan NFSShareResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.NFSShareCreateRequest{
		Path:    plan.Path.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
	}

	if !plan.Comment.IsNull() {
		createReq.Comment = plan.Comment.ValueString()
	}
	if !plan.ReadOnly.IsNull() {
		createReq.ReadOnly = plan.ReadOnly.ValueBool()
	}
	if !plan.MaprootUser.IsNull() {
		createReq.MaprootUser = plan.MaprootUser.ValueString()
	}
	if !plan.MaprootGroup.IsNull() {
		createReq.MaprootGroup = plan.MaprootGroup.ValueString()
	}
	if !plan.MapallUser.IsNull() {
		createReq.MapallUser = plan.MapallUser.ValueString()
	}
	if !plan.MapallGroup.IsNull() {
		createReq.MapallGroup = plan.MapallGroup.ValueString()
	}

	if !plan.Hosts.IsNull() {
		var hosts []string
		resp.Diagnostics.Append(plan.Hosts.ElementsAs(ctx, &hosts, false)...)
		createReq.Hosts = hosts
	}

	if !plan.Networks.IsNull() {
		var networks []string
		resp.Diagnostics.Append(plan.Networks.ElementsAs(ctx, &networks, false)...)
		createReq.Networks = networks
	}

	if !plan.Security.IsNull() {
		var security []string
		resp.Diagnostics.Append(plan.Security.ElementsAs(ctx, &security, false)...)
		createReq.Security = security
	}

	tflog.Debug(ctx, "Creating NFS share", map[string]interface{}{
		"path": plan.Path.ValueString(),
	})

	share, err := r.client.CreateNFSShare(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating NFS Share",
			fmt.Sprintf("Could not create NFS share for path %q: %s", plan.Path.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, share, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create NFSShare success")
}

func (r *NFSShareResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read NFSShare start")

	var state NFSShareResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NFS share ID %q: %s", state.ID.ValueString(), err))
		return
	}

	share, err := r.client.GetNFSShare(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading NFS Share",
			fmt.Sprintf("Could not read NFS share %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, share, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read NFSShare success")
}

func (r *NFSShareResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update NFSShare start")

	var plan NFSShareResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state NFSShareResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NFS share ID: %s", err))
		return
	}

	readOnly := plan.ReadOnly.ValueBool()
	enabled := plan.Enabled.ValueBool()

	updateReq := &client.NFSShareUpdateRequest{
		Path:     plan.Path.ValueString(),
		ReadOnly: &readOnly,
		Enabled:  &enabled,
	}

	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}
	if !plan.MaprootUser.IsNull() {
		updateReq.MaprootUser = plan.MaprootUser.ValueString()
	}
	if !plan.MaprootGroup.IsNull() {
		updateReq.MaprootGroup = plan.MaprootGroup.ValueString()
	}
	if !plan.MapallUser.IsNull() {
		updateReq.MapallUser = plan.MapallUser.ValueString()
	}
	if !plan.MapallGroup.IsNull() {
		updateReq.MapallGroup = plan.MapallGroup.ValueString()
	}

	if !plan.Hosts.IsNull() {
		var hosts []string
		resp.Diagnostics.Append(plan.Hosts.ElementsAs(ctx, &hosts, false)...)
		updateReq.Hosts = hosts
	}

	if !plan.Networks.IsNull() {
		var networks []string
		resp.Diagnostics.Append(plan.Networks.ElementsAs(ctx, &networks, false)...)
		updateReq.Networks = networks
	}

	if !plan.Security.IsNull() {
		var security []string
		resp.Diagnostics.Append(plan.Security.ElementsAs(ctx, &security, false)...)
		updateReq.Security = security
	}

	share, err := r.client.UpdateNFSShare(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating NFS Share",
			fmt.Sprintf("Could not update NFS share %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, share, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update NFSShare success")
}

func (r *NFSShareResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete NFSShare start")

	var state NFSShareResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse NFS share ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting NFS share", map[string]interface{}{
		"id": id,
	})

	err = r.client.DeleteNFSShare(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "NFS share already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting NFS Share",
			fmt.Sprintf("Could not delete NFS share %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete NFSShare success")
}

// ModifyPlan enforces NFS share cross-attribute constraints:
//
//   - `mapall_user` and `mapall_group` are mutually reinforcing: TrueNAS
//     applies the mapping only if BOTH are set. Setting one without the
//     other silently disables the mapping, which is usually a config bug.
//   - Same for `maproot_user` and `maproot_group`.
//   - `mapall_*` and `maproot_*` are mutually exclusive: mapall overrides
//     maproot, so setting both means maproot is dead config.
func (r *NFSShareResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_share_nfs")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config NFSShareResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	mapallUserSet := !config.MapallUser.IsNull() && !config.MapallUser.IsUnknown() && config.MapallUser.ValueString() != ""
	mapallGroupSet := !config.MapallGroup.IsNull() && !config.MapallGroup.IsUnknown() && config.MapallGroup.ValueString() != ""
	maprootUserSet := !config.MaprootUser.IsNull() && !config.MaprootUser.IsUnknown() && config.MaprootUser.ValueString() != ""
	maprootGroupSet := !config.MaprootGroup.IsNull() && !config.MaprootGroup.IsUnknown() && config.MaprootGroup.ValueString() != ""

	if mapallUserSet != mapallGroupSet {
		target := path.Root("mapall_group")
		if mapallGroupSet {
			target = path.Root("mapall_user")
		}
		resp.Diagnostics.AddAttributeError(
			target,
			"Incomplete mapall mapping",
			"mapall_user and mapall_group must be set together. TrueNAS only "+
				"applies the mapping when both are present.",
		)
	}

	if maprootUserSet != maprootGroupSet {
		target := path.Root("maproot_group")
		if maprootGroupSet {
			target = path.Root("maproot_user")
		}
		resp.Diagnostics.AddAttributeError(
			target,
			"Incomplete maproot mapping",
			"maproot_user and maproot_group must be set together. TrueNAS only "+
				"applies the mapping when both are present.",
		)
	}

	if (mapallUserSet || mapallGroupSet) && (maprootUserSet || maprootGroupSet) {
		resp.Diagnostics.AddAttributeError(
			path.Root("maproot_user"),
			"Conflicting mapall and maproot",
			"mapall_* and maproot_* are mutually exclusive: mapall overrides "+
				"maproot, so setting both leaves maproot as dead configuration.",
		)
	}
}

func (r *NFSShareResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("NFS share ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *NFSShareResource) mapResponseToModel(ctx context.Context, share *client.NFSShare, model *NFSShareResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(share.ID))
	model.Path = types.StringValue(share.Path)
	model.Comment = types.StringValue(share.Comment)
	model.ReadOnly = types.BoolValue(share.ReadOnly)
	model.Enabled = types.BoolValue(share.Enabled)
	model.MaprootUser = types.StringValue(share.MaprootUser)
	model.MaprootGroup = types.StringValue(share.MaprootGroup)
	model.MapallUser = types.StringValue(share.MapallUser)
	model.MapallGroup = types.StringValue(share.MapallGroup)

	hostValues, diags := types.ListValueFrom(ctx, types.StringType, share.Hosts)
	if !diags.HasError() {
		model.Hosts = hostValues
	}

	networkValues, diags := types.ListValueFrom(ctx, types.StringType, share.Networks)
	if !diags.HasError() {
		model.Networks = networkValues
	}

	securityValues, diags := types.ListValueFrom(ctx, types.StringType, share.Security)
	if !diags.HasError() {
		model.Security = securityValues
	}
}
