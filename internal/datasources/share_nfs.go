package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &ShareNFSDataSource{}

// ShareNFSDataSource provides information about an NFS share.
type ShareNFSDataSource struct {
	client *client.Client
}

// ShareNFSDataSourceModel describes the data source model.
type ShareNFSDataSourceModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Path         types.String `tfsdk:"path"`
	Aliases      types.List   `tfsdk:"aliases"`
	Comment      types.String `tfsdk:"comment"`
	Hosts        types.List   `tfsdk:"hosts"`
	ReadOnly     types.Bool   `tfsdk:"read_only"`
	MaprootUser  types.String `tfsdk:"maproot_user"`
	MaprootGroup types.String `tfsdk:"maproot_group"`
	MapallUser   types.String `tfsdk:"mapall_user"`
	MapallGroup  types.String `tfsdk:"mapall_group"`
	Security     types.List   `tfsdk:"security"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Networks     types.List   `tfsdk:"networks"`
}

func NewShareNFSDataSource() datasource.DataSource {
	return &ShareNFSDataSource{}
}

func (d *ShareNFSDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_nfs"
}

func (d *ShareNFSDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an NFS share on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric NFS share ID to look up.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "Exported filesystem path.",
				Computed:    true,
			},
			"aliases": schema.ListAttribute{
				Description: "Export path aliases.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"comment": schema.StringAttribute{
				Description: "Free-form comment.",
				Computed:    true,
			},
			"hosts": schema.ListAttribute{
				Description: "Authorized client hosts.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether the share is read-only.",
				Computed:    true,
			},
			"maproot_user": schema.StringAttribute{
				Description: "Map root to this user.",
				Computed:    true,
			},
			"maproot_group": schema.StringAttribute{
				Description: "Map root to this group.",
				Computed:    true,
			},
			"mapall_user": schema.StringAttribute{
				Description: "Map all requests to this user.",
				Computed:    true,
			},
			"mapall_group": schema.StringAttribute{
				Description: "Map all requests to this group.",
				Computed:    true,
			},
			"security": schema.ListAttribute{
				Description: "Allowed security flavors.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the share is enabled.",
				Computed:    true,
			},
			"networks": schema.ListAttribute{
				Description: "Authorized networks (CIDR notation).",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *ShareNFSDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ShareNFSDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ShareNFSDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	share, err := d.client.GetNFSShare(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading NFS Share",
			fmt.Sprintf("Could not read NFS share with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(share.ID))
	config.Path = types.StringValue(share.Path)
	config.Comment = types.StringValue(share.Comment)
	config.ReadOnly = types.BoolValue(share.ReadOnly)
	config.MaprootUser = types.StringValue(share.MaprootUser)
	config.MaprootGroup = types.StringValue(share.MaprootGroup)
	config.MapallUser = types.StringValue(share.MapallUser)
	config.MapallGroup = types.StringValue(share.MapallGroup)
	config.Enabled = types.BoolValue(share.Enabled)

	aliases, aDiags := types.ListValueFrom(ctx, types.StringType, share.Aliases)
	resp.Diagnostics.Append(aDiags...)
	config.Aliases = aliases

	hosts, hDiags := types.ListValueFrom(ctx, types.StringType, share.Hosts)
	resp.Diagnostics.Append(hDiags...)
	config.Hosts = hosts

	security, sDiags := types.ListValueFrom(ctx, types.StringType, share.Security)
	resp.Diagnostics.Append(sDiags...)
	config.Security = security

	networks, nDiags := types.ListValueFrom(ctx, types.StringType, share.Networks)
	resp.Diagnostics.Append(nDiags...)
	config.Networks = networks

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
