package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &DiskDataSource{}

// DiskDataSource provides disk information from TrueNAS.
type DiskDataSource struct {
	client *client.Client
}

// DiskDataSourceModel describes the data source model.
type DiskDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Serial      types.String `tfsdk:"serial"`
	Size        types.Int64  `tfsdk:"size"`
	Type        types.String `tfsdk:"type"`
	Pool        types.String `tfsdk:"pool"`
	Model       types.String `tfsdk:"model"`
	Description types.String `tfsdk:"description"`
	Bus         types.String `tfsdk:"bus"`
	Identifier  types.String `tfsdk:"identifier"`
	Devname     types.String `tfsdk:"devname"`
}

func NewDiskDataSource() datasource.DataSource {
	return &DiskDataSource{}
}

func (d *DiskDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_disk"
}

func (d *DiskDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a disk on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The disk device name to look up (e.g., sda).",
				Required:    true,
			},
			"serial": schema.StringAttribute{
				Description: "The disk serial number.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "The disk size in bytes.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The disk type (SSD, HDD).",
				Computed:    true,
			},
			"pool": schema.StringAttribute{
				Description: "The pool this disk belongs to, if any.",
				Computed:    true,
			},
			"model": schema.StringAttribute{
				Description: "The disk model.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The disk description.",
				Computed:    true,
			},
			"bus": schema.StringAttribute{
				Description: "The disk bus type.",
				Computed:    true,
			},
			"identifier": schema.StringAttribute{
				Description: "The disk identifier.",
				Computed:    true,
			},
			"devname": schema.StringAttribute{
				Description: "The device name.",
				Computed:    true,
			},
		},
	}
}

func (d *DiskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DiskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DiskDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := config.Name.ValueString()

	// List all disks and find by name
	disks, err := d.client.ListDisks(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Listing Disks",
			fmt.Sprintf("Could not list disks: %s", err),
		)
		return
	}

	found := false
	for _, disk := range disks {
		if disk.Name == name {
			d.mapDiskToModel(&disk, &config)
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			"Disk Not Found",
			fmt.Sprintf("No disk found with name %q", name),
		)
		return
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}

func (d *DiskDataSource) mapDiskToModel(disk *client.Disk, model *DiskDataSourceModel) {
	model.Name = types.StringValue(disk.Name)
	model.Serial = types.StringValue(disk.Serial)
	model.Size = types.Int64Value(disk.Size)
	model.Type = types.StringValue(disk.Type)
	model.Model = types.StringValue(disk.Model)
	model.Description = types.StringValue(disk.Description)
	model.Bus = types.StringValue(disk.Bus)
	model.Identifier = types.StringValue(disk.Identifier)
	model.Devname = types.StringValue(disk.Devname)

	if disk.Pool != nil {
		model.Pool = types.StringValue(*disk.Pool)
	} else {
		model.Pool = types.StringValue("")
	}
}
