package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &PoolsDataSource{}

// PoolsDataSource provides a listing of all ZFS pools.
type PoolsDataSource struct {
	client *client.Client
}

// PoolsDataSourceModel is the top-level data source model.
type PoolsDataSourceModel struct {
	Pools types.List `tfsdk:"pools"`
}

func NewPoolsDataSource() datasource.DataSource {
	return &PoolsDataSource{}
}

func (d *PoolsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pools"
}

// poolsListAttrTypes returns the per-element object types for the list.
func poolsListAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":           types.Int64Type,
		"name":         types.StringType,
		"guid":         types.StringType,
		"path":         types.StringType,
		"status":       types.StringType,
		"healthy":      types.BoolType,
		"is_decrypted": types.BoolType,
	}
}

func (d *PoolsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ZFS pools on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"pools": schema.ListNestedAttribute{
				Description: "List of ZFS pools.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "Pool ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Pool name.",
							Computed:    true,
						},
						"guid": schema.StringAttribute{
							Description: "Pool GUID.",
							Computed:    true,
						},
						"path": schema.StringAttribute{
							Description: "Pool mount path.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "Pool status.",
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
				},
			},
		},
	}
}

func (d *PoolsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PoolsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	pools, err := d.client.ListPools(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Pools",
			fmt.Sprintf("Could not list pools: %s", err),
		)
		return
	}

	objType := types.ObjectType{AttrTypes: poolsListAttrTypes()}
	elems := make([]attr.Value, 0, len(pools))
	for _, p := range pools {
		obj, objDiags := types.ObjectValue(poolsListAttrTypes(), map[string]attr.Value{
			"id":           types.Int64Value(int64(p.ID)),
			"name":         types.StringValue(p.Name),
			"guid":         types.StringValue(p.GUID),
			"path":         types.StringValue(p.Path),
			"status":       types.StringValue(p.Status),
			"healthy":      types.BoolValue(p.Healthy),
			"is_decrypted": types.BoolValue(p.IsDecrypted),
		})
		resp.Diagnostics.Append(objDiags...)
		elems = append(elems, obj)
	}

	list, lDiags := types.ListValue(objType, elems)
	resp.Diagnostics.Append(lDiags...)

	model := PoolsDataSourceModel{Pools: list}
	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}
