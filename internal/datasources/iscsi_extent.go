package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &ISCSIExtentDataSource{}

// ISCSIExtentDataSource provides information about an iSCSI extent.
type ISCSIExtentDataSource struct {
	client *client.Client
}

// ISCSIExtentDataSourceModel describes the data source model.
type ISCSIExtentDataSourceModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Type        types.String `tfsdk:"type"`
	Disk        types.String `tfsdk:"disk"`
	Path        types.String `tfsdk:"path"`
	Filesize    types.Int64  `tfsdk:"filesize"`
	Blocksize   types.Int64  `tfsdk:"blocksize"`
	RPM         types.String `tfsdk:"rpm"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Comment     types.String `tfsdk:"comment"`
	ReadOnly    types.Bool   `tfsdk:"readonly"`
	Xen         types.Bool   `tfsdk:"xen"`
	InsecureTPC types.Bool   `tfsdk:"insecure_tpc"`
}

func NewISCSIExtentDataSource() datasource.DataSource {
	return &ISCSIExtentDataSource{}
}

func (d *ISCSIExtentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_extent"
}

func (d *ISCSIExtentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an iSCSI extent on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.Int64Attribute{Description: "The numeric iSCSI extent ID to look up.", Required: true},
			"name":         schema.StringAttribute{Description: "The extent name.", Computed: true},
			"type":         schema.StringAttribute{Description: "The extent type (DISK or FILE).", Computed: true},
			"disk":         schema.StringAttribute{Description: "The zvol path for DISK type extents.", Computed: true},
			"path":         schema.StringAttribute{Description: "The file path for FILE type extents.", Computed: true},
			"filesize":     schema.Int64Attribute{Description: "File size in bytes.", Computed: true},
			"blocksize":    schema.Int64Attribute{Description: "Block size in bytes.", Computed: true},
			"rpm":          schema.StringAttribute{Description: "Reported RPM.", Computed: true},
			"enabled":      schema.BoolAttribute{Description: "Whether the extent is enabled.", Computed: true},
			"comment":      schema.StringAttribute{Description: "Comment.", Computed: true},
			"readonly":     schema.BoolAttribute{Description: "Whether the extent is read-only.", Computed: true},
			"xen":          schema.BoolAttribute{Description: "Xen compatibility mode.", Computed: true},
			"insecure_tpc": schema.BoolAttribute{Description: "Whether insecure TPC is enabled.", Computed: true},
		},
	}
}

func (d *ISCSIExtentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ISCSIExtentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ISCSIExtentDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	extent, err := d.client.GetISCSIExtent(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Extent",
			fmt.Sprintf("Could not read iSCSI extent with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(extent.ID))
	config.Name = types.StringValue(extent.Name)
	config.Type = types.StringValue(extent.Type)
	config.Disk = types.StringValue(extent.GetDisk())
	config.Path = types.StringValue(extent.Path)
	config.Filesize = types.Int64Value(extent.GetFilesize())
	config.Blocksize = types.Int64Value(int64(extent.Blocksize))
	config.RPM = types.StringValue(extent.RPM)
	config.Enabled = types.BoolValue(extent.Enabled)
	config.Comment = types.StringValue(extent.Comment)
	config.ReadOnly = types.BoolValue(extent.ReadOnly)
	config.Xen = types.BoolValue(extent.Xen)
	config.InsecureTPC = types.BoolValue(extent.InsecureTPC)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
