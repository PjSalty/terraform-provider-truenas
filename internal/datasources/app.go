package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &AppDataSource{}

// AppDataSource provides information about a deployed TrueNAS SCALE app.
type AppDataSource struct {
	client *client.Client
}

// AppDataSourceModel describes the data source model.
type AppDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	State            types.String `tfsdk:"state"`
	UpgradeAvailable types.Bool   `tfsdk:"upgrade_available"`
	LatestVersion    types.String `tfsdk:"latest_version"`
	HumanVersion     types.String `tfsdk:"human_version"`
	Version          types.String `tfsdk:"version"`
	CustomApp        types.Bool   `tfsdk:"custom_app"`
	Migrated         types.Bool   `tfsdk:"migrated"`
}

func NewAppDataSource() datasource.DataSource {
	return &AppDataSource{}
}

func (d *AppDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *AppDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a deployed app on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The app ID (name) to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The app name.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Current state (RUNNING, STOPPED, DEPLOYING, etc.).",
				Computed:    true,
			},
			"upgrade_available": schema.BoolAttribute{
				Description: "Whether an upgrade is available.",
				Computed:    true,
			},
			"latest_version": schema.StringAttribute{
				Description: "Latest available version.",
				Computed:    true,
			},
			"human_version": schema.StringAttribute{
				Description: "Human-readable version string.",
				Computed:    true,
			},
			"version": schema.StringAttribute{
				Description: "Installed chart version.",
				Computed:    true,
			},
			"custom_app": schema.BoolAttribute{
				Description: "Whether this is a user-defined custom app (docker compose).",
				Computed:    true,
			},
			"migrated": schema.BoolAttribute{
				Description: "Whether this app has been migrated from an older catalog.",
				Computed:    true,
			},
		},
	}
}

func (d *AppDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AppDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := d.client.GetApp(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading App",
			fmt.Sprintf("Could not read app %q: %s", config.ID.ValueString(), err),
		)
		return
	}

	config.ID = types.StringValue(app.ID)
	config.Name = types.StringValue(app.Name)
	config.State = types.StringValue(app.State)
	config.UpgradeAvailable = types.BoolValue(app.UpgradeAvailable)
	config.LatestVersion = types.StringValue(app.LatestVersion)
	config.HumanVersion = types.StringValue(app.HumanVersion)
	config.Version = types.StringValue(app.Version)
	config.CustomApp = types.BoolValue(app.CustomApp)
	config.Migrated = types.BoolValue(app.Migrated)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
