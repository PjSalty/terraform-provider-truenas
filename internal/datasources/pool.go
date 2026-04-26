package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &PoolDataSource{}

// PoolDataSource provides information about a ZFS pool.
type PoolDataSource struct {
	client *client.Client
}

// PoolDataSourceModel describes the data source model.
type PoolDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	GUID        types.String `tfsdk:"guid"`
	Path        types.String `tfsdk:"path"`
	Status      types.String `tfsdk:"status"`
	Healthy     types.Bool   `tfsdk:"healthy"`
	IsDecrypted types.Bool   `tfsdk:"is_decrypted"`
}

func NewPoolDataSource() datasource.DataSource {
	return &PoolDataSource{}
}

func (d *PoolDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pool"
}

func (d *PoolDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a ZFS pool on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The pool ID to look up.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The pool name to look up. Either id or name must be specified.",
				Optional:    true,
				Computed:    true,
			},
			"guid": schema.StringAttribute{
				Description: "The pool GUID.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The pool mount path.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The pool status (ONLINE, DEGRADED, FAULTED, etc.).",
				Computed:    true,
			},
			"healthy": schema.BoolAttribute{
				Description: "Whether the pool is healthy.",
				Computed:    true,
			},
			"is_decrypted": schema.BoolAttribute{
				Description: "Whether the pool is decrypted.",
				Computed:    true,
			},
		},
	}
}

func (d *PoolDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PoolDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Look up by ID or by name
	if !config.ID.IsNull() {
		pool, err := d.client.GetPool(ctx, int(config.ID.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Pool",
				fmt.Sprintf("Could not read pool with ID %d: %s", config.ID.ValueInt64(), err),
			)
			return
		}
		d.mapPoolToModel(pool, &config)
	} else if !config.Name.IsNull() {
		pools, err := d.client.ListPools(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Listing Pools",
				fmt.Sprintf("Could not list pools: %s", err),
			)
			return
		}
		found := false
		for _, pool := range pools {
			if pool.Name == config.Name.ValueString() {
				d.mapPoolToModel(&pool, &config)
				found = true
				break
			}
		}
		if !found {
			resp.Diagnostics.AddError(
				"Pool Not Found",
				fmt.Sprintf("No pool found with name %q", config.Name.ValueString()),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Missing Required Attribute",
			"Either 'id' or 'name' must be specified.",
		)
		return
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}

func (d *PoolDataSource) mapPoolToModel(pool *client.Pool, model *PoolDataSourceModel) {
	model.ID = types.Int64Value(int64(pool.ID))
	model.Name = types.StringValue(pool.Name)
	model.GUID = types.StringValue(pool.GUID)
	model.Path = types.StringValue(pool.Path)
	model.Status = types.StringValue(pool.Status)
	model.Healthy = types.BoolValue(pool.Healthy)
	model.IsDecrypted = types.BoolValue(pool.IsDecrypted)
}
