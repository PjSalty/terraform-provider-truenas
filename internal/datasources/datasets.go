package datasources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &DatasetsDataSource{}

// DatasetsDataSource provides a listing of ZFS datasets, optionally filtered
// by pool or parent dataset ID.
type DatasetsDataSource struct {
	client *client.Client
}

// DatasetsDataSourceModel is the top-level data source model.
type DatasetsDataSourceModel struct {
	Pool          types.String `tfsdk:"pool"`
	ParentDataset types.String `tfsdk:"parent_dataset"`
	Datasets      types.List   `tfsdk:"datasets"`
}

func NewDatasetsDataSource() datasource.DataSource {
	return &DatasetsDataSource{}
}

func (d *DatasetsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_datasets"
}

// datasetsListAttrTypes returns the per-element object types for the list.
func datasetsListAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":          types.StringType,
		"name":        types.StringType,
		"pool":        types.StringType,
		"type":        types.StringType,
		"mount_point": types.StringType,
	}
}

func (d *DatasetsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists ZFS datasets on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"pool": schema.StringAttribute{
				Description: "Optional pool filter. Only datasets in this pool are returned.",
				Optional:    true,
			},
			"parent_dataset": schema.StringAttribute{
				Description: "Optional parent dataset filter. Only datasets whose ID starts with this prefix are returned.",
				Optional:    true,
			},
			"datasets": schema.ListNestedAttribute{
				Description: "List of datasets matching the filters.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Full dataset ID (e.g., tank/data).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Dataset name.",
							Computed:    true,
						},
						"pool": schema.StringAttribute{
							Description: "Pool the dataset belongs to.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "Dataset type (FILESYSTEM or VOLUME).",
							Computed:    true,
						},
						"mount_point": schema.StringAttribute{
							Description: "Mount point.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *DatasetsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatasetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DatasetsDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	datasets, err := d.client.ListDatasets(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Datasets",
			fmt.Sprintf("Could not list datasets: %s", err),
		)
		return
	}

	var poolFilter, parentFilter string
	if !config.Pool.IsNull() && !config.Pool.IsUnknown() {
		poolFilter = config.Pool.ValueString()
	}
	if !config.ParentDataset.IsNull() && !config.ParentDataset.IsUnknown() {
		parentFilter = config.ParentDataset.ValueString()
	}

	objType := types.ObjectType{AttrTypes: datasetsListAttrTypes()}
	elems := make([]attr.Value, 0, len(datasets))
	for _, ds := range datasets {
		if poolFilter != "" && ds.Pool != poolFilter {
			continue
		}
		if parentFilter != "" {
			prefix := parentFilter + "/"
			if ds.ID != parentFilter && !strings.HasPrefix(ds.ID, prefix) {
				continue
			}
		}
		obj, objDiags := types.ObjectValue(datasetsListAttrTypes(), map[string]attr.Value{
			"id":          types.StringValue(ds.ID),
			"name":        types.StringValue(ds.Name),
			"pool":        types.StringValue(ds.Pool),
			"type":        types.StringValue(ds.Type),
			"mount_point": types.StringValue(ds.MountPoint),
		})
		resp.Diagnostics.Append(objDiags...)
		elems = append(elems, obj)
	}

	list, lDiags := types.ListValue(objType, elems)
	resp.Diagnostics.Append(lDiags...)

	config.Datasets = list
	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
