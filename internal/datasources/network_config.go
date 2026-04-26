package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &NetworkConfigDataSource{}

// NetworkConfigDataSource provides network configuration from TrueNAS.
type NetworkConfigDataSource struct {
	client *client.Client
}

// NetworkConfigDataSourceModel describes the data source model.
type NetworkConfigDataSourceModel struct {
	Hostname    types.String `tfsdk:"hostname"`
	Domain      types.String `tfsdk:"domain"`
	Nameserver1 types.String `tfsdk:"nameserver1"`
	Nameserver2 types.String `tfsdk:"nameserver2"`
	Nameserver3 types.String `tfsdk:"nameserver3"`
	IPv4Gateway types.String `tfsdk:"ipv4gateway"`
	HTTPProxy   types.String `tfsdk:"httpproxy"`
}

func NewNetworkConfigDataSource() datasource.DataSource {
	return &NetworkConfigDataSource{}
}

func (d *NetworkConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_network_config"
}

func (d *NetworkConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides network configuration from TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"hostname": schema.StringAttribute{
				Description: "The system hostname.",
				Computed:    true,
			},
			"domain": schema.StringAttribute{
				Description: "The system domain.",
				Computed:    true,
			},
			"nameserver1": schema.StringAttribute{
				Description: "Primary DNS nameserver.",
				Computed:    true,
			},
			"nameserver2": schema.StringAttribute{
				Description: "Secondary DNS nameserver.",
				Computed:    true,
			},
			"nameserver3": schema.StringAttribute{
				Description: "Tertiary DNS nameserver.",
				Computed:    true,
			},
			"ipv4gateway": schema.StringAttribute{
				Description: "IPv4 default gateway.",
				Computed:    true,
			},
			"httpproxy": schema.StringAttribute{
				Description: "HTTP proxy URL.",
				Computed:    true,
			},
		},
	}
}

func (d *NetworkConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NetworkConfigDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	config, err := d.client.GetNetworkConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Network Config",
			fmt.Sprintf("Could not read network configuration: %s", err),
		)
		return
	}

	state := NetworkConfigDataSourceModel{
		Hostname:    types.StringValue(config.Hostname),
		Domain:      types.StringValue(config.Domain),
		Nameserver1: types.StringValue(config.Nameserver1),
		Nameserver2: types.StringValue(config.Nameserver2),
		Nameserver3: types.StringValue(config.Nameserver3),
		IPv4Gateway: types.StringValue(config.IPv4Gateway),
		HTTPProxy:   types.StringValue(config.HTTPProxy),
	}

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
