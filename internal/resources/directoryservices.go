package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var (
	_ resource.Resource                = &DirectoryServicesResource{}
	_ resource.ResourceWithImportState = &DirectoryServicesResource{}
)

// DirectoryServicesResource manages the TrueNAS directory services singleton
// (Active Directory, IPA, or LDAP binding). Only one directory service can
// be configured at a time. Delete disables the service rather than removing
// the underlying singleton record.
type DirectoryServicesResource struct {
	client *client.Client
}

// DirectoryServicesResourceModel describes the resource data model.
//
// credential_json and configuration_json are JSON-encoded blobs because the
// underlying API uses polymorphic schemas whose shape depends on
// service_type and credential_type. Users are expected to pass the exact
// JSON payload from the TrueNAS API docs, e.g. via jsonencode().
type DirectoryServicesResourceModel struct {
	ID                 types.String   `tfsdk:"id"`
	ServiceType        types.String   `tfsdk:"service_type"`
	Enable             types.Bool     `tfsdk:"enable"`
	EnableAccountCache types.Bool     `tfsdk:"enable_account_cache"`
	EnableDNSUpdates   types.Bool     `tfsdk:"enable_dns_updates"`
	Timeout            types.Int64    `tfsdk:"timeout"`
	KerberosRealm      types.String   `tfsdk:"kerberos_realm"`
	CredentialJSON     types.String   `tfsdk:"credential_json"`
	ConfigurationJSON  types.String   `tfsdk:"configuration_json"`
	Force              types.Bool     `tfsdk:"force"`
	Timeouts           timeouts.Value `tfsdk:"timeouts"`
}

func NewDirectoryServicesResource() resource.Resource {
	return &DirectoryServicesResource{}
}

func (r *DirectoryServicesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directoryservices"
}

func (r *DirectoryServicesResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the TrueNAS directory services singleton configuration " +
			"(ACTIVEDIRECTORY, IPA, or LDAP). Only one directory service can be " +
			"active at a time. Creating this resource configures the service; " +
			"deleting it disables the service (it does not unjoin from AD — use " +
			"the TrueNAS UI/CLI or `enable = false` first)." + "\n\n" +
			"**Stability: Alpha.** Not end-to-end verified — requires a real Active Directory, LDAP, or FreeIPA server. Schema matches the TrueNAS REST API.",
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
				Description: "Singleton ID (always 1).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_type": schema.StringAttribute{
				Description: "One of ACTIVEDIRECTORY, IPA, LDAP. Set to empty string to disable.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("", "ACTIVEDIRECTORY", "IPA", "LDAP"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enable": schema.BoolAttribute{
				Description: "Whether the directory service is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_account_cache": schema.BoolAttribute{
				Description: "Enable backend caching for user and group lists.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"enable_dns_updates": schema.BoolAttribute{
				Description: "Enable automatic DNS updates via nsupdate.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout (seconds) for DNS queries performed during domain join (1-600).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10),
				Validators: []validator.Int64{
					int64validator.Between(1, 600),
				},
			},
			"kerberos_realm": schema.StringAttribute{
				Description: "Name of the Kerberos realm used for authentication. Required for " +
					"ACTIVEDIRECTORY and IPA; optional for LDAP.",
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					stringvalidator.LengthAtMost(253),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"credential_json": schema.StringAttribute{
				Description: "JSON-encoded credential object. Shape depends on credential_type " +
					"(KERBEROS_USER, KERBEROS_PRINCIPAL, LDAP_PLAIN, LDAP_MTLS, LDAP_ANONYMOUS). " +
					"Example: jsonencode({ credential_type = \"KERBEROS_USER\", username = \"admin\", password = \"...\" }).",
				Optional:  true,
				Sensitive: true,
			},
			"configuration_json": schema.StringAttribute{
				Description: "JSON-encoded service_type-specific configuration (domain, hostname, " +
					"base_dn, server_urls, etc.). See the TrueNAS API docs for required fields.",
				Optional: true,
			},
			"force": schema.BoolAttribute{
				Description: "Bypass validation that checks if a server with this hostname or " +
					"NetBIOS name is already joined. Use with caution.",
				Optional: true,
			},
		},
	}
}

func (r *DirectoryServicesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DirectoryServicesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create DirectoryServices start")

	var plan DirectoryServicesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Configuring directory services singleton")

	updateReq := r.buildUpdateRequest(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateDirectoryServicesConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Configuring Directory Services",
			fmt.Sprintf("Could not update directory services: %s", err),
		)
		return
	}

	r.mapResponseToModel(cfg, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Create DirectoryServices success")
}

func (r *DirectoryServicesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read DirectoryServices start")

	var state DirectoryServicesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetDirectoryServicesConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Directory Services",
			fmt.Sprintf("Could not read directory services config: %s", err),
		)
		return
	}

	r.mapResponseToModel(cfg, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	tflog.Trace(ctx, "Read DirectoryServices success")
}

func (r *DirectoryServicesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update DirectoryServices start")

	var plan DirectoryServicesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := r.buildUpdateRequest(&plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateDirectoryServicesConfig(ctx, updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Directory Services",
			fmt.Sprintf("Could not update directory services: %s", err),
		)
		return
	}

	r.mapResponseToModel(cfg, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	tflog.Trace(ctx, "Update DirectoryServices success")
}

func (r *DirectoryServicesResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete DirectoryServices start")

	tflog.Debug(ctx, "Disabling directory services (delete = disable singleton)")

	// Best-effort leave if currently ACTIVEDIRECTORY. We do not fail delete
	// on leave errors because the singleton may already be in a partially
	// disabled state.
	cur, err := r.client.GetDirectoryServicesConfig(ctx)
	if err == nil && cur != nil && cur.ServiceType != nil && *cur.ServiceType == "ACTIVEDIRECTORY" && cur.Enable {
		if _, lerr := r.client.LeaveDirectoryServices(ctx, map[string]interface{}{}); lerr != nil {
			tflog.Warn(ctx, "Directory services leave failed during delete", map[string]interface{}{"error": lerr.Error()})
		}
	}

	disabled := false
	emptyType := ""
	resetReq := &client.DirectoryServicesUpdateRequest{
		ServiceType: &emptyType,
		Enable:      &disabled,
	}
	if _, err := r.client.UpdateDirectoryServicesConfig(ctx, resetReq); err != nil {
		resp.Diagnostics.AddError(
			"Error Disabling Directory Services",
			fmt.Sprintf("Could not disable directory services: %s", err),
		)
		return
	}
	tflog.Trace(ctx, "Delete DirectoryServices success")
}

func (r *DirectoryServicesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DirectoryServicesResource) buildUpdateRequest(plan *DirectoryServicesResourceModel, diags *diag.Diagnostics) *client.DirectoryServicesUpdateRequest {
	out := &client.DirectoryServicesUpdateRequest{}

	if !plan.ServiceType.IsNull() && !plan.ServiceType.IsUnknown() {
		v := plan.ServiceType.ValueString()
		out.ServiceType = &v
	}
	if !plan.Enable.IsNull() && !plan.Enable.IsUnknown() {
		v := plan.Enable.ValueBool()
		out.Enable = &v
	}
	if !plan.EnableAccountCache.IsNull() && !plan.EnableAccountCache.IsUnknown() {
		v := plan.EnableAccountCache.ValueBool()
		out.EnableAccountCache = &v
	}
	if !plan.EnableDNSUpdates.IsNull() && !plan.EnableDNSUpdates.IsUnknown() {
		v := plan.EnableDNSUpdates.ValueBool()
		out.EnableDNSUpdates = &v
	}
	if !plan.Timeout.IsNull() && !plan.Timeout.IsUnknown() {
		v := int(plan.Timeout.ValueInt64())
		out.Timeout = &v
	}
	if !plan.KerberosRealm.IsNull() && !plan.KerberosRealm.IsUnknown() && plan.KerberosRealm.ValueString() != "" {
		v := plan.KerberosRealm.ValueString()
		out.KerberosRealm = &v
	}
	if !plan.Force.IsNull() && !plan.Force.IsUnknown() {
		v := plan.Force.ValueBool()
		out.Force = &v
	}

	if !plan.CredentialJSON.IsNull() && !plan.CredentialJSON.IsUnknown() && plan.CredentialJSON.ValueString() != "" {
		var cred map[string]interface{}
		if err := json.Unmarshal([]byte(plan.CredentialJSON.ValueString()), &cred); err != nil {
			diags.AddError("Invalid credential_json", fmt.Sprintf("Could not parse credential_json as JSON: %s", err))
			return out
		}
		out.Credential = &cred
	}
	if !plan.ConfigurationJSON.IsNull() && !plan.ConfigurationJSON.IsUnknown() && plan.ConfigurationJSON.ValueString() != "" {
		var conf map[string]interface{}
		if err := json.Unmarshal([]byte(plan.ConfigurationJSON.ValueString()), &conf); err != nil {
			diags.AddError("Invalid configuration_json", fmt.Sprintf("Could not parse configuration_json as JSON: %s", err))
			return out
		}
		out.Configuration = &conf
	}

	return out
}

func (r *DirectoryServicesResource) mapResponseToModel(cfg *client.DirectoryServicesConfig, model *DirectoryServicesResourceModel) {
	model.ID = types.StringValue("1")

	if cfg.ServiceType != nil {
		model.ServiceType = types.StringValue(*cfg.ServiceType)
	} else {
		model.ServiceType = types.StringValue("")
	}
	model.Enable = types.BoolValue(cfg.Enable)
	model.EnableAccountCache = types.BoolValue(cfg.EnableAccountCache)
	model.EnableDNSUpdates = types.BoolValue(cfg.EnableDNSUpdates)
	model.Timeout = types.Int64Value(int64(cfg.Timeout))

	if cfg.KerberosRealm != nil {
		model.KerberosRealm = types.StringValue(*cfg.KerberosRealm)
	} else {
		model.KerberosRealm = types.StringValue("")
	}

	// Preserve user-supplied credential_json / configuration_json if already
	// set (we don't want to overwrite sensitive secrets with API responses
	// that may redact passwords). For unset (import) case, marshal whatever
	// the API returned.
	// json.Marshal cannot fail on map[string]interface{} values from the API.
	if model.CredentialJSON.IsNull() || model.CredentialJSON.IsUnknown() {
		if cfg.Credential != nil {
			b, _ := json.Marshal(cfg.Credential)
			model.CredentialJSON = types.StringValue(string(b))
		}
	}
	if model.ConfigurationJSON.IsNull() || model.ConfigurationJSON.IsUnknown() {
		if cfg.Configuration != nil {
			b, _ := json.Marshal(cfg.Configuration)
			model.ConfigurationJSON = types.StringValue(string(b))
		}
	}
	// force is input-only; preserve whatever plan had. If nil, keep null.
}
