package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &CronJobDataSource{}

// CronJobDataSource provides information about a cron job.
type CronJobDataSource struct {
	client *client.Client
}

// CronJobDataSourceModel describes the data source model.
type CronJobDataSourceModel struct {
	ID             types.Int64  `tfsdk:"id"`
	User           types.String `tfsdk:"user"`
	Command        types.String `tfsdk:"command"`
	Description    types.String `tfsdk:"description"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Stdout         types.Bool   `tfsdk:"stdout"`
	Stderr         types.Bool   `tfsdk:"stderr"`
	ScheduleMinute types.String `tfsdk:"schedule_minute"`
	ScheduleHour   types.String `tfsdk:"schedule_hour"`
	ScheduleDom    types.String `tfsdk:"schedule_dom"`
	ScheduleMonth  types.String `tfsdk:"schedule_month"`
	ScheduleDow    types.String `tfsdk:"schedule_dow"`
}

func NewCronJobDataSource() datasource.DataSource {
	return &CronJobDataSource{}
}

func (d *CronJobDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cronjob"
}

func (d *CronJobDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a cron job on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric cron job ID to look up.",
				Required:    true,
			},
			"user": schema.StringAttribute{
				Description: "User the job runs as.",
				Computed:    true,
			},
			"command": schema.StringAttribute{
				Description: "Command to run.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Free-form description.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the job is enabled.",
				Computed:    true,
			},
			"stdout": schema.BoolAttribute{
				Description: "Whether stdout is captured.",
				Computed:    true,
			},
			"stderr": schema.BoolAttribute{
				Description: "Whether stderr is captured.",
				Computed:    true,
			},
			"schedule_minute": schema.StringAttribute{
				Description: "Cron minute field.",
				Computed:    true,
			},
			"schedule_hour": schema.StringAttribute{
				Description: "Cron hour field.",
				Computed:    true,
			},
			"schedule_dom": schema.StringAttribute{
				Description: "Cron day-of-month field.",
				Computed:    true,
			},
			"schedule_month": schema.StringAttribute{
				Description: "Cron month field.",
				Computed:    true,
			},
			"schedule_dow": schema.StringAttribute{
				Description: "Cron day-of-week field.",
				Computed:    true,
			},
		},
	}
}

func (d *CronJobDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CronJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config CronJobDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, err := d.client.GetCronJob(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Cron Job",
			fmt.Sprintf("Could not read cron job with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(job.ID))
	config.User = types.StringValue(job.User)
	config.Command = types.StringValue(job.Command)
	config.Description = types.StringValue(job.Description)
	config.Enabled = types.BoolValue(job.Enabled)
	config.Stdout = types.BoolValue(job.Stdout)
	config.Stderr = types.BoolValue(job.Stderr)
	config.ScheduleMinute = types.StringValue(job.Schedule.Minute)
	config.ScheduleHour = types.StringValue(job.Schedule.Hour)
	config.ScheduleDom = types.StringValue(job.Schedule.Dom)
	config.ScheduleMonth = types.StringValue(job.Schedule.Month)
	config.ScheduleDow = types.StringValue(job.Schedule.Dow)

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
