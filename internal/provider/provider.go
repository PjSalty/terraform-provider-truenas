package provider

import (
	"context"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/datasources"
	"github.com/PjSalty/terraform-provider-truenas/internal/resources"
)

var _ provider.Provider = &TrueNASProvider{}

// newClientFn is the constructor used by Configure to build the TrueNAS API
// client.  It is a package-level variable so tests can substitute a fake that
// returns an error and exercise the error-handling branch.
var newClientFn = client.NewWithOptions

// TrueNASProvider implements the TrueNAS SCALE Terraform provider.
type TrueNASProvider struct {
	version string
}

// TrueNASProviderModel describes the provider configuration.
type TrueNASProviderModel struct {
	URL                types.String `tfsdk:"url"`
	APIKey             types.String `tfsdk:"api_key"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	ReadOnly           types.Bool   `tfsdk:"read_only"`
	DestroyProtection  types.Bool   `tfsdk:"destroy_protection"`
	RequestTimeout     types.String `tfsdk:"request_timeout"`
}

// New returns a new provider factory function.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &TrueNASProvider{
			version: version,
		}
	}
}

func (p *TrueNASProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "truenas"
	resp.Version = p.version
}

func (p *TrueNASProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for managing TrueNAS SCALE resources via the REST API v2.0.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Description: "The base URL of the TrueNAS SCALE instance (e.g., https://truenas.example.com). " +
					"Can also be set via the TRUENAS_URL environment variable.",
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				Description: "The API key for authenticating with TrueNAS SCALE. " +
					"Can also be set via the TRUENAS_API_KEY environment variable.",
				Optional:  true,
				Sensitive: true,
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. Only use this for self-signed " +
					"test environments. Can also be set via the TRUENAS_INSECURE_SKIP_VERIFY " +
					"environment variable.",
				Optional: true,
			},
			"read_only": schema.BoolAttribute{
				Description: "When true, the provider refuses every mutating request " +
					"(POST/PUT/DELETE) before it reaches the network. Intended for phased " +
					"production rollout — point the provider at a real TrueNAS instance, " +
					"run `terraform plan`, and be physically unable to mutate anything. " +
					"Any resource that would be changed surfaces as a normal Terraform " +
					"error rather than a partial write. Can also be set via the " +
					"TRUENAS_READONLY environment variable. Defaults to false.",
				Optional: true,
			},
			"destroy_protection": schema.BoolAttribute{
				Description: "When true, the provider refuses DELETE requests at the " +
					"client layer while still allowing GET/POST/PUT through. This is the " +
					"apply-safe rail for the first production mutations: creates and " +
					"updates work normally but no resource can be destroyed until the " +
					"flag is cleared. Equivalent to the AWS provider's per-resource " +
					"`deletion_protection`, but enforced at the wire for every resource " +
					"in the provider at once. Layer it with `read_only=true` for Phase 0 " +
					"(read-only plan against prod) and drop `read_only` + keep " +
					"`destroy_protection=true` for Phase 1 (safe-apply: create and update " +
					"only). Can also be set via the TRUENAS_DESTROY_PROTECTION " +
					"environment variable. Defaults to false for backwards compatibility; " +
					"the docs/guides/phased-rollout.md guide strongly recommends setting " +
					"this to true for any HCL pointed at production.",
				Optional: true,
			},
			"request_timeout": schema.StringAttribute{
				Description: "Per-request HTTP timeout, parsed via time.ParseDuration " +
					"(e.g. \"60s\", \"2m\", \"5m\"). Raise this for loaded production " +
					"TrueNAS instances where pool scrub status, large dataset listings, " +
					"or replication checks may take longer than the default 60s. " +
					"Lower it for local/CI testing. Can also be set via the " +
					"TRUENAS_REQUEST_TIMEOUT environment variable. Values of zero or " +
					"less are ignored so this attribute can never disable the timeout.",
				Optional: true,
			},
		},
	}
}

func (p *TrueNASProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring TrueNAS provider")

	var config TrueNASProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Resolve URL from config or environment
	url := os.Getenv("TRUENAS_URL")
	if !config.URL.IsNull() {
		url = config.URL.ValueString()
	}
	if url == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("url"),
			"Missing TrueNAS URL",
			"The provider requires a TrueNAS URL to be set either in the provider configuration "+
				"or via the TRUENAS_URL environment variable.",
		)
	}

	// Resolve API key from config or environment
	apiKey := os.Getenv("TRUENAS_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing TrueNAS API Key",
			"The provider requires an API key to be set either in the provider configuration "+
				"or via the TRUENAS_API_KEY environment variable.",
		)
	}

	// Resolve insecure_skip_verify from config or environment
	insecureSkipVerify := false
	if v := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY"); v == "true" || v == "1" {
		insecureSkipVerify = true
	}
	if !config.InsecureSkipVerify.IsNull() {
		insecureSkipVerify = config.InsecureSkipVerify.ValueBool()
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Create the API client
	c, err := newClientFn(url, apiKey, insecureSkipVerify)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create TrueNAS API Client",
			"An unexpected error occurred when creating the TrueNAS API client: "+err.Error(),
		)
		return
	}

	// Read-only safety rail: read_only=true (HCL) or TRUENAS_READONLY={1,true}
	// (env) makes the provider refuse every mutating request (POST/PUT/DELETE)
	// before it reaches the network. Intended for phased production rollout —
	// point the provider at a real TrueNAS, run `terraform plan`, and be
	// physically unable to mutate anything. A surprised or buggy plan
	// surfaces as a normal Terraform error instead of a partial write.
	// HCL takes precedence over env var when both are set. See
	// internal/client/readonly.go for the gate implementation.
	readOnly := false
	if v := os.Getenv("TRUENAS_READONLY"); v == "true" || v == "1" {
		readOnly = true
	}
	if !config.ReadOnly.IsNull() {
		readOnly = config.ReadOnly.ValueBool()
	}
	if readOnly {
		c.ReadOnly = true
		tflog.Warn(ctx, "TrueNAS provider is in read-only mode — all mutating requests will fail with ErrReadOnly", map[string]interface{}{
			"url": url,
		})
	}

	// Destroy-protection safety rail: destroy_protection=true (HCL) or
	// TRUENAS_DESTROY_PROTECTION={1,true} (env) makes the provider refuse
	// DELETE requests at the client layer while allowing GET/POST/PUT
	// through. This is the "first production apply" profile: creates and
	// updates work freely but nothing can be destroyed until the operator
	// explicitly clears the flag. HCL takes precedence over env var.
	// See internal/client/destroy_protection.go for the gate.
	destroyProtection := false
	if v := os.Getenv("TRUENAS_DESTROY_PROTECTION"); v == "true" || v == "1" {
		destroyProtection = true
	}
	if !config.DestroyProtection.IsNull() {
		destroyProtection = config.DestroyProtection.ValueBool()
	}
	if destroyProtection {
		c.DestroyProtection = true
		tflog.Warn(ctx, "TrueNAS provider is in destroy-protected mode — DELETE requests will fail with ErrDestroyProtected", map[string]interface{}{
			"url": url,
		})
	}

	// Per-request HTTP timeout override. HCL takes precedence over env var.
	// Zero or negative durations are silently ignored by SetRequestTimeout
	// so a malformed config can never accidentally disable the timeout.
	// Parse errors on the string value surface as a normal provider
	// diagnostic instead of being silently swallowed.
	var requestTimeoutRaw string
	if v := os.Getenv("TRUENAS_REQUEST_TIMEOUT"); v != "" {
		requestTimeoutRaw = v
	}
	if !config.RequestTimeout.IsNull() {
		requestTimeoutRaw = config.RequestTimeout.ValueString()
	}
	if requestTimeoutRaw != "" {
		parsed, perr := time.ParseDuration(requestTimeoutRaw)
		if perr != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("request_timeout"),
				"Invalid request_timeout duration",
				"The request_timeout value could not be parsed as a Go time.Duration "+
					"(e.g. \"60s\", \"2m\", \"5m\"): "+perr.Error(),
			)
			return
		}
		c.SetRequestTimeout(parsed)
	}

	tflog.Debug(ctx, "TrueNAS provider configured", map[string]interface{}{
		"url":                url,
		"read_only":          c.ReadOnly,
		"destroy_protection": c.DestroyProtection,
		"request_timeout":    c.RequestTimeout().String(),
	})

	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *TrueNASProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatasetResource,
		resources.NewNFSShareResource,
		resources.NewSMBShareResource,
		resources.NewSnapshotTaskResource,
		resources.NewISCSITargetResource,
		resources.NewISCSIPortalResource,
		resources.NewISCSIExtentResource,
		resources.NewISCSIInitiatorResource,
		resources.NewCronJobResource,
		resources.NewAlertServiceResource,
		resources.NewReplicationResource,
		resources.NewServiceResource,
		resources.NewNFSConfigResource,
		resources.NewSMBConfigResource,
		resources.NewSSHConfigResource,
		resources.NewStaticRouteResource,
		resources.NewZvolResource,
		resources.NewUserResource,
		resources.NewGroupResource,
		resources.NewTunableResource,
		resources.NewCloudSyncResource,
		resources.NewRsyncTaskResource,
		resources.NewDNSNameserverResource,
		resources.NewInitScriptResource,
		resources.NewISCSITargetExtentResource,
		resources.NewScrubTaskResource,
		resources.NewNetworkConfigResource,
		resources.NewFilesystemACLResource,
		resources.NewMailConfigResource,
		resources.NewSNMPConfigResource,
		resources.NewUPSConfigResource,
		resources.NewFTPConfigResource,
		resources.NewCertificateResource,
		resources.NewAPIKeyResource,
		resources.NewKeychainCredentialResource,
		resources.NewACMEDNSAuthenticatorResource,
		// v0.4.0 — resource expansion wave 1 (18 resources)
		resources.NewVMResource,
		resources.NewVMDeviceResource,
		resources.NewAppResource,
		resources.NewCatalogResource,
		resources.NewPrivilegeResource,
		resources.NewKerberosRealmResource,
		resources.NewKerberosKeytabResource,
		resources.NewDirectoryServicesResource,
		resources.NewPoolResource,
		resources.NewNetworkInterfaceResource,
		resources.NewSystemDatasetResource,
		// v1.10.0 — SCALE update control (auto_download + train pinning)
		resources.NewSystemUpdateResource,
		resources.NewNVMetGlobalResource,
		resources.NewNVMetHostResource,
		resources.NewNVMetSubsysResource,
		resources.NewNVMetPortResource,
		resources.NewNVMetNamespaceResource,
		resources.NewNVMetHostSubsysResource,
		resources.NewNVMetPortSubsysResource,
		// v0.4.0 — resource expansion wave 2 (7 resources)
		resources.NewVMwareResource,
		resources.NewCloudBackupResource,
		resources.NewReportingExporterResource,
		resources.NewISCSIAuthResource,
		resources.NewKMIPConfigResource,
		resources.NewAlertClassesResource,
		resources.NewFilesystemACLTemplateResource,
		resources.NewCloudSyncCredentialResource,
	}
}

func (p *TrueNASProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewPoolDataSource,
		datasources.NewDatasetDataSource,
		datasources.NewSystemInfoDataSource,
		datasources.NewNetworkConfigDataSource,
		datasources.NewDiskDataSource,
		datasources.NewUserDataSource,
		datasources.NewGroupDataSource,
		datasources.NewServiceDataSource,
		datasources.NewCertificateDataSource,
		// v0.4.0 data sources — expansion
		datasources.NewVMDataSource,
		datasources.NewPrivilegeDataSource,
		datasources.NewKerberosRealmDataSource,
		datasources.NewAppDataSource,
		datasources.NewCatalogDataSource,
		datasources.NewDirectoryServicesDataSource,
		datasources.NewSystemDatasetDataSource,
		datasources.NewNetworkInterfaceDataSource,
		datasources.NewShareNFSDataSource,
		datasources.NewShareSMBDataSource,
		datasources.NewCronJobDataSource,
		datasources.NewVMsDataSource,
		datasources.NewDatasetsDataSource,
		datasources.NewPoolsDataSource,
		datasources.NewAppsDataSource,
		datasources.NewCloudSyncCredentialDataSource,
		// v0.5.0 — resource/data-source parity expansion
		datasources.NewISCSITargetDataSource,
		datasources.NewISCSIPortalDataSource,
		datasources.NewISCSIExtentDataSource,
		datasources.NewISCSIInitiatorDataSource,
		datasources.NewAPIKeyDataSource,
		datasources.NewKeychainCredentialDataSource,
		datasources.NewSnapshotTaskDataSource,
		datasources.NewAlertServiceDataSource,
	}
}
