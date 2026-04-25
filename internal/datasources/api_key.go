package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &APIKeyDataSource{}

// APIKeyDataSource provides information about an API key.
//
// The secret `key` value is never returned by the GET endpoint (only on
// creation), so this data source exposes only identifying metadata.
type APIKeyDataSource struct {
	client *client.Client
}

// APIKeyDataSourceModel describes the data source model.
type APIKeyDataSourceModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Username types.String `tfsdk:"username"`
	Local    types.Bool   `tfsdk:"local"`
	Revoked  types.Bool   `tfsdk:"revoked"`
}

func NewAPIKeyDataSource() datasource.DataSource {
	return &APIKeyDataSource{}
}

func (d *APIKeyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (d *APIKeyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a TrueNAS API key. The secret " +
			"`key` value is only returned at creation time and cannot be " +
			"retrieved via this data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric API key ID to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "API key display name.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The owning user.",
				Computed:    true,
			},
			"local": schema.BoolAttribute{
				Description: "Whether the key belongs to a local user.",
				Computed:    true,
			},
			"revoked": schema.BoolAttribute{
				Description: "Whether the key has been revoked.",
				Computed:    true,
			},
		},
	}
}

func (d *APIKeyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *APIKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config APIKeyDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	key, err := d.client.GetAPIKey(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading API Key",
			fmt.Sprintf("Could not read API key with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(key.ID))
	config.Name = types.StringValue(key.Name)
	config.Username = types.StringValue(key.Username)
	config.Local = types.BoolValue(key.Local)
	config.Revoked = types.BoolValue(key.Revoked)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
