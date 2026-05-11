package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
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
	_ resource.Resource                = &SSHConfigResource{}
	_ resource.ResourceWithImportState = &SSHConfigResource{}
)

// SSHConfigResource manages the TrueNAS SSH service configuration.
type SSHConfigResource struct {
	client *client.Client
}

// SSHConfigResourceModel describes the resource data model.
type SSHConfigResourceModel struct {
	ID              types.String   `tfsdk:"id"`
	TCPPort         types.Int64    `tfsdk:"tcpport"`
	PasswordAuth    types.Bool     `tfsdk:"passwordauth"`
	KerberosAuth    types.Bool     `tfsdk:"kerberosauth"`
	TCPFwd          types.Bool     `tfsdk:"tcpfwd"`
	Compression     types.Bool     `tfsdk:"compression"`
	SFTPLogLevel    types.String   `tfsdk:"sftp_log_level"`
	SFTPLogFacility types.String   `tfsdk:"sftp_log_facility"`
	WeakCiphers     types.List     `tfsdk:"weak_ciphers"`
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
}

func NewSSHConfigResource() resource.Resource {
	return &SSHConfigResource{}
}

func (r *SSHConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_config"
}

func (r *SSHConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{Blocks: map[string]schema.Block{"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true, Read: true, Update: true, Delete: true})}, Description: "Manages the SSH service configuration on TrueNAS SCALE. " +
		"This is a singleton resource — only one instance can exist.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The configuration ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tcpport": schema.Int64Attribute{
				Description: "TCP port for the SSH service.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(22),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"passwordauth": schema.BoolAttribute{
				Description: "Allow password authentication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"kerberosauth": schema.BoolAttribute{
				Description: "Allow Kerberos authentication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"tcpfwd": schema.BoolAttribute{
				Description: "Allow TCP port forwarding.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"compression": schema.BoolAttribute{
				Description: "Enable compression.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"sftp_log_level": schema.StringAttribute{
				Description: "SFTP log level (QUIET, FATAL, ERROR, INFO, VERBOSE, DEBUG, DEBUG2, DEBUG3).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.OneOf("", "QUIET", "FATAL", "ERROR", "INFO", "VERBOSE", "DEBUG", "DEBUG2", "DEBUG3"),
				},
			},
			"sftp_log_facility": schema.StringAttribute{
				Description: "SFTP log facility (DAEMON, USER, AUTH, LOCAL0-7).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.OneOf("", "DAEMON", "USER", "AUTH", "LOCAL0", "LOCAL1", "LOCAL2", "LOCAL3", "LOCAL4", "LOCAL5", "LOCAL6", "LOCAL7"),
				},
			},
			"weak_ciphers": schema.ListAttribute{
				Description: "List of weak ciphers to allow (e.g., AES128-CBC, NONE).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
		},
	}
}

func (r *SSHConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SSHConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create SSHConfig start")

	var plan SSHConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating SSH config resource (updating singleton)")

	var d diag.Diagnostics
	updateReq := r.buildUpdateRequest(ctx, &plan, &d)
	resp.Diagnostics.Append(d...)

	config, err := r.client.UpdateSSHConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SSH Config",
			fmt.Sprintf("Could not update SSH configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create SSHConfig success")
}

func (r *SSHConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read SSHConfig start")

	var state SSHConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	config, err := r.client.GetSSHConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SSH Config",
			fmt.Sprintf("Could not read SSH configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read SSHConfig success")
}

func (r *SSHConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update SSHConfig start")

	var plan SSHConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var d diag.Diagnostics
	updateReq := r.buildUpdateRequest(ctx, &plan, &d)
	resp.Diagnostics.Append(d...)

	config, err := r.client.UpdateSSHConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating SSH Config",
			fmt.Sprintf("Could not update SSH configuration: %s", err),
		)
		return
	}

	r.mapResponseToModel(ctx, config, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update SSHConfig success")
}

func (r *SSHConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete SSHConfig start")

	tflog.Debug(ctx, "Deleting SSH config resource (resetting to defaults)")

	tcpport := 22
	passwordauth := true
	kerberosauth := false
	tcpfwd := false
	compression := false
	sftpLogLevel := ""
	sftpLogFacility := ""

	_, err := r.client.UpdateSSHConfig(ctx, &client.SSHConfigUpdateRequest{
		TCPPort:         &tcpport,
		PasswordAuth:    &passwordauth,
		KerberosAuth:    &kerberosauth,
		TCPFwd:          &tcpfwd,
		Compression:     &compression,
		SFTPLogLevel:    &sftpLogLevel,
		SFTPLogFacility: &sftpLogFacility,
		WeakCiphers:     &[]string{},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resetting SSH Config",
			fmt.Sprintf("Could not reset SSH configuration to defaults: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete SSHConfig success")
}

func (r *SSHConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *SSHConfigResource) buildUpdateRequest(ctx context.Context, plan *SSHConfigResourceModel, d *diag.Diagnostics) *client.SSHConfigUpdateRequest {
	updateReq := &client.SSHConfigUpdateRequest{}

	if !plan.TCPPort.IsNull() && !plan.TCPPort.IsUnknown() {
		v := int(plan.TCPPort.ValueInt64())
		updateReq.TCPPort = &v
	}
	if !plan.PasswordAuth.IsNull() && !plan.PasswordAuth.IsUnknown() {
		v := plan.PasswordAuth.ValueBool()
		updateReq.PasswordAuth = &v
	}
	if !plan.KerberosAuth.IsNull() && !plan.KerberosAuth.IsUnknown() {
		v := plan.KerberosAuth.ValueBool()
		updateReq.KerberosAuth = &v
	}
	if !plan.TCPFwd.IsNull() && !plan.TCPFwd.IsUnknown() {
		v := plan.TCPFwd.ValueBool()
		updateReq.TCPFwd = &v
	}
	if !plan.Compression.IsNull() && !plan.Compression.IsUnknown() {
		v := plan.Compression.ValueBool()
		updateReq.Compression = &v
	}
	if !plan.SFTPLogLevel.IsNull() && !plan.SFTPLogLevel.IsUnknown() {
		v := plan.SFTPLogLevel.ValueString()
		updateReq.SFTPLogLevel = &v
	}
	if !plan.SFTPLogFacility.IsNull() && !plan.SFTPLogFacility.IsUnknown() {
		v := plan.SFTPLogFacility.ValueString()
		updateReq.SFTPLogFacility = &v
	}

	if !plan.WeakCiphers.IsNull() && !plan.WeakCiphers.IsUnknown() {
		var weakCiphers []string
		diags := plan.WeakCiphers.ElementsAs(ctx, &weakCiphers, false)
		d.Append(diags...)
		updateReq.WeakCiphers = &weakCiphers
	}

	return updateReq
}

func (r *SSHConfigResource) mapResponseToModel(ctx context.Context, config *client.SSHConfig, model *SSHConfigResourceModel) {
	model.ID = types.StringValue("1")
	model.TCPPort = types.Int64Value(int64(config.TCPPort))
	model.PasswordAuth = types.BoolValue(config.PasswordAuth)
	model.KerberosAuth = types.BoolValue(config.KerberosAuth)
	model.TCPFwd = types.BoolValue(config.TCPFwd)
	model.Compression = types.BoolValue(config.Compression)
	model.SFTPLogLevel = types.StringValue(config.SFTPLogLevel)
	model.SFTPLogFacility = types.StringValue(config.SFTPLogFacility)

	weakCipherValues, diags := types.ListValueFrom(ctx, types.StringType, config.WeakCiphers)
	if !diags.HasError() {
		model.WeakCiphers = weakCipherValues
	}
}
