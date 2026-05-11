package resources

import (
	"context"
	"encoding/json"
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
	_ resource.Resource                = &CloudSyncResource{}
	_ resource.ResourceWithImportState = &CloudSyncResource{}
	_ resource.ResourceWithModifyPlan  = &CloudSyncResource{}
)

// CloudSyncResource manages a TrueNAS cloud sync task.
type CloudSyncResource struct {
	client *client.Client
}

// CloudSyncResourceModel describes the resource data model.
type CloudSyncResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	Description    types.String   `tfsdk:"description"`
	Path           types.String   `tfsdk:"path"`
	Credentials    types.Int64    `tfsdk:"credentials"`
	Direction      types.String   `tfsdk:"direction"`
	TransferMode   types.String   `tfsdk:"transfer_mode"`
	Enabled        types.Bool     `tfsdk:"enabled"`
	AttributesJSON types.String   `tfsdk:"attributes_json"`
	Minute         types.String   `tfsdk:"schedule_minute"`
	Hour           types.String   `tfsdk:"schedule_hour"`
	Dom            types.String   `tfsdk:"schedule_dom"`
	Month          types.String   `tfsdk:"schedule_month"`
	Dow            types.String   `tfsdk:"schedule_dow"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func NewCloudSyncResource() resource.Resource {
	return &CloudSyncResource{}
}

func (r *CloudSyncResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_sync"
}

func (r *CloudSyncResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages a cloud sync task on TrueNAS SCALE." + "\n\n" +
		"**Stability: Beta.** Create/read/update/destroy wire format verified against TrueNAS SCALE 25.10. Full end-to-end run with real cloud credentials has not been observed — TrueNAS probes the provider at create time, so a full cycle requires valid cloud credentials that this project does not have.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the cloud sync task.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the cloud sync task.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The local path to sync (e.g., /mnt/tank/backup).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 1023),
				},
			},
			"credentials": schema.Int64Attribute{
				Description: "The ID of the cloud credential to use.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"direction": schema.StringAttribute{
				Description: "Sync direction: PUSH or PULL.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("PUSH", "PULL"),
				},
			},
			"transfer_mode": schema.StringAttribute{
				Description: "Transfer mode: SYNC, COPY, or MOVE.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("SYNC", "COPY", "MOVE"),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the cloud sync task is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"attributes_json": schema.StringAttribute{
				Description: "JSON-encoded provider-specific attributes (e.g. " +
					"`{\"bucket\":\"my-bucket\",\"folder\":\"/backups\"}`). " +
					"Exact keys depend on the cloud credential type.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("{}"),
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

// Ensure int64default is used (referenced in schema for potential future use).
var _ = int64default.StaticInt64

func (r *CloudSyncResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CloudSyncResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create CloudSync start")

	var plan CloudSyncResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := &client.CloudSyncCreateRequest{
		Path:         plan.Path.ValueString(),
		Credentials:  int(plan.Credentials.ValueInt64()),
		Direction:    plan.Direction.ValueString(),
		TransferMode: plan.TransferMode.ValueString(),
		Enabled:      plan.Enabled.ValueBool(),
		Schedule: client.Schedule{
			Minute: plan.Minute.ValueString(),
			Hour:   plan.Hour.ValueString(),
			Dom:    plan.Dom.ValueString(),
			Month:  plan.Month.ValueString(),
			Dow:    plan.Dow.ValueString(),
		},
		Attributes: map[string]interface{}{},
	}

	if !plan.AttributesJSON.IsNull() && plan.AttributesJSON.ValueString() != "" {
		if err := json.Unmarshal([]byte(plan.AttributesJSON.ValueString()), &createReq.Attributes); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("attributes_json"),
				"Invalid attributes_json",
				fmt.Sprintf("attributes_json must be a valid JSON object: %s", err),
			)
			return
		}
	}

	if !plan.Description.IsNull() {
		createReq.Description = plan.Description.ValueString()
	}

	tflog.Debug(ctx, "Creating cloud sync task", map[string]interface{}{
		"path": plan.Path.ValueString(),
	})

	cs, err := r.client.CreateCloudSync(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Cloud Sync Task",
			fmt.Sprintf("Could not create cloud sync task: %s", err),
		)
		return
	}

	r.mapResponseToModel(cs, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create CloudSync success")
}

func (r *CloudSyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read CloudSync start")

	var state CloudSyncResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud sync ID: %s", err))
		return
	}

	cs, err := r.client.GetCloudSync(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Cloud Sync Task",
			fmt.Sprintf("Could not read cloud sync task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(cs, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read CloudSync success")
}

func (r *CloudSyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update CloudSync start")

	var plan CloudSyncResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state CloudSyncResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud sync ID: %s", err))
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

	updateReq := &client.CloudSyncUpdateRequest{
		Path:         plan.Path.ValueString(),
		Credentials:  int(plan.Credentials.ValueInt64()),
		Direction:    plan.Direction.ValueString(),
		TransferMode: plan.TransferMode.ValueString(),
		Enabled:      &enabled,
		Schedule:     schedule,
		Attributes:   map[string]interface{}{},
	}

	if !plan.AttributesJSON.IsNull() && plan.AttributesJSON.ValueString() != "" {
		if err := json.Unmarshal([]byte(plan.AttributesJSON.ValueString()), &updateReq.Attributes); err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("attributes_json"),
				"Invalid attributes_json",
				fmt.Sprintf("attributes_json must be a valid JSON object: %s", err),
			)
			return
		}
	}

	if !plan.Description.IsNull() {
		updateReq.Description = plan.Description.ValueString()
	}

	cs, err := r.client.UpdateCloudSync(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Cloud Sync Task",
			fmt.Sprintf("Could not update cloud sync task %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(cs, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update CloudSync success")
}

func (r *CloudSyncResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete CloudSync start")

	var state CloudSyncResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse cloud sync ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting cloud sync task", map[string]interface{}{"id": id})

	err = r.client.DeleteCloudSync(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Cloud sync task already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Cloud Sync Task",
			fmt.Sprintf("Could not delete cloud sync task %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete CloudSync success")
}

// ModifyPlan emits a plan-time Warning whenever the plan would
// destroy this resource, so operators see the destructive intent
// before running apply. Non-blocking (use destroy_protection for
// the blocking rail). See internal/planhelpers/destroy_warning.go.
func (r *CloudSyncResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_cloud_sync")
}

func (r *CloudSyncResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Cloud sync ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *CloudSyncResource) mapResponseToModel(cs *client.CloudSync, model *CloudSyncResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(cs.ID))
	model.Description = types.StringValue(cs.Description)
	model.Path = types.StringValue(cs.Path)
	model.Credentials = types.Int64Value(int64(cs.Credentials))
	model.Direction = types.StringValue(cs.Direction)
	model.TransferMode = types.StringValue(cs.TransferMode)
	model.Enabled = types.BoolValue(cs.Enabled)
	model.Minute = types.StringValue(cs.Schedule.Minute)
	model.Hour = types.StringValue(cs.Schedule.Hour)
	model.Dom = types.StringValue(cs.Schedule.Dom)
	model.Month = types.StringValue(cs.Schedule.Month)
	model.Dow = types.StringValue(cs.Schedule.Dow)

	// Attributes: preserve the user's key subset to avoid drift from
	// server-added defaults. Same pattern as reporting_exporter.
	// json.Marshal cannot fail on a map[string]interface{} from Unmarshal.
	attrsBytes, _ := json.Marshal(cs.Attributes)
	prior := model.AttributesJSON.ValueString()
	// filterJSONByKeys and normalizeJSON both operate on known-valid JSON
	// produced by the server decode and re-marshal, so neither can fail here.
	filtered, _ := filterJSONByKeys(string(attrsBytes), prior)
	canon, _ := normalizeJSON(filtered)
	model.AttributesJSON = types.StringValue(string(canon))
}
