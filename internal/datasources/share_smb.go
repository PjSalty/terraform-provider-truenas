package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &ShareSMBDataSource{}

// ShareSMBDataSource provides information about an SMB share.
type ShareSMBDataSource struct {
	client *client.Client
}

// ShareSMBDataSourceModel describes the data source model.
type ShareSMBDataSourceModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Path      types.String `tfsdk:"path"`
	Name      types.String `tfsdk:"name"`
	Comment   types.String `tfsdk:"comment"`
	Browsable types.Bool   `tfsdk:"browsable"`
	ReadOnly  types.Bool   `tfsdk:"read_only"`
	ABE       types.Bool   `tfsdk:"access_based_share_enumeration"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	Purpose   types.String `tfsdk:"purpose"`
}

func NewShareSMBDataSource() datasource.DataSource {
	return &ShareSMBDataSource{}
}

func (d *ShareSMBDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_share_smb"
}

func (d *ShareSMBDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an SMB share on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric SMB share ID to look up.",
				Required:    true,
			},
			"path": schema.StringAttribute{
				Description: "Exported filesystem path.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Share name (network-visible).",
				Computed:    true,
			},
			"comment": schema.StringAttribute{
				Description: "Free-form comment.",
				Computed:    true,
			},
			"browsable": schema.BoolAttribute{
				Description: "Whether the share appears in network browsing.",
				Computed:    true,
			},
			"read_only": schema.BoolAttribute{
				Description: "Whether the share is read-only.",
				Computed:    true,
			},
			"access_based_share_enumeration": schema.BoolAttribute{
				Description: "Whether ABE (access-based share enumeration) is enabled.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the share is enabled.",
				Computed:    true,
			},
			"purpose": schema.StringAttribute{
				Description: "SMB share purpose preset.",
				Computed:    true,
			},
		},
	}
}

func (d *ShareSMBDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ShareSMBDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ShareSMBDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	share, err := d.client.GetSMBShare(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading SMB Share",
			fmt.Sprintf("Could not read SMB share with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(share.ID))
	config.Path = types.StringValue(share.Path)
	config.Name = types.StringValue(share.Name)
	config.Comment = types.StringValue(share.Comment)
	config.Browsable = types.BoolValue(share.Browsable)
	config.ReadOnly = types.BoolValue(share.ReadOnly)
	config.ABE = types.BoolValue(share.ABE)
	config.Enabled = types.BoolValue(share.Enabled)
	config.Purpose = types.StringValue(share.Purpose)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
