package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &PrivilegeResource{}
	_ resource.ResourceWithImportState = &PrivilegeResource{}
)

// PrivilegeResource manages a TrueNAS privilege (RBAC grant).
type PrivilegeResource struct {
	client *client.Client
}

// PrivilegeResourceModel describes the resource data model.
type PrivilegeResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Name        types.String   `tfsdk:"name"`
	LocalGroups types.List     `tfsdk:"local_groups"`
	DSGroups    types.List     `tfsdk:"ds_groups"`
	Roles       types.List     `tfsdk:"roles"`
	WebShell    types.Bool     `tfsdk:"web_shell"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func NewPrivilegeResource() resource.Resource {
	return &PrivilegeResource{}
}

func (r *PrivilegeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privilege"
}

func (r *PrivilegeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyStringList := types.ListValueMust(types.StringType, nil)
	emptyInt64List := types.ListValueMust(types.Int64Type, nil)

	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS RBAC privilege — a named grant of roles to one or " +
			"more local and/or directory-service groups.",
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
				Description: "Numeric privilege ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Display name of the privilege (must be unique).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"local_groups": schema.ListAttribute{
				Description: "List of local group GIDs granted by this privilege.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Default:     listdefault.StaticValue(emptyInt64List),
			},
			"ds_groups": schema.ListAttribute{
				Description: "List of directory-service group identifiers (GIDs or SID strings) granted by this privilege.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(emptyStringList),
			},
			"roles": schema.ListAttribute{
				Description: "List of role names included in this privilege (e.g. READONLY_ADMIN, FULL_ADMIN).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(emptyStringList),
			},
			"web_shell": schema.BoolAttribute{
				Description: "Whether holders of this privilege may access the TrueNAS web shell.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *PrivilegeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PrivilegeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Privilege start")

	var plan PrivilegeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	localGroups := privilegeListToIntSlice(ctx, plan.LocalGroups, &resp.Diagnostics)
	dsGroups := privilegeListToDSGroupSlice(ctx, plan.DSGroups, &resp.Diagnostics)
	roles := privilegeListToStringSlice(ctx, plan.Roles, &resp.Diagnostics)

	createReq := &client.PrivilegeCreateRequest{
		Name:        plan.Name.ValueString(),
		LocalGroups: localGroups,
		DSGroups:    dsGroups,
		Roles:       roles,
		WebShell:    plan.WebShell.ValueBool(),
	}

	tflog.Debug(ctx, "Creating privilege", map[string]interface{}{"name": createReq.Name})

	p, err := r.client.CreatePrivilege(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Privilege",
			fmt.Sprintf("Could not create privilege %q: %s", createReq.Name, err),
		)
		return
	}

	r.mapResponseToModel(ctx, p, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create Privilege success")
}

func (r *PrivilegeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Privilege start")

	var state PrivilegeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse privilege ID: %s", err))
		return
	}

	p, err := r.client.GetPrivilege(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Privilege",
			fmt.Sprintf("Could not read privilege %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, p, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read Privilege success")
}

func (r *PrivilegeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Privilege start")

	var plan PrivilegeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state PrivilegeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse privilege ID: %s", err))
		return
	}

	localGroups := privilegeListToIntSlice(ctx, plan.LocalGroups, &resp.Diagnostics)
	dsGroups := privilegeListToDSGroupSlice(ctx, plan.DSGroups, &resp.Diagnostics)
	roles := privilegeListToStringSlice(ctx, plan.Roles, &resp.Diagnostics)

	name := plan.Name.ValueString()
	webShell := plan.WebShell.ValueBool()
	updateReq := &client.PrivilegeUpdateRequest{
		Name:        &name,
		LocalGroups: &localGroups,
		DSGroups:    &dsGroups,
		Roles:       &roles,
		WebShell:    &webShell,
	}

	p, err := r.client.UpdatePrivilege(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Privilege",
			fmt.Sprintf("Could not update privilege %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, p, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update Privilege success")
}

func (r *PrivilegeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Privilege start")

	var state PrivilegeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse privilege ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting privilege", map[string]interface{}{"id": id})
	if err := r.client.DeletePrivilege(ctx, id); err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Privilege already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Privilege",
			fmt.Sprintf("Could not delete privilege %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Privilege success")
}

func (r *PrivilegeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Privilege ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PrivilegeResource) mapResponseToModel(ctx context.Context, p *client.Privilege, model *PrivilegeResourceModel, diags *diag.Diagnostics) {
	model.ID = types.StringValue(strconv.Itoa(p.ID))
	model.Name = types.StringValue(p.Name)
	model.WebShell = types.BoolValue(p.WebShell)

	gids := p.LocalGroupGIDs()
	int64s := make([]int64, len(gids))
	for i, v := range gids {
		int64s[i] = int64(v)
	}
	lgList, d := types.ListValueFrom(ctx, types.Int64Type, int64s)
	diags.Append(d...)
	model.LocalGroups = lgList

	dsList, d := types.ListValueFrom(ctx, types.StringType, p.DSGroupStrings())
	diags.Append(d...)
	model.DSGroups = dsList

	rolesList, d := types.ListValueFrom(ctx, types.StringType, p.Roles)
	diags.Append(d...)
	model.Roles = rolesList
}

// privilegeListToIntSlice converts a types.List of Int64 to []int.
func privilegeListToIntSlice(ctx context.Context, list types.List, diags *diag.Diagnostics) []int {
	if list.IsNull() || list.IsUnknown() {
		return []int{}
	}
	var ints []int64
	diags.Append(list.ElementsAs(ctx, &ints, false)...)
	out := make([]int, len(ints))
	for i, v := range ints {
		out[i] = int(v)
	}
	return out
}

// privilegeListToStringSlice converts a types.List of String to []string.
func privilegeListToStringSlice(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return []string{}
	}
	var ss []string
	diags.Append(list.ElementsAs(ctx, &ss, false)...)
	return ss
}

// privilegeListToDSGroupSlice converts a types.List of Strings to a
// []interface{} where each entry is either an int (if the string parses)
// or the original string. The TrueNAS API accepts both.
func privilegeListToDSGroupSlice(ctx context.Context, list types.List, diags *diag.Diagnostics) []interface{} {
	strs := privilegeListToStringSlice(ctx, list, diags)
	out := make([]interface{}, len(strs))
	for i, s := range strs {
		if n, err := strconv.Atoi(s); err == nil {
			out[i] = n
		} else {
			out[i] = s
		}
	}
	return out
}
