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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
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
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
	_ resource.ResourceWithModifyPlan  = &GroupResource{}
)

// GroupResource manages a TrueNAS local group.
type GroupResource struct {
	client *client.Client
}

// GroupResourceModel describes the resource data model.
type GroupResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	GID          types.Int64    `tfsdk:"gid"`
	SMB          types.Bool     `tfsdk:"smb"`
	SudoCommands types.List     `tfsdk:"sudo_commands"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

func (r *GroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a local group on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the group.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z_][a-z0-9_-]*\$?$`),
						"group name must start with a lowercase letter or underscore and contain only lowercase letters, digits, underscores, or hyphens",
					),
					stringvalidator.LengthBetween(1, 32),
				},
			},
			"gid": schema.Int64Attribute{
				Description: "The GID for the group. If not set, TrueNAS will assign one.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 4294967295),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the group should be mapped to a Samba group.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sudo_commands": schema.ListAttribute{
				Description: "List of sudo commands the group members are allowed to run.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Group start")

	var plan GroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.GroupCreateRequest{
		Name: plan.Name.ValueString(),
		SMB:  plan.SMB.ValueBool(),
	}

	if !plan.GID.IsNull() && !plan.GID.IsUnknown() {
		createReq.GID = int(plan.GID.ValueInt64())
	}

	if !plan.SudoCommands.IsNull() {
		var cmds []string
		resp.Diagnostics.Append(plan.SudoCommands.ElementsAs(ctx, &cmds, false)...)
		createReq.SudoCommands = cmds
	}

	tflog.Debug(ctx, "Creating group", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	group, err := r.client.CreateGroup(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Group",
			fmt.Sprintf("Could not create group %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(group, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Group success")
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Group start")

	var state GroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse group ID: %s", err))
		return
	}

	group, err := r.client.GetGroup(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Group",
			fmt.Sprintf("Could not read group %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(group, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Group success")
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Group start")

	var plan GroupResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse group ID: %s", err))
		return
	}

	smb := plan.SMB.ValueBool()

	updateReq := &client.GroupUpdateRequest{
		Name: plan.Name.ValueString(),
		SMB:  &smb,
	}

	if !plan.SudoCommands.IsNull() {
		var cmds []string
		resp.Diagnostics.Append(plan.SudoCommands.ElementsAs(ctx, &cmds, false)...)
		updateReq.SudoCommands = cmds
	}

	group, err := r.client.UpdateGroup(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Group",
			fmt.Sprintf("Could not update group %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(group, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Group success")
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Group start")

	var state GroupResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse group ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting group", map[string]interface{}{"id": id})

	err = r.client.DeleteGroup(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Group already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Group",
			fmt.Sprintf("Could not delete group %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Group success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *GroupResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_group")
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Group ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *GroupResource) mapResponseToModel(group *client.Group, model *GroupResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(group.ID))
	model.Name = types.StringValue(group.Name)
	model.GID = types.Int64Value(int64(group.GID))
	model.SMB = types.BoolValue(group.SMB)

	cmdValues := make([]attr.Value, len(group.SudoCommands))
	for i, c := range group.SudoCommands {
		cmdValues[i] = types.StringValue(c)
	}
	model.SudoCommands = types.ListValueMust(types.StringType, cmdValues)
}
