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
	_ resource.Resource                = &SnapshotTaskResource{}
	_ resource.ResourceWithImportState = &SnapshotTaskResource{}
	_ resource.ResourceWithModifyPlan  = &SnapshotTaskResource{}
)

// SnapshotTaskResource manages a periodic snapshot task.
type SnapshotTaskResource struct {
	client *client.Client
}

// SnapshotTaskResourceModel describes the resource data model.
type SnapshotTaskResourceModel struct {
	ID           types.String   `tfsdk:"id"`
	Dataset      types.String   `tfsdk:"dataset"`
	Recursive    types.Bool     `tfsdk:"recursive"`
	Lifetime     types.Int64    `tfsdk:"lifetime_value"`
	LifetimeUnit types.String   `tfsdk:"lifetime_unit"`
	NamingSchema types.String   `tfsdk:"naming_schema"`
	Enabled      types.Bool     `tfsdk:"enabled"`
	AllowEmpty   types.Bool     `tfsdk:"allow_empty"`
	Minute       types.String   `tfsdk:"schedule_minute"`
	Hour         types.String   `tfsdk:"schedule_hour"`
	Dom          types.String   `tfsdk:"schedule_dom"`
	Month        types.String   `tfsdk:"schedule_month"`
	Dow          types.String   `tfsdk:"schedule_dow"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func NewSnapshotTaskResource() resource.Resource {
	return &SnapshotTaskResource{}
}

func (r *SnapshotTaskResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot_task"
}

func (r *SnapshotTaskResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a periodic ZFS snapshot task on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the snapshot task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dataset": schema.StringAttribute{
				Description: "The dataset to snapshot (e.g., tank/data).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"recursive": schema.BoolAttribute{
				Description: "Whether to recursively snapshot child datasets.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"lifetime_value": schema.Int64Attribute{
				Description: "How long to keep snapshots (numeric value, 1-9999).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.Between(1, 9999),
				},
			},
			"lifetime_unit": schema.StringAttribute{
				Description: "Lifetime unit (HOUR, DAY, WEEK, MONTH, YEAR).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("WEEK"),
				Validators: []validator.String{
					stringvalidator.OneOf("HOUR", "DAY", "WEEK", "MONTH", "YEAR"),
				},
			},
			"naming_schema": schema.StringAttribute{
				Description: "Naming schema for snapshots (e.g., auto-%Y-%m-%d_%H-%M).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("auto-%Y-%m-%d_%H-%M"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"allow_empty": schema.BoolAttribute{
				Description: "Whether to create snapshots even if there are no changes.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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

func (r *SnapshotTaskResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SnapshotTaskResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SnapshotTask start")

	var plan SnapshotTaskResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.SnapshotTaskCreateRequest{
		Dataset:      plan.Dataset.ValueString(),
		Recursive:    plan.Recursive.ValueBool(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		LifetimeUnit: plan.LifetimeUnit.ValueString(),
		NamingSchema: plan.NamingSchema.ValueString(),
		Enabled:      plan.Enabled.ValueBool(),
		AllowEmpty:   plan.AllowEmpty.ValueBool(),
		Schedule: client.Schedule{
			Minute: plan.Minute.ValueString(),
			Hour:   plan.Hour.ValueString(),
			Dom:    plan.Dom.ValueString(),
			Month:  plan.Month.ValueString(),
			Dow:    plan.Dow.ValueString(),
		},
	}

	tflog.Debug(ctx, "Creating snapshot task", map[string]interface{}{
		"dataset": plan.Dataset.ValueString(),
	})

	task, err := r.client.CreateSnapshotTask(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Snapshot Task",
			fmt.Sprintf("Could not create snapshot task for dataset %q: %s", plan.Dataset.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(task, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SnapshotTask success")
}

func (r *SnapshotTaskResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SnapshotTask start")

	var state SnapshotTaskResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse snapshot task ID: %s", err))
		return
	}

	task, err := r.client.GetSnapshotTask(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Snapshot Task",
			fmt.Sprintf("Could not read snapshot task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(task, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SnapshotTask success")
}

func (r *SnapshotTaskResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SnapshotTask start")

	var plan SnapshotTaskResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state SnapshotTaskResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse snapshot task ID: %s", err))
		return
	}

	recursive := plan.Recursive.ValueBool()
	enabled := plan.Enabled.ValueBool()
	allowEmpty := plan.AllowEmpty.ValueBool()
	schedule := &client.Schedule{
		Minute: plan.Minute.ValueString(),
		Hour:   plan.Hour.ValueString(),
		Dom:    plan.Dom.ValueString(),
		Month:  plan.Month.ValueString(),
		Dow:    plan.Dow.ValueString(),
	}

	updateReq := &client.SnapshotTaskUpdateRequest{
		Dataset:      plan.Dataset.ValueString(),
		Recursive:    &recursive,
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		LifetimeUnit: plan.LifetimeUnit.ValueString(),
		NamingSchema: plan.NamingSchema.ValueString(),
		Enabled:      &enabled,
		AllowEmpty:   &allowEmpty,
		Schedule:     schedule,
	}

	task, err := r.client.UpdateSnapshotTask(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Snapshot Task",
			fmt.Sprintf("Could not update snapshot task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(task, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SnapshotTask success")
}

func (r *SnapshotTaskResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SnapshotTask start")

	var state SnapshotTaskResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse snapshot task ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting snapshot task", map[string]interface{}{"id": id})

	err = r.client.DeleteSnapshotTask(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Snapshot task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Snapshot Task",
			fmt.Sprintf("Could not delete snapshot task %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete SnapshotTask success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *SnapshotTaskResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_snapshot_task")
}

func (r *SnapshotTaskResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Snapshot task ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SnapshotTaskResource) mapResponseToModel(task *client.SnapshotTask, model *SnapshotTaskResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(task.ID))
	model.Dataset = types.StringValue(task.Dataset)
	model.Recursive = types.BoolValue(task.Recursive)
	model.Lifetime = types.Int64Value(int64(task.Lifetime))
	model.LifetimeUnit = types.StringValue(task.LifetimeUnit)
	model.NamingSchema = types.StringValue(task.NamingSchema)
	model.Enabled = types.BoolValue(task.Enabled)
	model.AllowEmpty = types.BoolValue(task.AllowEmpty)
	model.Minute = types.StringValue(task.Schedule.Minute)
	model.Hour = types.StringValue(task.Schedule.Hour)
	model.Dom = types.StringValue(task.Schedule.Dom)
	model.Month = types.StringValue(task.Schedule.Month)
	model.Dow = types.StringValue(task.Schedule.Dow)
}
