package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &SystemDatasetDataSource{}

// SystemDatasetDataSource provides the singleton system dataset config.
type SystemDatasetDataSource struct {
	client *client.Client
}

// SystemDatasetDataSourceModel describes the data source model.
type SystemDatasetDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Pool     types.String `tfsdk:"pool"`
	PoolSet  types.Bool   `tfsdk:"pool_set"`
	UUID     types.String `tfsdk:"uuid"`
	Basename types.String `tfsdk:"basename"`
	Path     types.String `tfsdk:"path"`
}

func NewSystemDatasetDataSource() datasource.DataSource {
	return &SystemDatasetDataSource{}
}

func (d *SystemDatasetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_systemdataset"
}

func (d *SystemDatasetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides the singleton system dataset configuration on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Singleton ID (always 'systemdataset').",
				Computed:    true,
			},
			"pool": schema.StringAttribute{
				Description: "Pool hosting the system dataset.",
				Computed:    true,
			},
			"pool_set": schema.BoolAttribute{
				Description: "Whether the pool has been explicitly set.",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "System dataset UUID.",
				Computed:    true,
			},
			"basename": schema.StringAttribute{
				Description: "System dataset basename.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "On-disk path of the system dataset.",
				Computed:    true,
			},
		},
	}
}

func (d *SystemDatasetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SystemDatasetDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	cfg, err := d.client.GetSystemDataset(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading System Dataset",
			fmt.Sprintf("Could not read system dataset: %s", err),
		)
		return
	}

	model := SystemDatasetDataSourceModel{
		ID:       types.StringValue("systemdataset"),
		Pool:     types.StringValue(cfg.Pool),
		PoolSet:  types.BoolValue(cfg.PoolSet),
		UUID:     types.StringValue(cfg.UUID),
		Basename: types.StringValue(cfg.Basename),
		Path:     types.StringValue(cfg.Path),
	}

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}
