package resources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
	_ resource.Resource                 = &CronJobResource{}
	_ resource.ResourceWithImportState  = &CronJobResource{}
	_ resource.ResourceWithUpgradeState = &CronJobResource{}
	_ resource.ResourceWithModifyPlan   = &CronJobResource{}
)

// CronJobResource manages a TrueNAS cron job.
type CronJobResource struct {
	client *client.Client
}

// CronJobResourceModel describes the resource data model.
type CronJobResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	User        types.String   `tfsdk:"user"`
	Command     types.String   `tfsdk:"command"`
	Description types.String   `tfsdk:"description"`
	Enabled     types.Bool     `tfsdk:"enabled"`
	Stdout      types.Bool     `tfsdk:"stdout"`
	Stderr      types.Bool     `tfsdk:"stderr"`
	Minute      types.String   `tfsdk:"schedule_minute"`
	Hour        types.String   `tfsdk:"schedule_hour"`
	Dom         types.String   `tfsdk:"schedule_dom"`
	Month       types.String   `tfsdk:"schedule_month"`
	Dow         types.String   `tfsdk:"schedule_dow"`
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
}

func NewCronJobResource() resource.Resource {
	return &CronJobResource{}
}

func (r *CronJobResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cronjob"
}

func (r *CronJobResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Blocks:      map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})},
		Description: "Manages a cron job on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the cron job.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user": schema.StringAttribute{
				Description: "The user to run the cron job as.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 32),
				},
			},
			"command": schema.StringAttribute{
				Description: "The command to execute.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 4096),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the cron job.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the cron job is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stdout": schema.BoolAttribute{
				Description: "Whether to redirect stdout.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stderr": schema.BoolAttribute{
				Description: "Whether to redirect stderr.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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

func (r *CronJobResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create CronJob start")

	var plan CronJobResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CronJobCreateRequest{
		User:    plan.User.ValueString(),
		Command: plan.Command.ValueString(),
		Enabled: plan.Enabled.ValueBool(),
		Stdout:  plan.Stdout.ValueBool(),
		Stderr:  plan.Stderr.ValueBool(),
		Schedule: client.Schedule{
			Minute: plan.Minute.ValueString(),
			Hour:   plan.Hour.ValueString(),
			Dom:    plan.Dom.ValueString(),
			Month:  plan.Month.ValueString(),
			Dow:    plan.Dow.ValueString(),
		},
	}

	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueString()
	}

	tflog.Debug(ctx, "Creating cron job", map[string]interface{}{
		"command": plan.Command.ValueString(),
	})

	job, err := r.client.CreateCronJob(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Cron Job",
			fmt.Sprintf("Could not create cron job: %s", err),
		)
		return
	}

	r.mapResponseToModel(job, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create CronJob success")
}

func (r *CronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read CronJob start")

	var state CronJobResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cron job ID: %s", err))
		return
	}

	job, err := r.client.GetCronJob(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Cron Job",
			fmt.Sprintf("Could not read cron job %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(job, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read CronJob success")
}

func (r *CronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update CronJob start")

	var plan CronJobResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CronJobResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cron job ID: %s", err))
		return
	}

	enabled := plan.Enabled.ValueBool()
	stdout := plan.Stdout.ValueBool()
	stderr := plan.Stderr.ValueBool()
	schedule := &client.Schedule{
		Minute: plan.Minute.ValueString(),
		Hour:   plan.Hour.ValueString(),
		Dom:    plan.Dom.ValueString(),
		Month:  plan.Month.ValueString(),
		Dow:    plan.Dow.ValueString(),
	}

	updateReq := &client.CronJobUpdateRequest{
		User:     plan.User.ValueString(),
		Command:  plan.Command.ValueString(),
		Enabled:  &enabled,
		Stdout:   &stdout,
		Stderr:   &stderr,
		Schedule: schedule,
	}

	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	job, err := r.client.UpdateCronJob(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Cron Job",
			fmt.Sprintf("Could not update cron job %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(job, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update CronJob success")
}

func (r *CronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete CronJob start")

	var state CronJobResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cron job ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting cron job", map[string]interface{}{"id": id})

	err = r.client.DeleteCronJob(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Cron job already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Cron Job",
			fmt.Sprintf("Could not delete cron job %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete CronJob success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *CronJobResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_cronjob")
}

func (r *CronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Cron job ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// cronjobSchemaV0 returns the historical v0 schema for the cron job resource.
// v0 and v1 are structurally identical — this entry seeds the schema-version migration
// pattern so future breaking schema changes have an idiomatic place to land.
// See terraform-plugin-framework ResourceWithUpgradeState for the contract.
func cronjobSchemaV0(ctx context.Context) schema.Schema {
	return schema.Schema{
		Version:     0,
		Blocks:      map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})},
		Description: "Manages a cron job on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"user":            schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 32)}},
			"command":         schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthBetween(1, 4096)}},
			"description":     schema.StringAttribute{Optional: true, Computed: true, Validators: []validator.String{stringvalidator.LengthAtMost(1024)}},
			"enabled":         schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"stdout":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"stderr":          schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"schedule_minute": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("0")},
			"schedule_hour":   schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("0")},
			"schedule_dom":    schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("*")},
			"schedule_month":  schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("*")},
			"schedule_dow":    schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("*")},
		},
	}
}

// UpgradeState provides schema-version upgraders. The 0 -> 1 migration is
// intentionally a no-op — it exists to establish the pattern so future
// schema changes can land a real migration without introducing new scaffolding.
func (r *CronJobResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	v0 := cronjobSchemaV0(ctx)
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &v0,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// v0 and v1 are structurally identical so we copy the model
				// through unchanged. This is the correct shape for a future
				// migration to evolve: load v0 state into a v0 model, transform
				// to a v1 model, then Set() the new state.
				var prior CronJobResourceModel
				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}
				resp.Diagnostics.Append(resp.State.Set(ctx, prior)...)
			},
		},
	}
}

func (r *CronJobResource) mapResponseToModel(job *client.CronJob, model *CronJobResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(job.ID))
	model.User = types.StringValue(job.User)
	model.Command = types.StringValue(job.Command)
	model.Description = types.StringValue(job.Description)
	model.Enabled = types.BoolValue(job.Enabled)
	model.Stdout = types.BoolValue(job.Stdout)
	model.Stderr = types.BoolValue(job.Stderr)
	model.Minute = types.StringValue(job.Schedule.Minute)
	model.Hour = types.StringValue(job.Schedule.Hour)
	model.Dom = types.StringValue(job.Schedule.Dom)
	model.Month = types.StringValue(job.Schedule.Month)
	model.Dow = types.StringValue(job.Schedule.Dow)
}
