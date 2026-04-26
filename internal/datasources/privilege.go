package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &PrivilegeDataSource{}

// PrivilegeDataSource provides information about a TrueNAS privilege.
type PrivilegeDataSource struct {
	client *client.Client
}

// PrivilegeDataSourceModel describes the data source model.
type PrivilegeDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	BuiltinName types.String `tfsdk:"builtin_name"`
	LocalGroups types.List   `tfsdk:"local_groups"`
	DSGroups    types.List   `tfsdk:"ds_groups"`
	Roles       types.List   `tfsdk:"roles"`
	WebShell    types.Bool   `tfsdk:"web_shell"`
}

func NewPrivilegeDataSource() datasource.DataSource {
	return &PrivilegeDataSource{}
}

func (d *PrivilegeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_privilege"
}

func (d *PrivilegeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a TrueNAS RBAC privilege.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric privilege ID to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The privilege name.",
				Computed:    true,
			},
			"builtin_name": schema.StringAttribute{
				Description: "Built-in privilege name, if this is a built-in privilege.",
				Computed:    true,
			},
			"local_groups": schema.ListAttribute{
				Description: "Local group GIDs granted this privilege.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"ds_groups": schema.ListAttribute{
				Description: "Directory service groups (GIDs or SIDs) granted this privilege.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"roles": schema.ListAttribute{
				Description: "Roles granted to this privilege.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"web_shell": schema.BoolAttribute{
				Description: "Whether web shell access is enabled.",
				Computed:    true,
			},
		},
	}
}

func (d *PrivilegeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PrivilegeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PrivilegeDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	p, err := d.client.GetPrivilege(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Privilege",
			fmt.Sprintf("Could not read privilege with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(p.ID))
	config.Name = types.StringValue(p.Name)
	if p.BuiltinName != nil {
		config.BuiltinName = types.StringValue(*p.BuiltinName)
	} else {
		config.BuiltinName = types.StringNull()
	}
	config.WebShell = types.BoolValue(p.WebShell)

	gids := p.LocalGroupGIDs()
	gidVals := make([]int64, 0, len(gids))
	for _, g := range gids {
		gidVals = append(gidVals, int64(g))
	}
	lg, lgDiags := types.ListValueFrom(ctx, types.Int64Type, gidVals)
	resp.Diagnostics.Append(lgDiags...)
	config.LocalGroups = lg

	ds, dsDiags := types.ListValueFrom(ctx, types.StringType, p.DSGroupStrings())
	resp.Diagnostics.Append(dsDiags...)
	config.DSGroups = ds

	roles, rDiags := types.ListValueFrom(ctx, types.StringType, p.Roles)
	resp.Diagnostics.Append(rDiags...)
	config.Roles = roles

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
