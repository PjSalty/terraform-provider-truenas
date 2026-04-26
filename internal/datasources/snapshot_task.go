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

var _ datasource.DataSource = &SnapshotTaskDataSource{}

// SnapshotTaskDataSource provides information about a periodic snapshot task.
type SnapshotTaskDataSource struct {
	client *client.Client
}

// SnapshotTaskDataSourceModel describes the data source model.
type SnapshotTaskDataSourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	Dataset        types.String `tfsdk:"dataset"`
	Recursive      types.Bool   `tfsdk:"recursive"`
	Lifetime       types.Int64  `tfsdk:"lifetime_value"`
	LifetimeUnit   types.String `tfsdk:"lifetime_unit"`
	NamingSchema   types.String `tfsdk:"naming_schema"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	AllowEmpty     types.Bool   `tfsdk:"allow_empty"`
	Exclude        types.List   `tfsdk:"exclude"`
	ScheduleMinute types.String `tfsdk:"schedule_minute"`
	ScheduleHour   types.String `tfsdk:"schedule_hour"`
	ScheduleDom    types.String `tfsdk:"schedule_dom"`
	ScheduleMonth  types.String `tfsdk:"schedule_month"`
	ScheduleDow    types.String `tfsdk:"schedule_dow"`
}

func NewSnapshotTaskDataSource() datasource.DataSource {
	return &SnapshotTaskDataSource{}
}

func (d *SnapshotTaskDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot_task"
}

func (d *SnapshotTaskDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a periodic snapshot task on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.Int64Attribute{Description: "The numeric snapshot task ID to look up.", Required: true},
			"dataset":         schema.StringAttribute{Description: "Dataset path.", Computed: true},
			"recursive":       schema.BoolAttribute{Description: "Whether the snapshot is recursive.", Computed: true},
			"lifetime_value":  schema.Int64Attribute{Description: "Lifetime quantity.", Computed: true},
			"lifetime_unit":   schema.StringAttribute{Description: "Lifetime unit (HOUR, DAY, WEEK, MONTH, YEAR).", Computed: true},
			"naming_schema":   schema.StringAttribute{Description: "Snapshot naming schema.", Computed: true},
			"enabled":         schema.BoolAttribute{Description: "Whether the task is enabled.", Computed: true},
			"allow_empty":     schema.BoolAttribute{Description: "Whether empty snapshots are allowed.", Computed: true},
			"exclude":         schema.ListAttribute{Description: "Excluded child datasets.", Computed: true, ElementType: types.StringType},
			"schedule_minute": schema.StringAttribute{Description: "Cron minute field.", Computed: true},
			"schedule_hour":   schema.StringAttribute{Description: "Cron hour field.", Computed: true},
			"schedule_dom":    schema.StringAttribute{Description: "Cron day-of-month field.", Computed: true},
			"schedule_month":  schema.StringAttribute{Description: "Cron month field.", Computed: true},
			"schedule_dow":    schema.StringAttribute{Description: "Cron day-of-week field.", Computed: true},
		},
	}
}

func (d *SnapshotTaskDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SnapshotTaskDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SnapshotTaskDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	task, err := d.client.GetSnapshotTask(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Snapshot Task",
			fmt.Sprintf("Could not read snapshot task with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(task.ID))
	config.Dataset = types.StringValue(task.Dataset)
	config.Recursive = types.BoolValue(task.Recursive)
	config.Lifetime = types.Int64Value(int64(task.Lifetime))
	config.LifetimeUnit = types.StringValue(task.LifetimeUnit)
	config.NamingSchema = types.StringValue(task.NamingSchema)
	config.Enabled = types.BoolValue(task.Enabled)
	config.AllowEmpty = types.BoolValue(task.AllowEmpty)
	config.ScheduleMinute = types.StringValue(task.Schedule.Minute)
	config.ScheduleHour = types.StringValue(task.Schedule.Hour)
	config.ScheduleDom = types.StringValue(task.Schedule.Dom)
	config.ScheduleMonth = types.StringValue(task.Schedule.Month)
	config.ScheduleDow = types.StringValue(task.Schedule.Dow)

	excludes := make([]attr.Value, 0, len(task.Exclude))
	for _, s := range task.Exclude {
		excludes = append(excludes, types.StringValue(s))
	}
	list, _ := types.ListValue(types.StringType, excludes)
	config.Exclude = list

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
