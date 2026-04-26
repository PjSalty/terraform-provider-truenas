package resources

import (
	"context"
	"fmt"
	"regexp"

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
)

var (
	_ resource.Resource                = &FTPConfigResource{}
	_ resource.ResourceWithImportState = &FTPConfigResource{}
)

// FTPConfigResource manages the TrueNAS FTP service configuration.
type FTPConfigResource struct {
	client *client.Client
}

// FTPConfigResourceModel describes the resource data model.
type FTPConfigResourceModel struct {
	ID            types.String   `tfsdk:"id"`
	Port          types.Int64    `tfsdk:"port"`
	Clients       types.Int64    `tfsdk:"clients"`
	IPConnections types.Int64    `tfsdk:"ipconnections"`
	LoginAttempt  types.Int64    `tfsdk:"loginattempt"`
	Timeout       types.Int64    `tfsdk:"timeout"`
	OnlyAnonymous types.Bool     `tfsdk:"onlyanonymous"`
	OnlyLocal     types.Bool     `tfsdk:"onlylocal"`
	Banner        types.String   `tfsdk:"banner"`
	Filemask      types.String   `tfsdk:"filemask"`
	Dirmask       types.String   `tfsdk:"dirmask"`
	FXP           types.Bool     `tfsdk:"fxp"`
	Resume        types.Bool     `tfsdk:"resume"`
	DefaultRoot   types.Bool     `tfsdk:"defaultroot"`
	TLS           types.Bool     `tfsdk:"tls"`
	Timeouts      timeouts.Value `tfsdk:"timeouts"`
}

func NewFTPConfigResource() resource.Resource {
	return &FTPConfigResource{}
}

func (r *FTPConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ftp_config"
}

func (r *FTPConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the FTP service configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"port": schema.Int64Attribute{
				Description: "FTP port.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(21),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"clients": schema.Int64Attribute{
				Description: "Maximum number of simultaneous clients (1-10000).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
				Validators: []validator.Int64{
					int64validator.Between(1, 10000),
				},
			},
			"ipconnections": schema.Int64Attribute{
				Description: "Maximum connections per IP address (0 = unlimited, max 1000).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.Between(0, 1000),
				},
			},
			"loginattempt": schema.Int64Attribute{
				Description: "Maximum login attempts before disconnect (1-1000).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 1000),
				},
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout in seconds for idle connections (1-10000).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(600),
				Validators: []validator.Int64{
					int64validator.Between(1, 10000),
				},
			},
			"onlyanonymous": schema.BoolAttribute{
				Description: "Allow only anonymous logins.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"onlylocal": schema.BoolAttribute{
				Description: "Allow only local user logins.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"banner": schema.StringAttribute{
				Description: "FTP banner message.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(4096),
				},
			},
			"filemask": schema.StringAttribute{
				Description: "File creation mask (umask), octal (e.g. 077).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("077"),
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-7]{3,4}$`),
						"must be a 3- or 4-digit octal umask value",
					),
				},
			},
			"dirmask": schema.StringAttribute{
				Description: "Directory creation mask (umask), octal (e.g. 022).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("022"),
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[0-7]{3,4}$`),
						"must be a 3- or 4-digit octal umask value",
					),
				},
			},
			"fxp": schema.BoolAttribute{
				Description: "Enable FXP (File eXchange Protocol).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"resume": schema.BoolAttribute{
				Description: "Allow transfer resume.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"defaultroot": schema.BoolAttribute{
				Description: "Chroot users to their home directory.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"tls": schema.BoolAttribute{
				Description: "Enable TLS for FTP connections.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *FTPConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FTPConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create FTPConfig start")

	var plan FTPConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating FTP config resource (updating singleton)")

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateFTPConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating FTP Config",
			fmt.Sprintf("Could not update FTP configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create FTPConfig success")
}

func (r *FTPConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read FTPConfig start")

	var state FTPConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetFTPConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading FTP Config",
			fmt.Sprintf("Could not read FTP configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read FTPConfig success")
}

func (r *FTPConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update FTPConfig start")

	var plan FTPConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan)

	config, err := r.client.UpdateFTPConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating FTP Config",
			fmt.Sprintf("Could not update FTP configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update FTPConfig success")
}

func (r *FTPConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete FTPConfig start")

	tflog.Debug(ctx, "Deleting FTP config resource (resetting to defaults)")

	port := 21
	clients := 5
	ipconnections := 2
	loginattempt := 1
	timeout := 600
	onlyanonymous := false
	onlylocal := false
	banner := ""
	filemask := "077"
	dirmask := "022"
	fxp := false
	resume := false
	defaultroot := true
	tls := false

	_, err := r.client.UpdateFTPConfig(ctx, &client.FTPConfigUpdateRequest{
		Port:          &port,
		Clients:       &clients,
		IPConnections: &ipconnections,
		LoginAttempt:  &loginattempt,
		Timeout:       &timeout,
		OnlyAnonymous: &onlyanonymous,
		OnlyLocal:     &onlylocal,
		Banner:        &banner,
		Filemask:      &filemask,
		Dirmask:       &dirmask,
		FXP:           &fxp,
		Resume:        &resume,
		DefaultRoot:   &defaultroot,
		TLS:           &tls,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting FTP Config",
			fmt.Sprintf("Could not reset FTP configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete FTPConfig success")
}

func (r *FTPConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *FTPConfigResource) buildUpdateRequest(plan *FTPConfigResourceModel) *client.FTPConfigUpdateRequest {
	updateReq := &client.FTPConfigUpdateRequest{}

	if !plan.Port.IsNull() && !plan.Port.IsUnknown() {
		v := int(plan.Port.ValueInt64())
		updateReq.Port = &v
	}
	if !plan.Clients.IsNull() && !plan.Clients.IsUnknown() {
		v := int(plan.Clients.ValueInt64())
		updateReq.Clients = &v
	}
	if !plan.IPConnections.IsNull() && !plan.IPConnections.IsUnknown() {
		v := int(plan.IPConnections.ValueInt64())
		updateReq.IPConnections = &v
	}
	if !plan.LoginAttempt.IsNull() && !plan.LoginAttempt.IsUnknown() {
		v := int(plan.LoginAttempt.ValueInt64())
		updateReq.LoginAttempt = &v
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		v := int(plan.Timeout.ValueInt64())
		updateReq.Timeout = &v
	}
	if !plan.OnlyAnonymous.IsNull() && !plan.OnlyAnonymous.IsUnknown() {
		v := plan.OnlyAnonymous.ValueBool()
		updateReq.OnlyAnonymous = &v
	}
	if !plan.OnlyLocal.IsNull() && !plan.OnlyLocal.IsUnknown() {
		v := plan.OnlyLocal.ValueBool()
		updateReq.OnlyLocal = &v
	}
	if !plan.Banner.IsNull() && !plan.Banner.IsUnknown() {
		v := plan.Banner.ValueString()
		updateReq.Banner = &v
	}
	if !plan.Filemask.IsNull() && !plan.Filemask.IsUnknown() {
		v := plan.Filemask.ValueString()
		updateReq.Filemask = &v
	}
	if !plan.Dirmask.IsNull() && !plan.Dirmask.IsUnknown() {
		v := plan.Dirmask.ValueString()
		updateReq.Dirmask = &v
	}
	if !plan.FXP.IsNull() && !plan.FXP.IsUnknown() {
		v := plan.FXP.ValueBool()
		updateReq.FXP = &v
	}
	if !plan.Resume.IsNull() && !plan.Resume.IsUnknown() {
		v := plan.Resume.ValueBool()
		updateReq.Resume = &v
	}
	if !plan.DefaultRoot.IsNull() && !plan.DefaultRoot.IsUnknown() {
		v := plan.DefaultRoot.ValueBool()
		updateReq.DefaultRoot = &v
	}
	if !plan.TLS.IsNull() && !plan.TLS.IsUnknown() {
		v := plan.TLS.ValueBool()
		updateReq.TLS = &v
	}

	return updateReq
}

func (r *FTPConfigResource) mapResponseToModel(config *client.FTPConfig, model *FTPConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.Port = types.Int64Value(int64(config.Port))
	model.Clients = types.Int64Value(int64(config.Clients))
	model.IPConnections = types.Int64Value(int64(config.IPConnections))
	model.LoginAttempt = types.Int64Value(int64(config.LoginAttempt))
	model.Timeout = types.Int64Value(int64(config.Timeout))
	model.OnlyAnonymous = types.BoolValue(config.OnlyAnonymous)
	model.OnlyLocal = types.BoolValue(config.OnlyLocal)
	model.Banner = types.StringValue(config.Banner)
	model.Filemask = types.StringValue(config.Filemask)
	model.Dirmask = types.StringValue(config.Dirmask)
	model.FXP = types.BoolValue(config.FXP)
	model.Resume = types.BoolValue(config.Resume)
	model.DefaultRoot = types.BoolValue(config.DefaultRoot)
	model.TLS = types.BoolValue(config.TLS)
}
