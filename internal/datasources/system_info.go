package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &SystemInfoDataSource{}

// SystemInfoDataSource provides system information from TrueNAS.
type SystemInfoDataSource struct {
	client *client.Client
}

// SystemInfoDataSourceModel describes the data source model.
type SystemInfoDataSourceModel struct {
	Version       types.String  `tfsdk:"version"`
	Hostname      types.String  `tfsdk:"hostname"`
	PhysicalMem   types.Int64   `tfsdk:"physical_memory"`
	Model         types.String  `tfsdk:"model"`
	Cores         types.Int64   `tfsdk:"cores"`
	Uptime        types.String  `tfsdk:"uptime"`
	UptimeSeconds types.Int64   `tfsdk:"uptime_seconds"`
	SystemSerial  types.String  `tfsdk:"system_serial"`
	SystemProduct types.String  `tfsdk:"system_product"`
	Timezone      types.String  `tfsdk:"timezone"`
	LoadAvg1      types.Float64 `tfsdk:"loadavg_1"`
	LoadAvg5      types.Float64 `tfsdk:"loadavg_5"`
	LoadAvg15     types.Float64 `tfsdk:"loadavg_15"`
}

func NewSystemInfoDataSource() datasource.DataSource {
	return &SystemInfoDataSource{}
}

func (d *SystemInfoDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_info"
}

func (d *SystemInfoDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides system information from TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"version": schema.StringAttribute{
				Description: "The TrueNAS version string.",
				Computed:    true,
			},
			"hostname": schema.StringAttribute{
				Description: "The system hostname.",
				Computed:    true,
			},
			"physical_memory": schema.Int64Attribute{
				Description: "Physical memory in bytes.",
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "CPU model.",
				Computed:    true,
			},
			"cores": schema.Int64Attribute{
				Description: "Number of CPU cores.",
				Computed:    true,
			},
			"uptime": schema.StringAttribute{
				Description: "System uptime as a human-readable string.",
				Computed:    true,
			},
			"uptime_seconds": schema.Int64Attribute{
				Description: "System uptime in seconds.",
				Computed:    true,
			},
			"system_serial": schema.StringAttribute{
				Description: "System serial number.",
				Computed:    true,
			},
			"system_product": schema.StringAttribute{
				Description: "System product name.",
				Computed:    true,
			},
			"timezone": schema.StringAttribute{
				Description: "System timezone.",
				Computed:    true,
			},
			"loadavg_1": schema.Float64Attribute{
				Description: "1-minute load average.",
				Computed:    true,
			},
			"loadavg_5": schema.Float64Attribute{
				Description: "5-minute load average.",
				Computed:    true,
			},
			"loadavg_15": schema.Float64Attribute{
				Description: "15-minute load average.",
				Computed:    true,
			},
		},
	}
}

func (d *SystemInfoDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SystemInfoDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	info, err := d.client.GetSystemInfo(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading System Info",
			fmt.Sprintf("Could not read system info: %s", err),
		)
		return
	}

	state := SystemInfoDataSourceModel{
		Version:       types.StringValue(info.Version),
		Hostname:      types.StringValue(info.Hostname),
		PhysicalMem:   types.Int64Value(info.PhysicalMem),
		Model:         types.StringValue(info.Model),
		Cores:         types.Int64Value(int64(info.Cores)),
		Uptime:        types.StringValue(info.Uptime),
		UptimeSeconds: types.Int64Value(int64(info.UptimeSeconds)),
		SystemSerial:  types.StringValue(info.SystemSerial),
		SystemProduct: types.StringValue(info.SystemProduct),
		Timezone:      types.StringValue(info.Timezone),
	}

	if len(info.Loadavg) >= 3 {
		state.LoadAvg1 = types.Float64Value(info.Loadavg[0])
		state.LoadAvg5 = types.Float64Value(info.Loadavg[1])
		state.LoadAvg15 = types.Float64Value(info.Loadavg[2])
	}

	diags := resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
