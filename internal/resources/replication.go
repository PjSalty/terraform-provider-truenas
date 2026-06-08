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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
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
	_ resource.Resource                = &ReplicationResource{}
	_ resource.ResourceWithImportState = &ReplicationResource{}
	_ resource.ResourceWithModifyPlan  = &ReplicationResource{}
)

// ReplicationResource manages a ZFS replication task.
type ReplicationResource struct {
	client *client.Client
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
	Timeouts                timeouts.Value `tfsdk:"timeouts"`
}

func NewReplicationResource() resource.Resource {
	return &ReplicationResource{}
}

func (r *ReplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_replication"
}

func (r *ReplicationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a ZFS replication task on TrueNAS SCALE. " +
		"Default timeouts: 30m for create/update (initial snapshot sync may be slow), 10m for delete.",
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
				Description: "Whether to run the replication automatically on schedule.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
		},
	}
}

func (r *ReplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create Replication start")

	var plan ReplicationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var sourceDatasets []string
	resp.Diagnostics.Append(plan.SourceDatasets.ElementsAs(ctx, &sourceDatasets, false)...)
	createReq := &client.ReplicationCreateRequest{
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

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	repl, err := r.client.GetReplication(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
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

	updateReq := &client.ReplicationUpdateRequest{
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

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse replication ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting replication task", map[string]interface{}{"id": id})

	err = r.client.DeleteReplication(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
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
//     to be set — otherwise snapshots never expire.
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

	if transport != "LOCAL" && transport != "" && !sshSet {
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
	lifetimeSet := !config.LifetimeValue.IsNull() && !config.LifetimeValue.IsUnknown() && config.LifetimeValue.ValueInt64() > 0
	lifetimeUnitSet := !config.LifetimeUnit.IsNull() && !config.LifetimeUnit.IsUnknown() && config.LifetimeUnit.ValueString() != ""
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

func (r *ReplicationResource) mapResponseToModel(ctx context.Context, repl *client.Replication, model *ReplicationResourceModel) {
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
}
