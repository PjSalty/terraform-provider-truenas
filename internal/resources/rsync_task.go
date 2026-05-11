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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &RsyncTaskResource{}
	_ resource.ResourceWithImportState = &RsyncTaskResource{}
	_ resource.ResourceWithModifyPlan  = &RsyncTaskResource{}
)

// RsyncTaskResource manages a TrueNAS rsync task.
type RsyncTaskResource struct {
	client *client.Client
}

// RsyncTaskResourceModel describes the resource data model.
type RsyncTaskResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Path         types.String   `tfsdk:"path"`
	Remotehost   types.String   `tfsdk:"remotehost"`
	Remoteport   types.Int64    `tfsdk:"remoteport"`
	Mode         types.String   `tfsdk:"mode"`
	Remotemodule types.String   `tfsdk:"remotemodule"`
	Remotepath   types.String   `tfsdk:"remotepath"`
	Direction    types.String   `tfsdk:"direction"`
	User         types.String   `tfsdk:"user"`
	Enabled      types.Bool     `tfsdk:"enabled"`
	Desc         types.String   `tfsdk:"desc"`
	Minute       types.String   `tfsdk:"schedule_minute"`
	Hour         types.String   `tfsdk:"schedule_hour"`
	Dom          types.String   `tfsdk:"schedule_dom"`
	Month        types.String   `tfsdk:"schedule_month"`
	Dow          types.String   `tfsdk:"schedule_dow"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func NewRsyncTaskResource() resource.Resource {
	return &RsyncTaskResource{}
}

func (r *RsyncTaskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rsync_task"
}

func (r *RsyncTaskResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an rsync task on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the rsync task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The local path to sync (e.g., /mnt/tank/data).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
				},
			},
			"remotehost": schema.StringAttribute{
				Description: "The remote host to sync with.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"remoteport": schema.Int64Attribute{
				Description: "The remote SSH port.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(22),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"mode": schema.StringAttribute{
				Description: "Rsync mode: SSH or MODULE.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("MODULE"),
				Validators: []validator.String{
					stringvalidator.OneOf("SSH", "MODULE"),
				},
			},
			"remotemodule": schema.StringAttribute{
				Description: "The remote rsync module name (for MODULE mode).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(120),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"remotepath": schema.StringAttribute{
				Description: "The remote path (for SSH mode).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1023),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"direction": schema.StringAttribute{
				Description: "Sync direction: PUSH or PULL.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("PUSH"),
				Validators: []validator.String{
					stringvalidator.OneOf("PUSH", "PULL"),
				},
			},
			"user": schema.StringAttribute{
				Description: "The user to run the rsync task as.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the rsync task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"desc": schema.StringAttribute{
				Description: "A description for the rsync task.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"schedule_minute": schema.StringAttribute{
				Description: "Cron schedule minute field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"schedule_hour": schema.StringAttribute{
				Description: "Cron schedule hour field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0"),
			},
			"schedule_dom": schema.StringAttribute{
				Description: "Cron schedule day-of-month field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*"),
			},
			"schedule_month": schema.StringAttribute{
				Description: "Cron schedule month field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*"),
			},
			"schedule_dow": schema.StringAttribute{
				Description: "Cron schedule day-of-week field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("*"),
			},
		},
	}
}

func (r *RsyncTaskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RsyncTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create RsyncTask start")

	var plan RsyncTaskResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.RsyncTaskCreateRequest{
		Path:    plan.Path.ValueString(),
		User:    plan.User.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
		Schedule: client.Schedule{
			Minute: plan.Minute.ValueString(),
			Hour:   plan.Hour.ValueString(),
			Dom:    plan.Dom.ValueString(),
			Month:  plan.Month.ValueString(),
			Dow:    plan.Dow.ValueString(),
		},
	}

	if !plan.Remotehost.IsNull() {
		createReq.Remotehost = plan.Remotehost.ValueString()
	}
	if !plan.Remoteport.IsNull() {
		createReq.Remoteport = int(plan.Remoteport.ValueInt64())
	}
	if !plan.Mode.IsNull() {
		createReq.Mode = plan.Mode.ValueString()
	}
	if !plan.Remotemodule.IsNull() {
		createReq.Remotemodule = plan.Remotemodule.ValueString()
	}
	if !plan.Remotepath.IsNull() {
		createReq.Remotepath = plan.Remotepath.ValueString()
	}
	if !plan.Direction.IsNull() {
		createReq.Direction = plan.Direction.ValueString()
	}
	if !plan.Desc.IsNull() {
		createReq.Desc = plan.Desc.ValueString()
	}

	tflog.Debug(ctx, "Creating rsync task", map[string]interface{}{
		"path": plan.Path.ValueString(),
	})

	task, err := r.client.CreateRsyncTask(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Rsync Task",
			fmt.Sprintf("Could not create rsync task: %s", err),
		)
		return
	}

	r.mapResponseToModel(task, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create RsyncTask success")
}

func (r *RsyncTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read RsyncTask start")

	var state RsyncTaskResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse rsync task ID: %s", err))
		return
	}

	task, err := r.client.GetRsyncTask(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Rsync Task",
			fmt.Sprintf("Could not read rsync task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(task, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read RsyncTask success")
}

func (r *RsyncTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update RsyncTask start")

	var plan RsyncTaskResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RsyncTaskResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse rsync task ID: %s", err))
		return
	}

	enabled := plan.Enabled.ValueBool()
	schedule := &client.Schedule{
		Minute: plan.Minute.ValueString(),
		Hour:   plan.Hour.ValueString(),
		Dom:    plan.Dom.ValueString(),
		Month:  plan.Month.ValueString(),
		Dow:    plan.Dow.ValueString(),
	}

	updateReq := &client.RsyncTaskUpdateRequest{
		Path:     plan.Path.ValueString(),
		User:     plan.User.ValueString(),
		Enabled:  &enabled,
		Schedule: schedule,
	}

	if !plan.Remotehost.IsNull() {
		updateReq.Remotehost = plan.Remotehost.ValueString()
	}
	if !plan.Remoteport.IsNull() {
		updateReq.Remoteport = int(plan.Remoteport.ValueInt64())
	}
	if !plan.Mode.IsNull() {
		updateReq.Mode = plan.Mode.ValueString()
	}
	if !plan.Remotemodule.IsNull() {
		updateReq.Remotemodule = plan.Remotemodule.ValueString()
	}
	if !plan.Remotepath.IsNull() {
		updateReq.Remotepath = plan.Remotepath.ValueString()
	}
	if !plan.Direction.IsNull() {
		updateReq.Direction = plan.Direction.ValueString()
	}
	if !plan.Desc.IsNull() {
		updateReq.Desc = plan.Desc.ValueString()
	}

	task, err := r.client.UpdateRsyncTask(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Rsync Task",
			fmt.Sprintf("Could not update rsync task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(task, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update RsyncTask success")
}

func (r *RsyncTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete RsyncTask start")

	var state RsyncTaskResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse rsync task ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting rsync task", map[string]interface{}{"id": id})

	err = r.client.DeleteRsyncTask(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Rsync task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Rsync Task",
			fmt.Sprintf("Could not delete rsync task %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete RsyncTask success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *RsyncTaskResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_rsync_task")
}

func (r *RsyncTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Rsync task ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *RsyncTaskResource) mapResponseToModel(task *client.RsyncTask, model *RsyncTaskResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(task.ID))
	model.Path = types.StringValue(task.Path)
	model.Remotehost = types.StringValue(task.Remotehost)
	model.Remoteport = types.Int64Value(int64(task.Remoteport))
	model.Mode = types.StringValue(task.Mode)
	model.Remotemodule = types.StringValue(task.Remotemodule)
	model.Remotepath = types.StringValue(task.Remotepath)
	model.Direction = types.StringValue(task.Direction)
	model.User = types.StringValue(task.User)
	model.Enabled = types.BoolValue(task.Enabled)
	model.Desc = types.StringValue(task.Desc)
	model.Minute = types.StringValue(task.Schedule.Minute)
	model.Hour = types.StringValue(task.Schedule.Hour)
	model.Dom = types.StringValue(task.Schedule.Dom)
	model.Month = types.StringValue(task.Schedule.Month)
	model.Dow = types.StringValue(task.Schedule.Dow)
}
