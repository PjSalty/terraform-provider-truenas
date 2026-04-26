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

var _ datasource.DataSource = &NetworkInterfaceDataSource{}

// NetworkInterfaceDataSource provides information about a network interface.
type NetworkInterfaceDataSource struct {
	client *client.Client
}

// networkInterfaceAliasAttrTypes returns the tfsdk attribute types for the
// nested alias object. It must match the `aliases` nested schema below.
func networkInterfaceAliasAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"type":    types.StringType,
		"address": types.StringType,
		"netmask": types.Int64Type,
	}
}

// NetworkInterfaceDataSourceModel describes the data source model.
type NetworkInterfaceDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Type                types.String `tfsdk:"type"`
	Description         types.String `tfsdk:"description"`
	IPv4DHCP            types.Bool   `tfsdk:"ipv4_dhcp"`
	IPv6Auto            types.Bool   `tfsdk:"ipv6_auto"`
	MTU                 types.Int64  `tfsdk:"mtu"`
	BridgeMembers       types.List   `tfsdk:"bridge_members"`
	LagProtocol         types.String `tfsdk:"lag_protocol"`
	LagPorts            types.List   `tfsdk:"lag_ports"`
	VlanParentInterface types.String `tfsdk:"vlan_parent_interface"`
	VlanTag             types.Int64  `tfsdk:"vlan_tag"`
	VlanPCP             types.Int64  `tfsdk:"vlan_pcp"`
	Aliases             types.List   `tfsdk:"aliases"`
}

func NewNetworkInterfaceDataSource() datasource.DataSource {
	return &NetworkInterfaceDataSource{}
}

func (d *NetworkInterfaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_interface"
}

func (d *NetworkInterfaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a network interface on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The interface ID (name, e.g. 'br0', 'vlan10').",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The interface name.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "Interface type (PHYSICAL, BRIDGE, LINK_AGGREGATION, VLAN).",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Interface description.",
				Computed:    true,
			},
			"ipv4_dhcp": schema.BoolAttribute{
				Description: "Whether IPv4 DHCP is enabled.",
				Computed:    true,
			},
			"ipv6_auto": schema.BoolAttribute{
				Description: "Whether IPv6 autoconfiguration is enabled.",
				Computed:    true,
			},
			"mtu": schema.Int64Attribute{
				Description: "Interface MTU.",
				Computed:    true,
			},
			"bridge_members": schema.ListAttribute{
				Description: "Bridge member interfaces (for BRIDGE type).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"lag_protocol": schema.StringAttribute{
				Description: "LAG protocol (for LINK_AGGREGATION type).",
				Computed:    true,
			},
			"lag_ports": schema.ListAttribute{
				Description: "LAG member ports (for LINK_AGGREGATION type).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"vlan_parent_interface": schema.StringAttribute{
				Description: "Parent interface (for VLAN type).",
				Computed:    true,
			},
			"vlan_tag": schema.Int64Attribute{
				Description: "VLAN tag (for VLAN type).",
				Computed:    true,
			},
			"vlan_pcp": schema.Int64Attribute{
				Description: "VLAN PCP (for VLAN type).",
				Computed:    true,
			},
			"aliases": schema.ListNestedAttribute{
				Description: "Static IP aliases.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Description: "Alias address family (INET, INET6).",
							Computed:    true,
						},
						"address": schema.StringAttribute{
							Description: "Alias address.",
							Computed:    true,
						},
						"netmask": schema.Int64Attribute{
							Description: "Alias netmask (prefix length).",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *NetworkInterfaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkInterfaceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config NetworkInterfaceDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, err := d.client.GetInterface(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Interface",
			fmt.Sprintf("Could not read interface %q: %s", config.ID.ValueString(), err),
		)
		return
	}

	config.ID = types.StringValue(iface.ID)
	config.Name = types.StringValue(iface.Name)
	config.Type = types.StringValue(iface.Type)
	config.Description = types.StringValue(iface.Description)
	config.IPv4DHCP = types.BoolValue(iface.IPv4DHCP)
	config.IPv6Auto = types.BoolValue(iface.IPv6Auto)
	if iface.MTU != nil {
		config.MTU = types.Int64Value(int64(*iface.MTU))
	} else {
		config.MTU = types.Int64Null()
	}
	config.LagProtocol = types.StringValue(iface.LagProtocol)
	config.VlanParentInterface = types.StringValue(iface.VlanParentInterface)
	if iface.VlanTag != nil {
		config.VlanTag = types.Int64Value(int64(*iface.VlanTag))
	} else {
		config.VlanTag = types.Int64Null()
	}
	if iface.VlanPCP != nil {
		config.VlanPCP = types.Int64Value(int64(*iface.VlanPCP))
	} else {
		config.VlanPCP = types.Int64Null()
	}

	bm, bmDiags := types.ListValueFrom(ctx, types.StringType, iface.BridgeMembers)
	resp.Diagnostics.Append(bmDiags...)
	config.BridgeMembers = bm

	lp, lpDiags := types.ListValueFrom(ctx, types.StringType, iface.LagPorts)
	resp.Diagnostics.Append(lpDiags...)
	config.LagPorts = lp

	aliasObjType := types.ObjectType{AttrTypes: networkInterfaceAliasAttrTypes()}
	aliasValues := make([]attr.Value, 0, len(iface.Aliases))
	for _, a := range iface.Aliases {
		obj, objDiags := types.ObjectValue(networkInterfaceAliasAttrTypes(), map[string]attr.Value{
			"type":    types.StringValue(a.Type),
			"address": types.StringValue(a.Address),
			"netmask": types.Int64Value(int64(a.Netmask)),
		})
		resp.Diagnostics.Append(objDiags...)
		aliasValues = append(aliasValues, obj)
	}
	aliasesOut, alDiags := types.ListValue(aliasObjType, aliasValues)
	resp.Diagnostics.Append(alDiags...)
	config.Aliases = aliasesOut

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
