// kmip_config.go
//
// Singleton resource (/kmip is PUT-only). Create and Update both delegate to
// the same PUT payload. Delete resets to disabled/empty defaults so that
// `terraform destroy` reverts the configuration.
package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
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

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &KMIPConfigResource{}
	_ resource.ResourceWithImportState = &KMIPConfigResource{}
)

type KMIPConfigResource struct {
	client *client.Client
}

type KMIPConfigResourceModel struct {
	ID                   types.String   `tfsdk:"id"`
	Enabled              types.Bool     `tfsdk:"enabled"`
	ManageSEDDisks       types.Bool     `tfsdk:"manage_sed_disks"`
	ManageZFSKeys        types.Bool     `tfsdk:"manage_zfs_keys"`
	Certificate          types.Int64    `tfsdk:"certificate"`
	CertificateAuthority types.Int64    `tfsdk:"certificate_authority"`
	Port                 types.Int64    `tfsdk:"port"`
	Server               types.String   `tfsdk:"server"`
	SSLVersion           types.String   `tfsdk:"ssl_version"`
	ChangeServer         types.Bool     `tfsdk:"change_server"`
	Validate             types.Bool     `tfsdk:"validate"`
	ForceClear           types.Bool     `tfsdk:"force_clear"`
	Timeouts             timeouts.Value `tfsdk:"timeouts"`
}

func NewKMIPConfigResource() resource.Resource {
	return &KMIPConfigResource{}
}

func (r *KMIPConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kmip_config"
}

func (r *KMIPConfigResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the KMIP (Key Management Interoperability Protocol) singleton configuration on TrueNAS SCALE.",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether KMIP functionality is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"manage_sed_disks": schema.BoolAttribute{
				Description: "Use KMIP to manage Self-Encrypting Drive keys.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"manage_zfs_keys": schema.BoolAttribute{
				Description: "Use KMIP to manage ZFS encryption keys.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"certificate": schema.Int64Attribute{
				Description: "ID of the client certificate used for KMIP authentication (0 = none).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"certificate_authority": schema.Int64Attribute{
				Description: "ID of the CA used to verify the KMIP server (0 = none).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"port": schema.Int64Attribute{
				Description: "TCP port for the KMIP server connection.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5696),
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"server": schema.StringAttribute{
				Description: "Hostname or IP of the KMIP server. Empty string disables.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthBetween(0, 253),
				},
			},
			"ssl_version": schema.StringAttribute{
				Description: "SSL/TLS protocol version. One of PROTOCOL_TLSv1, PROTOCOL_TLSv1_1, PROTOCOL_TLSv1_2.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("PROTOCOL_TLSv1_2"),
				Validators: []validator.String{
					stringvalidator.OneOf("PROTOCOL_TLSv1", "PROTOCOL_TLSv1_1", "PROTOCOL_TLSv1_2"),
				},
			},
			"change_server": schema.BoolAttribute{
				Description: "Flag indicating the KMIP server endpoint is being changed.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"validate": schema.BoolAttribute{
				Description: "Validate the KMIP server connection before saving.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"force_clear": schema.BoolAttribute{
				Description: "Force clear existing keys when disabling KMIP.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *KMIPConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *KMIPConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create KMIPConfig start")

	var plan KMIPConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating KMIP config (updating singleton)")

	cfg, err := r.client.UpdateKMIPConfig(ctx, r.buildUpdateRequest(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating KMIP Config", fmt.Sprintf("Could not update KMIP configuration: %s", err))
		return
	}

	r.mapResponseToModel(cfg, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create KMIPConfig success")
}

func (r *KMIPConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read KMIPConfig start")

	var state KMIPConfigResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetKMIPConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading KMIP Config", fmt.Sprintf("Could not read KMIP configuration: %s", err))
		return
	}

	r.mapResponseToModel(cfg, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read KMIPConfig success")
}

func (r *KMIPConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update KMIPConfig start")

	var plan KMIPConfigResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateKMIPConfig(ctx, r.buildUpdateRequest(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error Updating KMIP Config", fmt.Sprintf("Could not update KMIP configuration: %s", err))
		return
	}

	r.mapResponseToModel(cfg, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update KMIPConfig success")
}

func (r *KMIPConfigResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete KMIPConfig start")

	tflog.Debug(ctx, "Deleting KMIP config resource (resetting to defaults)")

	enabled := false
	manageSED := false
	manageZFS := false
	port := 5696
	server := ""
	sslVersion := "PROTOCOL_TLSv1_2"
	forceClear := true

	_, err := r.client.UpdateKMIPConfig(ctx, &client.KMIPUpdateRequest{
		Enabled:        &enabled,
		ManageSEDDisks: &manageSED,
		ManageZFSKeys:  &manageZFS,
		Port:           &port,
		Server:         &server,
		SSLVersion:     &sslVersion,
		ForceClear:     &forceClear,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error Resetting KMIP Config", fmt.Sprintf("Could not reset KMIP configuration: %s", err))
		return
	}
	tflog.Trace(ctx, "Delete KMIPConfig success")
}

func (r *KMIPConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("change_server"), types.BoolValue(false))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("validate"), types.BoolValue(true))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("force_clear"), types.BoolValue(false))...)
}

func (r *KMIPConfigResource) buildUpdateRequest(plan *KMIPConfigResourceModel) *client.KMIPUpdateRequest {
	enabled := plan.Enabled.ValueBool()
	manageSED := plan.ManageSEDDisks.ValueBool()
	manageZFS := plan.ManageZFSKeys.ValueBool()
	port := int(plan.Port.ValueInt64())
	server := plan.Server.ValueString()
	sslVersion := plan.SSLVersion.ValueString()
	changeServer := plan.ChangeServer.ValueBool()
	validate := plan.Validate.ValueBool()
	forceClear := plan.ForceClear.ValueBool()

	req := &client.KMIPUpdateRequest{
		Enabled:        &enabled,
		ManageSEDDisks: &manageSED,
		ManageZFSKeys:  &manageZFS,
		Port:           &port,
		Server:         &server,
		SSLVersion:     &sslVersion,
		ChangeServer:   &changeServer,
		Validate:       &validate,
		ForceClear:     &forceClear,
	}

	if !plan.Certificate.IsNull() && !plan.Certificate.IsUnknown() {
		v := int(plan.Certificate.ValueInt64())
		if v != 0 {
			req.Certificate = &v
		}
	}
	if !plan.CertificateAuthority.IsNull() && !plan.CertificateAuthority.IsUnknown() {
		v := int(plan.CertificateAuthority.ValueInt64())
		if v != 0 {
			req.CertificateAuthority = &v
		}
	}

	return req
}

func (r *KMIPConfigResource) mapResponseToModel(cfg *client.KMIPConfig, model *KMIPConfigResourceModel) {
	model.ID = types.StringValue("kmip")
	model.Enabled = types.BoolValue(cfg.Enabled)
	model.ManageSEDDisks = types.BoolValue(cfg.ManageSEDDisks)
	model.ManageZFSKeys = types.BoolValue(cfg.ManageZFSKeys)
	if cfg.Certificate != nil {
		model.Certificate = types.Int64Value(int64(*cfg.Certificate))
	} else {
		model.Certificate = types.Int64Value(0)
	}
	if cfg.CertificateAuthority != nil {
		model.CertificateAuthority = types.Int64Value(int64(*cfg.CertificateAuthority))
	} else {
		model.CertificateAuthority = types.Int64Value(0)
	}
	model.Port = types.Int64Value(int64(cfg.Port))
	if cfg.Server != nil {
		model.Server = types.StringValue(*cfg.Server)
	} else {
		model.Server = types.StringValue("")
	}
	model.SSLVersion = types.StringValue(cfg.SSLVersion)
	// Preserve any plan-only flags already set on the model (change_server etc.).
	if model.ChangeServer.IsNull() || model.ChangeServer.IsUnknown() {
		model.ChangeServer = types.BoolValue(false)
	}
	if model.Validate.IsNull() || model.Validate.IsUnknown() {
		model.Validate = types.BoolValue(true)
	}
	if model.ForceClear.IsNull() || model.ForceClear.IsUnknown() {
		model.ForceClear = types.BoolValue(false)
	}
}
