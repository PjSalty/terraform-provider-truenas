package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
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
	_ resource.Resource                = &UPSConfigResource{}
	_ resource.ResourceWithImportState = &UPSConfigResource{}
)

// UPSConfigResource manages the TrueNAS UPS configuration.
type UPSConfigResource struct {
	client *client.Client
}

// UPSConfigResourceModel describes the resource data model.
type UPSConfigResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Mode          types.String   `tfsdk:"mode"`
	Identifier    types.String   `tfsdk:"identifier"`
	Driver        types.String   `tfsdk:"driver"`
	Port          types.String   `tfsdk:"port"`
	RemoteHost    types.String   `tfsdk:"remotehost"`
	RemotePort    types.Int64    `tfsdk:"remoteport"`
	Shutdown      types.String   `tfsdk:"shutdown"`
	ShutdownTimer types.Int64    `tfsdk:"shutdowntimer"`
	Description   types.String   `tfsdk:"description"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewUPSConfigResource() resource.Resource {
	return &UPSConfigResource{}
}

func (r *UPSConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ups_config"
}

func (r *UPSConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the UPS configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "UPS mode (MASTER or SLAVE).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("MASTER"),
				Validators: []validator.String{
					stringvalidator.OneOf("MASTER", "SLAVE"),
				},
			},
			"identifier": schema.StringAttribute{
				Description: "UPS identifier name.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("ups"),
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 120),
				},
			},
			"driver": schema.StringAttribute{
				Description: "UPS driver name.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(120),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port": schema.StringAttribute{
				Description: "UPS port or device path.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1023),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"remotehost": schema.StringAttribute{
				Description: "Remote UPS host (for SLAVE mode).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(253),
				},
			},
			"remoteport": schema.Int64Attribute{
				Description: "Remote UPS port.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3493),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"shutdown": schema.StringAttribute{
				Description: "Shutdown mode (BATT or LOWBATT).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("BATT"),
				Validators: []validator.String{
					stringvalidator.OneOf("BATT", "LOWBATT"),
				},
			},
			"shutdowntimer": schema.Int64Attribute{
				Description: "Shutdown timer in seconds (0-3600).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
				Validators: []validator.Int64{
					int64validator.Between(0, 3600),
				},
			},
			"description": schema.StringAttribute{
				Description: "UPS description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 1024),
				},
			},
		},
	}
}

func (r *UPSConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UPSConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create UPSConfig start")

	var plan UPSConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating UPS config resource (updating singleton)")

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateUPSConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating UPS Config",
			fmt.Sprintf("Could not update UPS configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create UPSConfig success")
}

func (r *UPSConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read UPSConfig start")

	var state UPSConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetUPSConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading UPS Config",
			fmt.Sprintf("Could not read UPS configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read UPSConfig success")
}

func (r *UPSConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update UPSConfig start")

	var plan UPSConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateUPSConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating UPS Config",
			fmt.Sprintf("Could not update UPS configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update UPSConfig success")
}

func (r *UPSConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete UPSConfig start")

	tflog.Debug(ctx, "Deleting UPS config resource (resetting to defaults)")

	mode := "MASTER"
	identifier := "ups"
	remotehost := ""
	remoteport := 3493
	shutdown := "BATT"
	shutdowntimer := 30
	description := ""

	// driver and port are not reset because the API rejects empty values
	_, err := r.client.UpdateUPSConfig(ctx, &client.UPSConfigUpdateRequest{
		Mode:          &mode,
		Identifier:    &identifier,
		RemoteHost:    &remotehost,
		RemotePort:    &remoteport,
		Shutdown:      &shutdown,
		ShutdownTimer: &shutdowntimer,
		Description:   &description,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting UPS Config",
			fmt.Sprintf("Could not reset UPS configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete UPSConfig success")
}

func (r *UPSConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *UPSConfigResource) buildUpdateRequest(plan *UPSConfigResourceModel) *client.UPSConfigUpdateRequest {
	updateReq := &client.UPSConfigUpdateRequest{}

	if !plan.Mode.IsNull() && !plan.Mode.IsUnknown() {
		v := plan.Mode.ValueString()
		updateReq.Mode = &v
	}
	if !plan.Identifier.IsNull() && !plan.Identifier.IsUnknown() {
		v := plan.Identifier.ValueString()
		updateReq.Identifier = &v
	}
	if !plan.Driver.IsNull() && !plan.Driver.IsUnknown() {
		v := plan.Driver.ValueString()
		if v != "" {
			updateReq.Driver = &v
		}
	}
	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		v := plan.Port.ValueString()
		if v != "" {
			updateReq.Port = &v
		}
	}
	if !plan.RemoteHost.IsNull() && !plan.RemoteHost.IsUnknown() {
		v := plan.RemoteHost.ValueString()
		updateReq.RemoteHost = &v
	}
	if !plan.RemotePort.IsNull() && !plan.RemotePort.IsUnknown() {
		v := int(plan.RemotePort.ValueInt64())
		updateReq.RemotePort = &v
	}
	if !plan.Shutdown.IsNull() && !plan.Shutdown.IsUnknown() {
		v := plan.Shutdown.ValueString()
		updateReq.Shutdown = &v
	}
	if !plan.ShutdownTimer.IsNull() && !plan.ShutdownTimer.IsUnknown() {
		v := int(plan.ShutdownTimer.ValueInt64())
		updateReq.ShutdownTimer = &v
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}

	return updateReq
}

func (r *UPSConfigResource) mapResponseToModel(config *client.UPSConfig, model *UPSConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.Mode = types.StringValue(config.Mode)
	model.Identifier = types.StringValue(config.Identifier)
	model.Driver = types.StringValue(config.Driver)
	model.Port = types.StringValue(config.Port)
	model.RemoteHost = types.StringValue(config.RemoteHost)
	model.RemotePort = types.Int64Value(int64(config.RemotePort))
	model.Shutdown = types.StringValue(config.Shutdown)
	model.ShutdownTimer = types.Int64Value(int64(config.ShutdownTimer))
	model.Description = types.StringValue(config.Description)
}
