package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &GroupDataSource{}

// GroupDataSource provides information about a TrueNAS group.
type GroupDataSource struct {
	client *client.Client
}

// GroupDataSourceModel describes the data source model.
type GroupDataSourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	GID          types.Int64  `tfsdk:"gid"`
	SMB          types.Bool   `tfsdk:"smb"`
	Builtin      types.Bool   `tfsdk:"builtin"`
	SudoCommands types.String `tfsdk:"sudo_commands"`
}

func NewGroupDataSource() datasource.DataSource {
	return &GroupDataSource{}
}

func (d *GroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *GroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a group on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The group ID.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The group name to look up.",
				Required:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "The UNIX GID.",
				Computed:    true,
			},
			"smb": schema.BoolAttribute{
				Description: "Whether the group has SMB access.",
				Computed:    true,
			},
			"builtin": schema.BoolAttribute{
				Description: "Whether this is a built-in system group.",
				Computed:    true,
			},
			"sudo_commands": schema.StringAttribute{
				Description: "Comma-separated list of sudo commands allowed.",
				Computed:    true,
			},
		},
	}
}

func (d *GroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config GroupDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()

	group, err := d.client.GetGroupByName(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Group",
			fmt.Sprintf("Could not find group %q: %s", name, err),
		)
		return
	}

	config.ID = types.Int64Value(int64(group.ID))
	config.Name = types.StringValue(group.Name)
	config.GID = types.Int64Value(int64(group.GID))
	config.SMB = types.BoolValue(group.SMB)
	config.Builtin = types.BoolValue(group.Builtin)

	// Join sudo commands into a comma-separated string
	sudoCmds := ""
	for i, cmd := range group.SudoCommands {
		if i > 0 {
			sudoCmds += ","
		}
		sudoCmds += cmd
	}
	config.SudoCommands = types.StringValue(sudoCmds)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
