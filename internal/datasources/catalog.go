package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &CatalogDataSource{}

// CatalogDataSource provides information about the singleton TrueNAS catalog.
type CatalogDataSource struct {
	client *client.Client
}

// CatalogDataSourceModel describes the data source model.
type CatalogDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Label           types.String `tfsdk:"label"`
	PreferredTrains types.List   `tfsdk:"preferred_trains"`
	Location        types.String `tfsdk:"location"`
}

func NewCatalogDataSource() datasource.DataSource {
	return &CatalogDataSource{}
}

func (d *CatalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_catalog"
}

func (d *CatalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about the singleton TrueNAS SCALE application catalog.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Catalog ID (always 'catalog' for the singleton).",
				Computed:    true,
			},
			"label": schema.StringAttribute{
				Description: "Catalog label (typically TRUENAS).",
				Computed:    true,
			},
			"preferred_trains": schema.ListAttribute{
				Description: "List of preferred catalog trains.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"location": schema.StringAttribute{
				Description: "On-disk location of the catalog data.",
				Computed:    true,
			},
		},
	}
}

func (d *CatalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CatalogDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	cat, err := d.client.GetCatalog(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Catalog",
			fmt.Sprintf("Could not read catalog: %s", err),
		)
		return
	}

	model := CatalogDataSourceModel{
		ID:       types.StringValue("catalog"),
		Label:    types.StringValue(cat.Label),
		Location: types.StringValue(cat.Location),
	}
	if cat.ID != "" {
		model.ID = types.StringValue(cat.ID)
	}

	trains, tDiags := types.ListValueFrom(ctx, types.StringType, cat.PreferredTrains)
	resp.Diagnostics.Append(tDiags...)
	model.PreferredTrains = trains

	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}
