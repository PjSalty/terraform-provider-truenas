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

var _ datasource.DataSource = &ISCSIInitiatorDataSource{}

// ISCSIInitiatorDataSource provides information about an iSCSI initiator group.
type ISCSIInitiatorDataSource struct {
	client *client.Client
}

// ISCSIInitiatorDataSourceModel describes the data source model.
type ISCSIInitiatorDataSourceModel struct {
	ID         types.Int64  `tfsdk:"id"`
	Initiators types.List   `tfsdk:"initiators"`
	Comment    types.String `tfsdk:"comment"`
}

func NewISCSIInitiatorDataSource() datasource.DataSource {
	return &ISCSIInitiatorDataSource{}
}

func (d *ISCSIInitiatorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iscsi_initiator"
}

func (d *ISCSIInitiatorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about an iSCSI authorized initiator group on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric initiator group ID to look up.",
				Required:    true,
			},
			"initiators": schema.ListAttribute{
				Description: "List of initiator IQNs allowed to connect.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"comment": schema.StringAttribute{
				Description: "Comment for the initiator group.",
				Computed:    true,
			},
		},
	}
}

func (d *ISCSIInitiatorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ISCSIInitiatorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ISCSIInitiatorDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	init, err := d.client.GetISCSIInitiator(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading iSCSI Initiator",
			fmt.Sprintf("Could not read iSCSI initiator with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(init.ID))
	config.Comment = types.StringValue(init.Comment)

	vals := make([]attr.Value, 0, len(init.Initiators))
	for _, s := range init.Initiators {
		vals = append(vals, types.StringValue(s))
	}
	list, _ := types.ListValue(types.StringType, vals)
	config.Initiators = list

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
