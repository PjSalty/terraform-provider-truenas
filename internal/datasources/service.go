package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &ServiceDataSource{}

// ServiceDataSource provides information about a TrueNAS service.
type ServiceDataSource struct {
	client *client.Client
}

// ServiceDataSourceModel describes the data source model.
type ServiceDataSourceModel struct {
	ID      types.Int64  `tfsdk:"id"`
	Service types.String `tfsdk:"service"`
	Enable  types.Bool   `tfsdk:"enable"`
	State   types.String `tfsdk:"state"`
}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

func (d *ServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a service on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The service ID.",
				Computed:    true,
			},
			"service": schema.StringAttribute{
				Description: "The service name to look up (e.g., ssh, nfs, cifs, ftp, snmp, ups).",
				Required:    true,
			},
			"enable": schema.BoolAttribute{
				Description: "Whether the service is enabled to start at boot.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The current state of the service (RUNNING, STOPPED).",
				Computed:    true,
			},
		},
	}
}

func (d *ServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ServiceDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceName := config.Service.ValueString()

	svc, err := d.client.GetServiceByName(ctx, serviceName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Service",
			fmt.Sprintf("Could not find service %q: %s", serviceName, err),
		)
		return
	}

	config.ID = types.Int64Value(int64(svc.ID))
	config.Service = types.StringValue(svc.Service)
	config.Enable = types.BoolValue(svc.Enable)
	config.State = types.StringValue(svc.State)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
