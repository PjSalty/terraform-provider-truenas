package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &AlertServiceDataSource{}

// AlertServiceDataSource provides information about an alert service.
type AlertServiceDataSource struct {
	client *client.Client
}

// AlertServiceDataSourceModel describes the data source model.
type AlertServiceDataSourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Type         types.String `tfsdk:"type"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Level        types.String `tfsdk:"level"`
	SettingsJSON types.String `tfsdk:"settings_json"`
}

func NewAlertServiceDataSource() datasource.DataSource {
	return &AlertServiceDataSource{}
}

func (d *AlertServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_service"
}

func (d *AlertServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an alert service on TrueNAS SCALE. " +
			"Service-specific settings are exposed as a JSON string because the " +
			"shape varies per service type (Slack, PagerDuty, Mail, etc.).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric alert service ID to look up.",
				Required:    true,
			},
			"name":    schema.StringAttribute{Description: "Alert service display name.", Computed: true},
			"type":    schema.StringAttribute{Description: "Alert service type (Slack, PagerDuty, Mail, etc.).", Computed: true},
			"enabled": schema.BoolAttribute{Description: "Whether the service is enabled.", Computed: true},
			"level":   schema.StringAttribute{Description: "Minimum severity level (INFO, NOTICE, WARNING, etc.).", Computed: true},
			"settings_json": schema.StringAttribute{
				Description: "Service-specific settings as an opaque JSON string. " +
					"May contain sensitive tokens or webhooks.",
				Computed:  true,
				Sensitive: true,
			},
		},
	}
}

func (d *AlertServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AlertServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AlertServiceDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	svc, err := d.client.GetAlertService(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Alert Service",
			fmt.Sprintf("Could not read alert service with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(svc.ID))
	config.Name = types.StringValue(svc.Name)
	config.Type = types.StringValue(svc.GetType())
	config.Enabled = types.BoolValue(svc.Enabled)
	config.Level = types.StringValue(svc.Level)

	if svc.Settings == nil {
		config.SettingsJSON = types.StringValue("{}")
	} else {
		b, _ := json.Marshal(svc.Settings)
		config.SettingsJSON = types.StringValue(string(b))
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
