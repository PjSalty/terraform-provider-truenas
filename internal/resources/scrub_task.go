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
	_ resource.Resource                = &ScrubTaskResource{}
	_ resource.ResourceWithImportState = &ScrubTaskResource{}
	_ resource.ResourceWithModifyPlan  = &ScrubTaskResource{}
)

// ScrubTaskResource manages a ZFS pool scrub task.
type ScrubTaskResource struct {
	client *client.Client
}

// ScrubTaskResourceModel describes the resource data model.
type ScrubTaskResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	Pool        types.Int64    `tfsdk:"pool"`
	PoolName    types.String   `tfsdk:"pool_name"`
	Threshold   types.Int64    `tfsdk:"threshold"`
	Description types.String   `tfsdk:"description"`
	Enabled     types.Bool     `tfsdk:"enabled"`
	Minute      types.String   `tfsdk:"schedule_minute"`
	Hour        types.String   `tfsdk:"schedule_hour"`
	Dom         types.String   `tfsdk:"schedule_dom"`
	Month       types.String   `tfsdk:"schedule_month"`
	Dow         types.String   `tfsdk:"schedule_dow"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func NewScrubTaskResource() resource.Resource {
	return &ScrubTaskResource{}
}

func (r *ScrubTaskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scrub_task"
}

func (r *ScrubTaskResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a ZFS pool scrub schedule on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the scrub task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pool": schema.Int64Attribute{
				Description: "The pool ID to scrub.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"pool_name": schema.StringAttribute{
				Description: "The pool name (read-only, populated by API).",
				Computed:    true,
			},
			"threshold": schema.Int64Attribute{
				Description: "Number of days between scrubs (0-365).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(35),
				Validators: []validator.Int64{
					int64validator.Between(0, 365),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the scrub task.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the scrub task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"schedule_minute": schema.StringAttribute{
				Description: "Cron schedule minute field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("00"),
			},
			"schedule_hour": schema.StringAttribute{
				Description: "Cron schedule hour field.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("00"),
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
				Default:     stringdefault.StaticString("7"),
			},
		},
	}
}

func (r *ScrubTaskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScrubTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create ScrubTask start")

	var plan ScrubTaskResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.ScrubTaskCreateRequest{
		Pool:        int(plan.Pool.ValueInt64()),
		Threshold:   int(plan.Threshold.ValueInt64()),
		Description: plan.Description.ValueString(),
		Enabled:     plan.Enabled.ValueBool(),
		Schedule: client.Schedule{
			Minute: plan.Minute.ValueString(),
			Hour:   plan.Hour.ValueString(),
			Dom:    plan.Dom.ValueString(),
			Month:  plan.Month.ValueString(),
			Dow:    plan.Dow.ValueString(),
		},
	}

	tflog.Debug(ctx, "Creating scrub task", map[string]interface{}{
		"pool": plan.Pool.ValueInt64(),
	})

	task, err := r.client.CreateScrubTask(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Scrub Task",
			fmt.Sprintf("Could not create scrub task for pool %d: %s", plan.Pool.ValueInt64(), err),
		)
		return
	}

	r.mapResponseToModel(task, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create ScrubTask success")
}

func (r *ScrubTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read ScrubTask start")

	var state ScrubTaskResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse scrub task ID: %s", err))
		return
	}

	task, err := r.client.GetScrubTask(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Scrub Task",
			fmt.Sprintf("Could not read scrub task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(task, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read ScrubTask success")
}

func (r *ScrubTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update ScrubTask start")

	var plan ScrubTaskResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ScrubTaskResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse scrub task ID: %s", err))
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

	updateReq := &client.ScrubTaskUpdateRequest{
		Pool:        int(plan.Pool.ValueInt64()),
		Threshold:   int(plan.Threshold.ValueInt64()),
		Description: plan.Description.ValueString(),
		Enabled:     &enabled,
		Schedule:    schedule,
	}

	task, err := r.client.UpdateScrubTask(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Scrub Task",
			fmt.Sprintf("Could not update scrub task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(task, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update ScrubTask success")
}

func (r *ScrubTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete ScrubTask start")

	var state ScrubTaskResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse scrub task ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting scrub task", map[string]interface{}{"id": id})

	err = r.client.DeleteScrubTask(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Scrub task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Scrub Task",
			fmt.Sprintf("Could not delete scrub task %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete ScrubTask success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *ScrubTaskResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_scrub_task")
}

func (r *ScrubTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Scrub task ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ScrubTaskResource) mapResponseToModel(task *client.ScrubTask, model *ScrubTaskResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(task.ID))
	model.Pool = types.Int64Value(int64(task.Pool))
	model.PoolName = types.StringValue(task.PoolName)
	model.Threshold = types.Int64Value(int64(task.Threshold))
	model.Description = types.StringValue(task.Description)
	model.Enabled = types.BoolValue(task.Enabled)
	model.Minute = types.StringValue(task.Schedule.Minute)
	model.Hour = types.StringValue(task.Schedule.Hour)
	model.Dom = types.StringValue(task.Schedule.Dom)
	model.Month = types.StringValue(task.Schedule.Month)
	model.Dow = types.StringValue(task.Schedule.Dow)
}
