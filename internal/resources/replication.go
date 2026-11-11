package resources

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/flex"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var (
	_ resource.Resource                = &ReplicationResource{}
	_ resource.ResourceWithImportState = &ReplicationResource{}
	_ resource.ResourceWithModifyPlan  = &ReplicationResource{}
)

// ReplicationResource manages a ZFS replication task.
type ReplicationResource struct {
	client *wsclient.Client
}

// ReplicationResourceModel describes the resource data model.
type ReplicationResourceModel struct {
	ID                      types.String   `tfsdk:"id"`
	Name                    types.String   `tfsdk:"name"`
	Direction               types.String   `tfsdk:"direction"`
	Transport               types.String   `tfsdk:"transport"`
	SourceDatasets          types.List     `tfsdk:"source_datasets"`
	TargetDataset           types.String   `tfsdk:"target_dataset"`
	Recursive               types.Bool     `tfsdk:"recursive"`
	Auto                    types.Bool     `tfsdk:"auto"`
	Enabled                 types.Bool     `tfsdk:"enabled"`
	RetentionPolicy         types.String   `tfsdk:"retention_policy"`
	LifetimeValue           types.Int64    `tfsdk:"lifetime_value"`
	LifetimeUnit            types.String   `tfsdk:"lifetime_unit"`
	SSHCredentials          types.Int64    `tfsdk:"ssh_credentials"`
	NamingSchema            types.List     `tfsdk:"naming_schema"`
	AlsoIncludeNamingSchema types.List     `tfsdk:"also_include_naming_schema"`
	PeriodicSnapshotTasks   types.Set      `tfsdk:"periodic_snapshot_tasks"`
	HoldPendingSnapshots    types.Bool     `tfsdk:"hold_pending_snapshots"`
	Schedule                types.List     `tfsdk:"schedule"`
	RestrictSchedule        types.List     `tfsdk:"restrict_schedule"`
	OnlyMatchingSchedule    types.Bool     `tfsdk:"only_matching_schedule"`
	Timeouts                timeouts.Value `tfsdk:"timeouts"`
}

// replicationScheduleModel is the one entry of schedule or restrict_schedule.
type replicationScheduleModel struct {
	Minute types.String `tfsdk:"minute"`
	Hour   types.String `tfsdk:"hour"`
	Dom    types.String `tfsdk:"dom"`
	Month  types.String `tfsdk:"month"`
	Dow    types.String `tfsdk:"dow"`
	Begin  types.String `tfsdk:"begin"`
	End    types.String `tfsdk:"end"`
}

// timeOfDay is the HH:MM form TrueNAS returns for begin and end. Only that
// form is accepted, so the planned value and the value read back agree.
var timeOfDay = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

func replicationScheduleObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"minute": types.StringType,
		"hour":   types.StringType,
		"dom":    types.StringType,
		"month":  types.StringType,
		"dow":    types.StringType,
		"begin":  types.StringType,
		"end":    types.StringType,
	}}
}

// replicationScheduleAttribute is the schema of schedule and
// restrict_schedule. TrueNAS holds either a schedule or null, and a nested
// object that is read back when unset has no way to say null, so the
// attribute is a list of at most one entry: left out it keeps what TrueNAS
// has, `[]` removes the schedule, and one entry sets it.
//
// A field left out of the entry is read back too. It stays unknown on a new
// entry, where there is no prior value and TrueNAS fills its default, so the
// plain UseStateForUnknown, which would plan null there, is not used.
func replicationScheduleAttribute(description string) schema.ListNestedAttribute {
	field := func(description string, validators ...validator.String) schema.StringAttribute {
		return schema.StringAttribute{
			Description: description,
			Optional:    true,
			Computed:    true,
			Validators:  validators,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseNonNullStateForUnknown(),
			},
		}
	}
	isTime := stringvalidator.RegexMatches(timeOfDay, "must be a 24-hour time written as HH:MM, such as 09:30")
	return schema.ListNestedAttribute{
		Description: description,
		Optional:    true,
		Computed:    true,
		Validators: []validator.List{
			listvalidator.SizeAtMost(1),
		},
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseNonNullStateForUnknown(),
		},
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"minute": field("Cron minute field. TrueNAS defaults to `00`."),
				"hour":   field("Cron hour field. TrueNAS defaults to `*`."),
				"dom":    field("Cron day-of-month field. TrueNAS defaults to `*`."),
				"month":  field("Cron month field. TrueNAS defaults to `*`."),
				"dow":    field("Cron day-of-week field, `1` (Monday) to `7`. TrueNAS defaults to `*`."),
				"begin":  field("Start of the daily window in which the schedule applies, as HH:MM. TrueNAS defaults to `00:00`.", isTime),
				"end":    field("End of the daily window, as HH:MM. TrueNAS defaults to `23:59`.", isTime),
			},
		},
	}
}

func NewReplicationResource() resource.Resource {
	return &ReplicationResource{}
}

func (r *ReplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_replication"
}

func (r *ReplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a ZFS replication task on TrueNAS SCALE. " +
		"The initial snapshot sync can be slow; raise `create` and `update` in the `timeouts` block if they time out.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the replication task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the replication task.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"direction": schema.StringAttribute{
				Description: "Replication direction (PUSH or PULL).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PUSH", "PULL"),
				},
			},
			"transport": schema.StringAttribute{
				Description: "Transport type (SSH, SSH+NETCAT, LOCAL).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LOCAL"),
				Validators: []validator.String{
					stringvalidator.OneOf("SSH", "SSH+NETCAT", "LOCAL", "LEGACY"),
				},
			},
			"source_datasets": schema.ListAttribute{
				Description: "List of source dataset paths.",
				Required:    true,
				ElementType: types.StringType,
				// A replication task with no sources replicates nothing.
				// The API accepts an empty list on update, so without this
				// the task silently stops doing anything and the apply
				// still reports success.
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"target_dataset": schema.StringAttribute{
				Description: "Target dataset path.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 200),
				},
			},
			"recursive": schema.BoolAttribute{
				Description: "Whether to recursively replicate child datasets.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"auto": schema.BoolAttribute{
				Description: "Whether the task runs on its own. TrueNAS accepts `auto = true` only with a `schedule`, " +
					"or, for a push task, with `periodic_snapshot_tasks`. Set `auto = false` for a task that is started by hand.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the replication task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"retention_policy": schema.StringAttribute{
				Description: "Snapshot retention policy (SOURCE, CUSTOM, NONE).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("SOURCE"),
				Validators: []validator.String{
					stringvalidator.OneOf("SOURCE", "CUSTOM", "NONE"),
				},
			},
			"lifetime_value": schema.Int64Attribute{
				Description: "Lifetime value for CUSTOM retention policy (1-9999).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(1, 9999),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"lifetime_unit": schema.StringAttribute{
				Description: "Lifetime unit for CUSTOM retention (HOUR, DAY, WEEK, MONTH, YEAR).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("HOUR", "DAY", "WEEK", "MONTH", "YEAR"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_credentials": schema.Int64Attribute{
				Description: "SSH credentials ID for remote replication.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"naming_schema": schema.ListAttribute{
				Description: "Naming schema for matching snapshots on the source (for pull replication).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"also_include_naming_schema": schema.ListAttribute{
				Description: "Naming schema for snapshots to include (for push replication without periodic snapshot tasks).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"periodic_snapshot_tasks": schema.SetAttribute{
				Description: "IDs of the periodic snapshot tasks a push replication is bound to. The task replicates their " +
					"snapshots and, with `auto = true`, runs after each of them. Left out, the current binding is kept. " +
					"Set to `[]` to unbind every task.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Validators: []validator.Set{
					setvalidator.ValueInt64sAre(int64validator.AtLeast(1)),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseNonNullStateForUnknown(),
				},
			},
			"hold_pending_snapshots": schema.BoolAttribute{
				Description: "Keep source snapshots that have not been replicated yet, so that retention cannot delete them " +
					"after a failed run. Push replication only. Left out, the current value is kept; TrueNAS defaults to false.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseNonNullStateForUnknown(),
				},
			},
			"schedule": replicationScheduleAttribute("When the task runs on its own: a list with one entry. " +
				"Left out, the current schedule is kept. Set to `[]` for no schedule, which TrueNAS requires when " +
				"`auto = false`. In the entry, a field that is left out keeps its current value, or takes the TrueNAS " +
				"default on a new schedule."),
			"restrict_schedule": replicationScheduleAttribute("Limits when a task bound to `periodic_snapshot_tasks` runs: " +
				"a list with one entry. For example, snapshot tasks can run every 15 minutes while the replication runs " +
				"once an hour. Left out, the current value is kept. Set to `[]` for no restriction. In the entry, a field " +
				"that is left out keeps its current value, or takes the TrueNAS default on a new entry."),
			"only_matching_schedule": schema.BoolAttribute{
				Description: "Replicate only the snapshots that match `schedule` or `restrict_schedule`. TrueNAS accepts " +
					"true only when `schedule` is set. Left out, the current value is kept; TrueNAS defaults to false.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseNonNullStateForUnknown(),
				},
			},
		},
	}
}

func (r *ReplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*wsclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *wsclient.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

func (r *ReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Replication start")

	var plan ReplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithCreateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	var sourceDatasets []string
	resp.Diagnostics.Append(plan.SourceDatasets.ElementsAs(ctx, &sourceDatasets, false)...)
	createReq := &truenas.ReplicationCreateRequest{
		Name:            plan.Name.ValueString(),
		Direction:       plan.Direction.ValueString(),
		Transport:       plan.Transport.ValueString(),
		SourceDatasets:  sourceDatasets,
		TargetDataset:   plan.TargetDataset.ValueString(),
		Recursive:       plan.Recursive.ValueBool(),
		AutoBool:        plan.Auto.ValueBool(),
		Enabled:         plan.Enabled.ValueBool(),
		RetentionPolicy: plan.RetentionPolicy.ValueString(),
	}

	if !plan.LifetimeValue.IsNull() {
		createReq.LifetimeValue = int(plan.LifetimeValue.ValueInt64())
	}
	if !plan.LifetimeUnit.IsNull() {
		createReq.LifetimeUnit = plan.LifetimeUnit.ValueString()
	}
	if !plan.SSHCredentials.IsNull() && !plan.SSHCredentials.IsUnknown() {
		createReq.SSHCredentials = int(plan.SSHCredentials.ValueInt64())
	}
	if !plan.NamingSchema.IsNull() && !plan.NamingSchema.IsUnknown() {
		var ns []string
		resp.Diagnostics.Append(plan.NamingSchema.ElementsAs(ctx, &ns, false)...)
		createReq.NamingSchema = ns
	}
	if !plan.AlsoIncludeNamingSchema.IsNull() && !plan.AlsoIncludeNamingSchema.IsUnknown() {
		var ains []string
		resp.Diagnostics.Append(plan.AlsoIncludeNamingSchema.ElementsAs(ctx, &ains, false)...)
		createReq.AlsoIncludeNamingSchema = ains
	}
	// Each of these is left out while its planned value is unknown, and
	// TrueNAS then applies its own default.
	createReq.Schedule = replicationScheduleRequest(ctx, plan.Schedule, &resp.Diagnostics)
	createReq.RestrictSchedule = replicationScheduleRequest(ctx, plan.RestrictSchedule, &resp.Diagnostics)
	createReq.OnlyMatchingSchedule = flex.BoolFromPointer(plan.OnlyMatchingSchedule)
	createReq.HoldPendingSnapshots = flex.BoolFromPointer(plan.HoldPendingSnapshots)
	createReq.PeriodicSnapshotTasks = snapshotTaskIDs(ctx, plan.PeriodicSnapshotTasks, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating replication task", map[string]interface{}{
		"name": plan.Name.ValueString(),
	})

	repl, err := r.client.CreateReplication(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Replication",
			fmt.Sprintf("Could not create replication task %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(ctx, repl, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create Replication success")
}

func (r *ReplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read Replication start")

	var state ReplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithReadTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	repl, err := r.client.GetReplication(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Replication",
			fmt.Sprintf("Could not read replication %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, repl, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read Replication success")
}

func (r *ReplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update Replication start")

	var plan ReplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithUpdateTimeout(ctx, plan.Timeouts, &resp.Diagnostics)
	defer cancel()

	var state ReplicationResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	var sourceDatasets []string
	resp.Diagnostics.Append(plan.SourceDatasets.ElementsAs(ctx, &sourceDatasets, false)...)
	recursive := plan.Recursive.ValueBool()
	auto := plan.Auto.ValueBool()
	enabled := plan.Enabled.ValueBool()

	updateReq := &truenas.ReplicationUpdateRequest{
		Name:            plan.Name.ValueString(),
		Direction:       plan.Direction.ValueString(),
		Transport:       plan.Transport.ValueString(),
		SourceDatasets:  sourceDatasets,
		TargetDataset:   plan.TargetDataset.ValueString(),
		Recursive:       &recursive,
		AutoBool:        &auto,
		Enabled:         &enabled,
		RetentionPolicy: plan.RetentionPolicy.ValueString(),
	}

	if !plan.LifetimeValue.IsNull() {
		updateReq.LifetimeValue = int(plan.LifetimeValue.ValueInt64())
	}
	if !plan.LifetimeUnit.IsNull() {
		updateReq.LifetimeUnit = plan.LifetimeUnit.ValueString()
	}
	if !plan.SSHCredentials.IsNull() {
		updateReq.SSHCredentials = int(plan.SSHCredentials.ValueInt64())
	}
	updateReq.NamingSchema = namingSchemaRequest(ctx, plan.NamingSchema, &resp.Diagnostics)
	updateReq.AlsoIncludeNamingSchema = namingSchemaRequest(ctx, plan.AlsoIncludeNamingSchema, &resp.Diagnostics)
	updateReq.Schedule = replicationScheduleRequest(ctx, plan.Schedule, &resp.Diagnostics)
	updateReq.RestrictSchedule = replicationScheduleRequest(ctx, plan.RestrictSchedule, &resp.Diagnostics)
	updateReq.OnlyMatchingSchedule = flex.BoolFromPointer(plan.OnlyMatchingSchedule)
	updateReq.HoldPendingSnapshots = flex.BoolFromPointer(plan.HoldPendingSnapshots)
	updateReq.PeriodicSnapshotTasks = snapshotTaskIDs(ctx, plan.PeriodicSnapshotTasks, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	repl, err := r.client.UpdateReplication(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Replication",
			fmt.Sprintf("Could not update replication %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(ctx, repl, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update Replication success")
}

func (r *ReplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete Replication start")

	var state ReplicationResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := planhelpers.WithDeleteTimeout(ctx, state.Timeouts, &resp.Diagnostics)
	defer cancel()

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting replication task", map[string]interface{}{"id": id})

	err = r.client.DeleteReplication(ctx, id)
	if err != nil {
		if wsclient.IsNotFound(err) {
			tflog.Warn(ctx, "Replication task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Replication",
			fmt.Sprintf("Could not delete replication %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete Replication success")
}

// ModifyPlan enforces replication cross-attribute constraints:
//
//   - Any transport other than LOCAL requires `ssh_credentials` to be set.
//     SSH and SSH+NETCAT both need an SSH keypair; LOCAL replication does not.
//   - `retention_policy=CUSTOM` requires `lifetime_value` and `lifetime_unit`
//     to be set, otherwise snapshots never expire.
func (r *ReplicationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_replication")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config ReplicationResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	transport := "LOCAL"
	if !config.Transport.IsNull() && !config.Transport.IsUnknown() {
		transport = config.Transport.ValueString()
	}
	sshSet := !config.SSHCredentials.IsNull() && !config.SSHCredentials.IsUnknown() && config.SSHCredentials.ValueInt64() > 0

	if transport != "LOCAL" && transport != "" && !sshSet && !config.SSHCredentials.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("ssh_credentials"),
			"Missing ssh_credentials",
			fmt.Sprintf("transport=%s requires ssh_credentials to reference a valid "+
				"keychain credential. Only transport=LOCAL can run without SSH.", transport),
		)
	}

	retention := ""
	if !config.RetentionPolicy.IsNull() && !config.RetentionPolicy.IsUnknown() {
		retention = config.RetentionPolicy.ValueString()
	}
	// An unknown lifetime (wired from another resource) is set with its value
	// pending: Terraform plans again with the real value at apply.
	lifetimeSet := !config.LifetimeValue.IsNull() &&
		(config.LifetimeValue.IsUnknown() || config.LifetimeValue.ValueInt64() > 0)
	lifetimeUnitSet := !config.LifetimeUnit.IsNull() &&
		(config.LifetimeUnit.IsUnknown() || config.LifetimeUnit.ValueString() != "")
	if retention == "CUSTOM" && (!lifetimeSet || !lifetimeUnitSet) {
		resp.Diagnostics.AddAttributeError(
			path.Root("lifetime_value"),
			"Missing lifetime for CUSTOM retention",
			"retention_policy=CUSTOM requires both lifetime_value (>0) and lifetime_unit to be set.",
		)
	}
}

func (r *ReplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Replication ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *ReplicationResource) mapResponseToModel(ctx context.Context, repl *truenas.Replication, model *ReplicationResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(repl.ID))
	model.Name = types.StringValue(repl.Name)
	model.Direction = types.StringValue(repl.Direction)
	model.Transport = types.StringValue(repl.Transport)
	model.TargetDataset = types.StringValue(repl.TargetDataset)
	model.Recursive = types.BoolValue(repl.Recursive)
	model.Auto = types.BoolValue(repl.AutoBool)
	model.Enabled = types.BoolValue(repl.Enabled)
	model.RetentionPolicy = types.StringValue(repl.RetentionPolicy)

	if repl.LifetimeValue > 0 {
		model.LifetimeValue = types.Int64Value(int64(repl.LifetimeValue))
	} else {
		model.LifetimeValue = types.Int64Null()
	}
	if repl.LifetimeUnit != "" {
		model.LifetimeUnit = types.StringValue(repl.LifetimeUnit)
	} else {
		model.LifetimeUnit = types.StringNull()
	}
	if repl.SSHCredentials > 0 {
		model.SSHCredentials = types.Int64Value(int64(repl.SSHCredentials))
	} else {
		model.SSHCredentials = types.Int64Null()
	}

	sourceValues, diags := types.ListValueFrom(ctx, types.StringType, repl.SourceDatasets)
	if !diags.HasError() {
		model.SourceDatasets = sourceValues
	}

	if len(repl.NamingSchema) > 0 {
		nsValues, d := types.ListValueFrom(ctx, types.StringType, repl.NamingSchema)
		if !d.HasError() {
			model.NamingSchema = nsValues
		}
	} else {
		model.NamingSchema = types.ListNull(types.StringType)
	}

	if len(repl.AlsoIncludeNamingSchema) > 0 {
		ainsValues, d := types.ListValueFrom(ctx, types.StringType, repl.AlsoIncludeNamingSchema)
		if !d.HasError() {
			model.AlsoIncludeNamingSchema = ainsValues
		}
	} else {
		model.AlsoIncludeNamingSchema = types.ListNull(types.StringType)
	}

	model.HoldPendingSnapshots = types.BoolValue(repl.HoldPendingSnapshots)
	model.OnlyMatchingSchedule = types.BoolValue(repl.OnlyMatchingSchedule)
	model.Schedule = replicationScheduleList(repl.Schedule)
	model.RestrictSchedule = replicationScheduleList(repl.RestrictSchedule)
	taskIDs := make([]attr.Value, 0, len(repl.PeriodicSnapshotTasks))
	for _, id := range repl.PeriodicSnapshotTasks {
		taskIDs = append(taskIDs, types.Int64Value(int64(id)))
	}
	model.PeriodicSnapshotTasks = types.SetValueMust(types.Int64Type, taskIDs)
}

// replicationScheduleList is the state form of a schedule. A task without
// one gives an empty list, never null, so `[]` in a configuration plans clean.
func replicationScheduleList(s *truenas.TimeSchedule) types.List {
	objType := replicationScheduleObjectType()
	if s == nil {
		return types.ListValueMust(objType, []attr.Value{})
	}
	entry := types.ObjectValueMust(objType.AttrTypes, map[string]attr.Value{
		"minute": types.StringValue(s.Minute),
		"hour":   types.StringValue(s.Hour),
		"dom":    types.StringValue(s.Dom),
		"month":  types.StringValue(s.Month),
		"dow":    types.StringValue(s.Dow),
		"begin":  types.StringValue(s.Begin),
		"end":    types.StringValue(s.End),
	})
	return types.ListValueMust(objType, []attr.Value{entry})
}

// replicationScheduleRequest is the request form of a planned schedule list.
// A null or unknown list leaves the field out, an empty list removes the
// schedule, and one entry sets it. A field of the entry that is still
// unknown is left out, and TrueNAS fills its default.
func replicationScheduleRequest(ctx context.Context, l types.List, diags *diag.Diagnostics) truenas.ScheduleRequest {
	if l.IsNull() || l.IsUnknown() {
		return truenas.ScheduleRequest{}
	}
	var entries []replicationScheduleModel
	diags.Append(l.ElementsAs(ctx, &entries, false)...)
	if len(entries) == 0 {
		return truenas.ScheduleRequest{Clear: true}
	}
	e := entries[0]
	return truenas.ScheduleRequest{Value: &truenas.TimeSchedule{
		Minute: e.Minute.ValueString(),
		Hour:   e.Hour.ValueString(),
		Dom:    e.Dom.ValueString(),
		Month:  e.Month.ValueString(),
		Dow:    e.Dow.ValueString(),
		Begin:  e.Begin.ValueString(),
		End:    e.End.ValueString(),
	}}
}

// snapshotTaskIDs is the request form of the planned periodic_snapshot_tasks.
// A null or unknown set gives nil, which leaves the field out. An empty set
// gives an empty list, which unbinds every task. The IDs are sorted, so the
// same set always sends the same request.
func snapshotTaskIDs(ctx context.Context, s types.Set, diags *diag.Diagnostics) *[]int {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	var planned []int64
	diags.Append(s.ElementsAs(ctx, &planned, false)...)
	ids := make([]int, 0, len(planned))
	for _, id := range planned {
		ids = append(ids, int(id))
	}
	slices.Sort(ids)
	return &ids
}

// namingSchemaRequest is the list an update sends for naming_schema or
// also_include_naming_schema. These two are not read back when unset, so a
// null list means none and is sent as an empty list, which clears the field.
// An update used to leave both out, so a changed or removed schema stayed on
// the task.
func namingSchemaRequest(ctx context.Context, l types.List, diags *diag.Diagnostics) *[]string {
	names, d := flex.StringsFromListValue(ctx, l)
	diags.Append(d...)
	if names == nil {
		names = []string{}
	}
	return &names
}
