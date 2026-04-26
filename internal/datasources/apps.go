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

var _ datasource.DataSource = &AppsDataSource{}

// AppsDataSource provides a listing of deployed TrueNAS SCALE apps.
type AppsDataSource struct {
	client *client.Client
}

// AppsDataSourceModel is the top-level data source model.
type AppsDataSourceModel struct {
	Apps types.List `tfsdk:"apps"`
}

func NewAppsDataSource() datasource.DataSource {
	return &AppsDataSource{}
}

func (d *AppsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_apps"
}

// appsListAttrTypes returns the per-element object types for the list.
func appsListAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"id":                types.StringType,
		"name":              types.StringType,
		"state":             types.StringType,
		"version":           types.StringType,
		"human_version":     types.StringType,
		"latest_version":    types.StringType,
		"upgrade_available": types.BoolType,
		"custom_app":        types.BoolType,
	}
}

func (d *AppsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all deployed apps on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"apps": schema.ListNestedAttribute{
				Description: "List of deployed apps.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "App ID (name).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "App name.",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "Current state.",
							Computed:    true,
						},
						"version": schema.StringAttribute{
							Description: "Installed version.",
							Computed:    true,
						},
						"human_version": schema.StringAttribute{
							Description: "Human-readable version.",
							Computed:    true,
						},
						"latest_version": schema.StringAttribute{
							Description: "Latest available version.",
							Computed:    true,
						},
						"upgrade_available": schema.BoolAttribute{
							Description: "Whether an upgrade is available.",
							Computed:    true,
						},
						"custom_app": schema.BoolAttribute{
							Description: "Whether this is a custom (compose) app.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *AppsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AppsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	apps, err := d.client.ListApps(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Apps",
			fmt.Sprintf("Could not list apps: %s", err),
		)
		return
	}

	objType := types.ObjectType{AttrTypes: appsListAttrTypes()}
	elems := make([]attr.Value, 0, len(apps))
	for _, a := range apps {
		obj, objDiags := types.ObjectValue(appsListAttrTypes(), map[string]attr.Value{
			"id":                types.StringValue(a.ID),
			"name":              types.StringValue(a.Name),
			"state":             types.StringValue(a.State),
			"version":           types.StringValue(a.Version),
			"human_version":     types.StringValue(a.HumanVersion),
			"latest_version":    types.StringValue(a.LatestVersion),
			"upgrade_available": types.BoolValue(a.UpgradeAvailable),
			"custom_app":        types.BoolValue(a.CustomApp),
		})
		resp.Diagnostics.Append(objDiags...)
		elems = append(elems, obj)
	}

	list, lDiags := types.ListValue(objType, elems)
	resp.Diagnostics.Append(lDiags...)

	model := AppsDataSourceModel{Apps: list}
	diags := resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}
