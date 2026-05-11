package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &DirectoryServicesDataSource{}

// DirectoryServicesDataSource provides the singleton directory services config.
type DirectoryServicesDataSource struct {
	client *client.Client
}

// DirectoryServicesDataSourceModel describes the data source model.
//
// The polymorphic `credential` and `configuration` objects are exposed as
// JSON strings so callers can decode them in-terraform without the provider
// having to model every variant.
type DirectoryServicesDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ServiceType        types.String `tfsdk:"service_type"`
	Enable             types.Bool   `tfsdk:"enable"`
	EnableAccountCache types.Bool   `tfsdk:"enable_account_cache"`
	EnableDNSUpdates   types.Bool   `tfsdk:"enable_dns_updates"`
	Timeout            types.Int64  `tfsdk:"timeout"`
	KerberosRealm      types.String `tfsdk:"kerberos_realm"`
	CredentialJSON     types.String `tfsdk:"credential_json"`
	ConfigurationJSON  types.String `tfsdk:"configuration_json"`
}

func NewDirectoryServicesDataSource() datasource.DataSource {
	return &DirectoryServicesDataSource{}
}

func (d *DirectoryServicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directoryservices"
}

func (d *DirectoryServicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides the singleton directory services configuration on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Singleton ID (always 'directoryservices').",
				Computed:    true,
			},
			"service_type": schema.StringAttribute{
				Description: "Directory service type (ACTIVEDIRECTORY, IPA, LDAP).",
				Computed:    true,
			},
			"enable": schema.BoolAttribute{
				Description: "Whether directory services are enabled.",
				Computed:    true,
			},
			"enable_account_cache": schema.BoolAttribute{
				Description: "Whether the account cache is enabled.",
				Computed:    true,
			},
			"enable_dns_updates": schema.BoolAttribute{
				Description: "Whether DNS updates are enabled.",
				Computed:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "Timeout in seconds.",
				Computed:    true,
			},
			"kerberos_realm": schema.StringAttribute{
				Description: "Associated Kerberos realm, if any.",
				Computed:    true,
			},
			"credential_json": schema.StringAttribute{
				Description: "Polymorphic credential object as a JSON string.",
				Computed:    true,
			},
			"configuration_json": schema.StringAttribute{
				Description: "Polymorphic configuration object as a JSON string.",
				Computed:    true,
			},
		},
	}
}

func (d *DirectoryServicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *DirectoryServicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	cfg, err := d.client.GetDirectoryServicesConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Directory Services",
			fmt.Sprintf("Could not read directory services config: %s", err),
		)
		return
	}

	model := DirectoryServicesDataSourceModel{
		ID:                 types.StringValue("directoryservices"),
		Enable:             types.BoolValue(cfg.Enable),
		EnableAccountCache: types.BoolValue(cfg.EnableAccountCache),
		EnableDNSUpdates:   types.BoolValue(cfg.EnableDNSUpdates),
		Timeout:            types.Int64Value(int64(cfg.Timeout)),
	}

	if cfg.ServiceType != nil {
		model.ServiceType = types.StringValue(*cfg.ServiceType)
	} else {
		model.ServiceType = types.StringNull()
	}
	if cfg.KerberosRealm != nil {
		model.KerberosRealm = types.StringValue(*cfg.KerberosRealm)
	} else {
		model.KerberosRealm = types.StringNull()
	}

	if cfg.Credential != nil {
		// json.Marshal of map[string]interface{} decoded from JSON cannot
		// return an error, so the error is deliberately ignored.
		b, _ := json.Marshal(cfg.Credential)
		model.CredentialJSON = types.StringValue(string(b))
	} else {
		model.CredentialJSON = types.StringNull()
	}

	if cfg.Configuration != nil {
		// json.Marshal of map[string]interface{} decoded from JSON cannot
		// return an error, so the error is deliberately ignored.
		b, _ := json.Marshal(cfg.Configuration)
		model.ConfigurationJSON = types.StringValue(string(b))
	} else {
		model.ConfigurationJSON = types.StringNull()
	}

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}
