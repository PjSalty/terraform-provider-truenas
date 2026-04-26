package datasources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &DatasetDataSource{}

// DatasetDataSource provides information about a ZFS dataset.
type DatasetDataSource struct {
	client *client.Client
}

// DatasetDataSourceModel describes the data source model.
type DatasetDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Pool          types.String `tfsdk:"pool"`
	Type          types.String `tfsdk:"type"`
	MountPoint    types.String `tfsdk:"mount_point"`
	Compression   types.String `tfsdk:"compression"`
	Atime         types.String `tfsdk:"atime"`
	Deduplication types.String `tfsdk:"deduplication"`
	Quota         types.Int64  `tfsdk:"quota"`
	Refquota      types.Int64  `tfsdk:"refquota"`
	Comments      types.String `tfsdk:"comments"`
	Sync          types.String `tfsdk:"sync"`
	Readonly      types.String `tfsdk:"readonly"`
	RecordSize    types.String `tfsdk:"record_size"`
}

func NewDatasetDataSource() datasource.DataSource {
	return &DatasetDataSource{}
}

func (d *DatasetDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dataset"
}

func (d *DatasetDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a ZFS dataset on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The full dataset path (e.g., tank/mydata).",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The dataset name.",
				Computed:    true,
			},
			"pool": schema.StringAttribute{
				Description: "The pool the dataset belongs to.",
				Computed:    true,
			},
			"type": schema.StringAttribute{
				Description: "The dataset type (FILESYSTEM or VOLUME).",
				Computed:    true,
			},
			"mount_point": schema.StringAttribute{
				Description: "The mount point.",
				Computed:    true,
			},
			"compression": schema.StringAttribute{
				Description: "The compression algorithm.",
				Computed:    true,
			},
			"atime": schema.StringAttribute{
				Description: "Access time setting.",
				Computed:    true,
			},
			"deduplication": schema.StringAttribute{
				Description: "Deduplication setting.",
				Computed:    true,
			},
			"quota": schema.Int64Attribute{
				Description: "Dataset quota in bytes.",
				Computed:    true,
			},
			"refquota": schema.Int64Attribute{
				Description: "Dataset reference quota in bytes.",
				Computed:    true,
			},
			"comments": schema.StringAttribute{
				Description: "Dataset comments.",
				Computed:    true,
			},
			"sync": schema.StringAttribute{
				Description: "Sync write behavior.",
				Computed:    true,
			},
			"readonly": schema.StringAttribute{
				Description: "Read-only setting.",
				Computed:    true,
			},
			"record_size": schema.StringAttribute{
				Description: "Record size.",
				Computed:    true,
			},
		},
	}
}

func (d *DatasetDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DatasetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DatasetDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	dataset, err := d.client.GetDataset(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Dataset",
			fmt.Sprintf("Could not read dataset %q: %s", config.ID.ValueString(), err),
		)
		return
	}

	config.ID = types.StringValue(dataset.ID)
	config.Name = types.StringValue(dataset.Name)
	config.Pool = types.StringValue(dataset.Pool)
	config.Type = types.StringValue(dataset.Type)
	config.MountPoint = types.StringValue(dataset.MountPoint)

	if dataset.Compression != nil {
		config.Compression = types.StringValue(dataset.Compression.Value)
	}
	if dataset.Atime != nil {
		config.Atime = types.StringValue(dataset.Atime.Value)
	}
	if dataset.Deduplication != nil {
		config.Deduplication = types.StringValue(dataset.Deduplication.Value)
	}
	if dataset.Quota != nil {
		if v, err := strconv.ParseInt(dataset.Quota.Rawvalue, 10, 64); err == nil {
			config.Quota = types.Int64Value(v)
		}
	}
	if dataset.Refquota != nil {
		if v, err := strconv.ParseInt(dataset.Refquota.Rawvalue, 10, 64); err == nil {
			config.Refquota = types.Int64Value(v)
		}
	}
	if dataset.Comments != nil {
		config.Comments = types.StringValue(dataset.Comments.Value)
	}
	if dataset.Sync != nil {
		config.Sync = types.StringValue(dataset.Sync.Value)
	}
	if dataset.Readonly != nil {
		config.Readonly = types.StringValue(dataset.Readonly.Value)
	}
	if dataset.RecordSize != nil {
		config.RecordSize = types.StringValue(dataset.RecordSize.Value)
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
