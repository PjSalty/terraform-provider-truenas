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

var _ datasource.DataSource = &ISCSITargetDataSource{}

// ISCSITargetDataSource provides information about an iSCSI target.
type ISCSITargetDataSource struct {
	client *client.Client
}

// ISCSITargetDataSourceModel describes the data source model.
type ISCSITargetDataSourceModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Alias  types.String `tfsdk:"alias"`
	Mode   types.String `tfsdk:"mode"`
	Groups types.List   `tfsdk:"groups"`
}

// iscsiTargetGroupDSAttrTypes is the object schema for a group entry.
var iscsiTargetGroupDSAttrTypes = map[string]attr.Type{
	"portal":      types.Int64Type,
	"initiator":   types.Int64Type,
	"auth_method": types.StringType,
	"auth":        types.Int64Type,
}

func NewISCSITargetDataSource() datasource.DataSource {
	return &ISCSITargetDataSource{}
}

func (d *ISCSITargetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_target"
}

func (d *ISCSITargetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an iSCSI target on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric iSCSI target ID to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The iSCSI target name (IQN suffix).",
				Computed:    true,
			},
			"alias": schema.StringAttribute{
				Description: "Optional alias for the target.",
				Computed:    true,
			},
			"mode": schema.StringAttribute{
				Description: "The target mode (ISCSI, FC, BOTH).",
				Computed:    true,
			},
			"groups": schema.ListNestedAttribute{
				Description: "Target groups linking portals and initiators.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"portal":      schema.Int64Attribute{Description: "Portal ID.", Computed: true},
						"initiator":   schema.Int64Attribute{Description: "Initiator group ID.", Computed: true},
						"auth_method": schema.StringAttribute{Description: "Authentication method.", Computed: true},
						"auth":        schema.Int64Attribute{Description: "Auth group ID.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *ISCSITargetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ISCSITargetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ISCSITargetDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	target, err := d.client.GetISCSITarget(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Target",
			fmt.Sprintf("Could not read iSCSI target with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(target.ID))
	config.Name = types.StringValue(target.Name)
	config.Alias = types.StringValue(target.Alias)
	config.Mode = types.StringValue(target.Mode)

	groupObjects := make([]attr.Value, 0, len(target.Groups))
	for _, g := range target.Groups {
		obj, _ := types.ObjectValue(iscsiTargetGroupDSAttrTypes, map[string]attr.Value{
			"portal":      types.Int64Value(int64(g.Portal)),
			"initiator":   types.Int64Value(int64(g.Initiator)),
			"auth_method": types.StringValue(g.AuthMethod),
			"auth":        types.Int64Value(int64(g.Auth)),
		})
		groupObjects = append(groupObjects, obj)
	}
	groupList, _ := types.ListValue(types.ObjectType{AttrTypes: iscsiTargetGroupDSAttrTypes}, groupObjects)
	config.Groups = groupList

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
