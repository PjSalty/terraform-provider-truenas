package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	_ resource.Resource                = &InitScriptResource{}
	_ resource.ResourceWithImportState = &InitScriptResource{}
	_ resource.ResourceWithModifyPlan  = &InitScriptResource{}
)

// InitScriptResource manages a TrueNAS init/startup script.
type InitScriptResource struct {
	client *client.Client
}

// InitScriptResourceModel describes the resource data model.
type InitScriptResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Type     types.String   `tfsdk:"type"`
	Command  types.String   `tfsdk:"command"`
	Script   types.String   `tfsdk:"script"`
	When     types.String   `tfsdk:"when"`
	Enabled  types.Bool     `tfsdk:"enabled"`
	Timeout  types.Int64    `tfsdk:"timeout"`
	Comment  types.String   `tfsdk:"comment"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewInitScriptResource() resource.Resource {
	return &InitScriptResource{}
}

func (r *InitScriptResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_init_script"
}

func (r *InitScriptResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an init/startup script on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the init script.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Script type: COMMAND or SCRIPT.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("COMMAND", "SCRIPT"),
				},
			},
			"command": schema.StringAttribute{
				Description: "The command to execute (when type is COMMAND).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(4096),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"script": schema.StringAttribute{
				Description: "The script path (when type is SCRIPT).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1023),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"when": schema.StringAttribute{
				Description: "When to run: PREINIT, POSTINIT, or SHUTDOWN.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PREINIT", "POSTINIT", "SHUTDOWN"),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the script is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout in seconds for the script (1-3600).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
				Validators: []validator.Int64{
					int64validator.Between(1, 3600),
				},
			},
			"comment": schema.StringAttribute{
				Description: "A comment for the script.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *InitScriptResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InitScriptResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create InitScript start")

	var plan InitScriptResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.InitScriptCreateRequest{
		Type:    plan.Type.ValueString(),
		When:    plan.When.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
		Timeout: int(plan.Timeout.ValueInt64()),
	}

	if !plan.Command.IsNull() {
		createReq.Command = plan.Command.ValueString()
	}
	if !plan.Script.IsNull() {
		createReq.Script = plan.Script.ValueString()
	}
	if !plan.Comment.IsNull() {
		createReq.Comment = plan.Comment.ValueString()
	}

	tflog.Debug(ctx, "Creating init script", map[string]interface{}{
		"type": plan.Type.ValueString(),
		"when": plan.When.ValueString(),
	})

	script, err := r.client.CreateInitScript(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Init Script",
			fmt.Sprintf("Could not create init script: %s", err),
		)
		return
	}

	r.mapResponseToModel(script, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create InitScript success")
}

func (r *InitScriptResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read InitScript start")

	var state InitScriptResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse init script ID: %s", err))
		return
	}

	script, err := r.client.GetInitScript(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Init Script",
			fmt.Sprintf("Could not read init script %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(script, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read InitScript success")
}

func (r *InitScriptResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update InitScript start")

	var plan InitScriptResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state InitScriptResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse init script ID: %s", err))
		return
	}

	enabled := plan.Enabled.ValueBool()
	updateReq := &client.InitScriptUpdateRequest{
		Type:    plan.Type.ValueString(),
		When:    plan.When.ValueString(),
		Enabled: &enabled,
		Timeout: int(plan.Timeout.ValueInt64()),
	}

	if !plan.Command.IsNull() {
		updateReq.Command = plan.Command.ValueString()
	}
	if !plan.Script.IsNull() {
		updateReq.Script = plan.Script.ValueString()
	}
	if !plan.Comment.IsNull() {
		updateReq.Comment = plan.Comment.ValueString()
	}

	script, err := r.client.UpdateInitScript(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Init Script",
			fmt.Sprintf("Could not update init script %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(script, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update InitScript success")
}

func (r *InitScriptResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete InitScript start")

	var state InitScriptResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse init script ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting init script", map[string]interface{}{"id": id})

	err = r.client.DeleteInitScript(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Init/shutdown script already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Init Script",
			fmt.Sprintf("Could not delete init script %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete InitScript success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *InitScriptResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_init_script")
}

func (r *InitScriptResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Init script ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *InitScriptResource) mapResponseToModel(script *client.InitScript, model *InitScriptResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(script.ID))
	model.Type = types.StringValue(script.Type)
	model.Command = types.StringValue(script.Command)
	model.Script = types.StringValue(script.Script)
	model.When = types.StringValue(script.When)
	model.Enabled = types.BoolValue(script.Enabled)
	model.Timeout = types.Int64Value(int64(script.Timeout))
	model.Comment = types.StringValue(script.Comment)
}
