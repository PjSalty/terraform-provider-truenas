package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &CloudSyncCredentialDataSource{}

// CloudSyncCredentialDataSource provides information about a cloud sync credential.
// The credential's provider attributes (access keys, secrets) are intentionally
// NOT exposed to avoid leaking secrets into plan output and state.
type CloudSyncCredentialDataSource struct {
	client *client.Client
}

// CloudSyncCredentialDataSourceModel describes the data source model.
type CloudSyncCredentialDataSourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderType types.String `tfsdk:"provider_type"`
}

func NewCloudSyncCredentialDataSource() datasource.DataSource {
	return &CloudSyncCredentialDataSource{}
}

func (d *CloudSyncCredentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloudsync_credential"
}

func (d *CloudSyncCredentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a cloud sync credential on TrueNAS SCALE by numeric ID or name. " +
			"Secret-bearing provider attributes (access keys, etc.) are not exposed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric cloud sync credential ID. Either `id` or `name` must be set.",
				Optional:    true,
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The cloud sync credential name. Either `id` or `name` must be set.",
				Optional:    true,
				Computed:    true,
			},
			"provider_type": schema.StringAttribute{
				Description: "The cloud provider type (e.g. S3, B2, AZUREBLOB).",
				Computed:    true,
			},
		},
	}
}

func (d *CloudSyncCredentialDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CloudSyncCredentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CloudSyncCredentialDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasID := !config.ID.IsNull() && !config.ID.IsUnknown()
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown() && config.Name.ValueString() != ""

	if !hasID && !hasName {
		resp.Diagnostics.AddError(
			"Missing Lookup Key",
			"Either `id` or `name` must be set to look up a cloud sync credential.",
		)
		return
	}

	var cred *client.CloudSyncCredential
	var err error

	switch {
	case hasID:
		cred, err = d.client.GetCloudSyncCredential(ctx, int(config.ID.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Cloud Sync Credential",
				fmt.Sprintf("Could not find cloud sync credential with ID %d: %s", config.ID.ValueInt64(), err),
			)
			return
		}
	default:
		cred, err = d.client.GetCloudSyncCredentialByName(ctx, config.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Cloud Sync Credential",
				fmt.Sprintf("Could not find cloud sync credential %q: %s", config.Name.ValueString(), err),
			)
			return
		}
	}

	config.ID = types.Int64Value(int64(cred.ID))
	config.Name = types.StringValue(cred.Name)

	providerType := ""
	if v, ok := cred.Provider["type"]; ok {
		if s, ok := v.(string); ok {
			providerType = s
		}
	}
	config.ProviderType = types.StringValue(providerType)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
