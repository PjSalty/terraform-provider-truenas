package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

var _ datasource.DataSource = &LXCConfigDataSource{}

// LXCConfigDataSource reads the singleton LXC container configuration.
//
// The lxc namespace is new in TrueNAS 26.0. On 25.10 and older it does not
// exist, and the client turns the resulting method-not-found into a
// diagnostic naming the required version.
type LXCConfigDataSource struct {
	client *wsclient.Client
}

// LXCConfigDataSourceModel describes the data source model.
type LXCConfigDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	PreferredPool types.String `tfsdk:"preferred_pool"`
	Bridge        types.String `tfsdk:"bridge"`
	V4Network     types.String `tfsdk:"v4_network"`
	V6Network     types.String `tfsdk:"v6_network"`
}

// NewLXCConfigDataSource returns a new LXCConfigDataSource factory.
func NewLXCConfigDataSource() datasource.DataSource {
	return &LXCConfigDataSource{}
}

func (d *LXCConfigDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lxc_config"
}

func (d *LXCConfigDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the singleton LXC container configuration on TrueNAS SCALE. " +
			"LXC containers are introduced in TrueNAS 26.0; this data source cannot be used " +
			"against 25.10 or older, which have no lxc namespace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Always `lxc_config`: this is a singleton.",
				Computed:    true,
			},
			"preferred_pool": schema.StringAttribute{
				Description: "Pool where container storage is created. Empty when unset.",
				Computed:    true,
			},
			"bridge": schema.StringAttribute{
				Description: "Bridge interface containers attach to. Empty when TrueNAS manages one automatically.",
				Computed:    true,
			},
			"v4_network": schema.StringAttribute{
				Description: "IPv4 network in CIDR notation for the managed bridge. Empty when unset.",
				Computed:    true,
			},
			"v6_network": schema.StringAttribute{
				Description: "IPv6 network in CIDR notation for the managed bridge. Empty when unset.",
				Computed:    true,
			},
		},
	}
}

func (d *LXCConfigDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*wsclient.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *wsclient.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *LXCConfigDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	cfg, err := d.client.GetLXCConfig(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading LXC Configuration",
			fmt.Sprintf("Could not read LXC configuration: %s", err),
		)
		return
	}

	// preferred_pool and bridge are nullable upstream; they are written as
	// known empty strings
	// rather than null, so a consumer interpolating one never has to guard
	// against an unknown value. v4_network and v6_network are not nullable.
	str := func(p *string) types.String {
		if p == nil {
			return types.StringValue("")
		}
		return types.StringValue(*p)
	}

	model := LXCConfigDataSourceModel{
		ID:            types.StringValue("lxc_config"),
		PreferredPool: str(cfg.PreferredPool),
		Bridge:        str(cfg.Bridge),
		V4Network:     types.StringValue(cfg.V4Network),
		V6Network:     types.StringValue(cfg.V6Network),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
