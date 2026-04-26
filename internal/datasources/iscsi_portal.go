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

var _ datasource.DataSource = &ISCSIPortalDataSource{}

// ISCSIPortalDataSource provides information about an iSCSI portal.
type ISCSIPortalDataSource struct {
	client *client.Client
}

// ISCSIPortalDataSourceModel describes the data source model.
type ISCSIPortalDataSourceModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Comment types.String `tfsdk:"comment"`
	Tag     types.Int64  `tfsdk:"tag"`
	Listen  types.List   `tfsdk:"listen"`
}

var iscsiPortalListenDSAttrTypes = map[string]attr.Type{
	"ip":   types.StringType,
	"port": types.Int64Type,
}

func NewISCSIPortalDataSource() datasource.DataSource {
	return &ISCSIPortalDataSource{}
}

func (d *ISCSIPortalDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_portal"
}

func (d *ISCSIPortalDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an iSCSI portal on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric iSCSI portal ID to look up.",
				Required:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Portal comment.",
				Computed:    true,
			},
			"tag": schema.Int64Attribute{
				Description: "Portal group tag.",
				Computed:    true,
			},
			"listen": schema.ListNestedAttribute{
				Description: "Listen addresses for the portal.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"ip":   schema.StringAttribute{Description: "Listen IP.", Computed: true},
						"port": schema.Int64Attribute{Description: "Listen port.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *ISCSIPortalDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ISCSIPortalDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ISCSIPortalDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	portal, err := d.client.GetISCSIPortal(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Portal",
			fmt.Sprintf("Could not read iSCSI portal with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(portal.ID))
	config.Comment = types.StringValue(portal.Comment)
	config.Tag = types.Int64Value(int64(portal.Tag))

	listenObjects := make([]attr.Value, 0, len(portal.Listen))
	for _, l := range portal.Listen {
		obj, _ := types.ObjectValue(iscsiPortalListenDSAttrTypes, map[string]attr.Value{
			"ip":   types.StringValue(l.IP),
			"port": types.Int64Value(int64(l.Port)),
		})
		listenObjects = append(listenObjects, obj)
	}
	listenList, _ := types.ListValue(types.ObjectType{AttrTypes: iscsiPortalListenDSAttrTypes}, listenObjects)
	config.Listen = listenList

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
