package resources

import (
	"context"
	"encoding/json"
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
)

var (
	_ resource.Resource                = &AlertServiceResource{}
	_ resource.ResourceWithImportState = &AlertServiceResource{}
)

// AlertServiceResource manages a TrueNAS alert service.
type AlertServiceResource struct {
	client *client.Client
}

// AlertServiceResourceModel describes the resource data model.
type AlertServiceResourceModel struct {
	ID       types.String   `tfsdk:"id"`
	Name     types.String   `tfsdk:"name"`
	Type     types.String   `tfsdk:"type"`
	Enabled  types.Bool     `tfsdk:"enabled"`
	Level    types.String   `tfsdk:"level"`
	Settings types.String   `tfsdk:"settings_json"`
	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewAlertServiceResource() resource.Resource {
	return &AlertServiceResource{}
}

func (r *AlertServiceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_service"
}

func (r *AlertServiceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages an alert service on TrueNAS SCALE (email, Pushover, Slack, etc.).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the alert service.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the alert service.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"type": schema.StringAttribute{
				Description: "The alert service type (Mail, AWSSNS, InfluxDB, Mattermost, OpsGenie, PagerDuty, PushBullet, PushOver, Slack, SNMPTrap, Telegram, VictorOps).",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"AWSSNS", "InfluxDB", "Mail", "Mattermost",
						"OpsGenie", "PagerDuty", "PushBullet", "PushOver",
						"Slack", "SNMPTrap", "Telegram", "VictorOps",
					),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the alert service is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"level": schema.StringAttribute{
				Description: "Minimum alert level (INFO, NOTICE, WARNING, ERROR, CRITICAL, ALERT, EMERGENCY).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("WARNING"),
				Validators: []validator.String{
					stringvalidator.OneOf("INFO", "NOTICE", "WARNING", "ERROR", "CRITICAL", "ALERT", "EMERGENCY"),
				},
			},
			"settings_json": schema.StringAttribute{
				Description: "Service-specific settings as a JSON string. " +
					"The structure depends on the service type.",
				Required:  true,
				Sensitive: true,
			},
		},
	}
}

func (r *AlertServiceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AlertServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create AlertService start")

	var plan AlertServiceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Settings.ValueString()), &settings); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Settings JSON",
			fmt.Sprintf("Could not parse settings_json: %s", err),
		)
		return
	}

	// The TrueNAS API expects the type inside the attributes map.
	settings["type"] = plan.Type.ValueString()

	createReq := &client.AlertServiceCreateRequest{
		Name:     plan.Name.ValueString(),
		Enabled:  plan.Enabled.ValueBool(),
		Level:    plan.Level.ValueString(),
		Settings: settings,
	}

	tflog.Debug(ctx, "Creating alert service", map[string]interface{}{
		"name": plan.Name.ValueString(),
		"type": plan.Type.ValueString(),
	})

	svc, err := r.client.CreateAlertService(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Alert Service",
			fmt.Sprintf("Could not create alert service %q: %s", plan.Name.ValueString(), err),
		)
		return
	}

	r.mapResponseToModel(svc, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create AlertService success")
}

func (r *AlertServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read AlertService start")

	var state AlertServiceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse alert service ID: %s", err))
		return
	}

	svc, err := r.client.GetAlertService(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Alert Service",
			fmt.Sprintf("Could not read alert service %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(svc, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read AlertService success")
}

func (r *AlertServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update AlertService start")

	var plan AlertServiceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AlertServiceResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse alert service ID: %s", err))
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(plan.Settings.ValueString()), &settings); err != nil {
		resp.Diagnostics.AddError(
			"Invalid Settings JSON",
			fmt.Sprintf("Could not parse settings_json: %s", err),
		)
		return
	}

	enabled := plan.Enabled.ValueBool()

	// The TrueNAS API expects the type inside the attributes map.
	settings["type"] = plan.Type.ValueString()

	updateReq := &client.AlertServiceUpdateRequest{
		Name:     plan.Name.ValueString(),
		Enabled:  &enabled,
		Level:    plan.Level.ValueString(),
		Settings: settings,
	}

	svc, err := r.client.UpdateAlertService(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Alert Service",
			fmt.Sprintf("Could not update alert service %d: %s", id, err),
		)
		return
	}

	r.mapResponseToModel(svc, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update AlertService success")
}

func (r *AlertServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete AlertService start")

	var state AlertServiceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse alert service ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting alert service", map[string]interface{}{"id": id})

	err = r.client.DeleteAlertService(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "Alert service already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting Alert Service",
			fmt.Sprintf("Could not delete alert service %d: %s", id, err),
		)
		return
	}
	tflog.Trace(ctx, "Delete AlertService success")
}

func (r *AlertServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Alert service ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *AlertServiceResource) mapResponseToModel(svc *client.AlertService, model *AlertServiceResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(svc.ID))
	model.Name = types.StringValue(svc.Name)
	model.Type = types.StringValue(svc.GetType())
	model.Enabled = types.BoolValue(svc.Enabled)
	model.Level = types.StringValue(svc.Level)

	if svc.Settings != nil {
		// Remove the "type" key from settings before serializing,
		// as it is managed as a separate Terraform attribute.
		settingsCopy := make(map[string]interface{}, len(svc.Settings))
		for k, v := range svc.Settings {
			if k != "type" {
				settingsCopy[k] = v
			}
		}
		settingsJSON, err := json.Marshal(settingsCopy)
		if err == nil {
			model.Settings = types.StringValue(string(settingsJSON))
		}
	}
}
