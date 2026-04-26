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

var _ datasource.DataSource = &KeychainCredentialDataSource{}

// KeychainCredentialDataSource provides information about a keychain credential.
//
// The `attributes` map may contain sensitive fields (private keys, passphrases).
// To avoid leaking secrets via state, it is surfaced as an opaque JSON string
// that callers can inspect deliberately rather than as typed attributes.
type KeychainCredentialDataSource struct {
	client *client.Client
}

// KeychainCredentialDataSourceModel describes the data source model.
type KeychainCredentialDataSourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	AttributesJSON types.String `tfsdk:"attributes_json"`
}

func NewKeychainCredentialDataSource() datasource.DataSource {
	return &KeychainCredentialDataSource{}
}

func (d *KeychainCredentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_keychain_credential"
}

func (d *KeychainCredentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a TrueNAS keychain credential (SSH keypair " +
			"or connection). Credential attributes are surfaced as a JSON string because " +
			"they may contain private-key material.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric keychain credential ID to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Credential name.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Credential type (SSH_KEY_PAIR, SSH_CREDENTIALS).",
				Computed:    true,
			},
			"attributes_json": schema.StringAttribute{
				Description: "Credential attributes as an opaque JSON string. " +
					"May contain sensitive private-key material.",
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func (d *KeychainCredentialDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *KeychainCredentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config KeychainCredentialDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	cred, err := d.client.GetKeychainCredential(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Keychain Credential",
			fmt.Sprintf("Could not read keychain credential with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(cred.ID))
	config.Name = types.StringValue(cred.Name)
	config.Type = types.StringValue(cred.Type)

	if cred.Attributes == nil {
		config.AttributesJSON = types.StringValue("{}")
	} else {
		// map[string]any unmarshalled from JSON always round-trips cleanly,
		// so Marshal cannot realistically fail here.
		b, _ := json.Marshal(cred.Attributes)
		config.AttributesJSON = types.StringValue(string(b))
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
