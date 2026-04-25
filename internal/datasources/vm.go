package datasources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

var _ datasource.DataSource = &VMDataSource{}

// VMDataSource provides information about a TrueNAS SCALE virtual machine.
type VMDataSource struct {
	client *client.Client
}

// VMDataSourceModel describes the data source model.
type VMDataSourceModel struct {
	ID               types.Int64  `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Vcpus            types.Int64  `tfsdk:"vcpus"`
	Cores            types.Int64  `tfsdk:"cores"`
	Threads          types.Int64  `tfsdk:"threads"`
	Memory           types.Int64  `tfsdk:"memory"`
	MinMemory        types.Int64  `tfsdk:"min_memory"`
	Bootloader       types.String `tfsdk:"bootloader"`
	BootloaderOvmf   types.String `tfsdk:"bootloader_ovmf"`
	Autostart        types.Bool   `tfsdk:"autostart"`
	Time             types.String `tfsdk:"time"`
	ShutdownTimeout  types.Int64  `tfsdk:"shutdown_timeout"`
	CPUMode          types.String `tfsdk:"cpu_mode"`
	CPUModel         types.String `tfsdk:"cpu_model"`
	EnableSecureBoot types.Bool   `tfsdk:"enable_secure_boot"`
	State            types.String `tfsdk:"state"`
	UUID             types.String `tfsdk:"uuid"`
}

func NewVMDataSource() datasource.DataSource {
	return &VMDataSource{}
}

func (d *VMDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (d *VMDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides information about a virtual machine on TrueNAS SCALE.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Description: "The numeric VM ID to look up.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The VM name.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "VM description.",
				Computed:    true,
			},
			"vcpus": schema.Int64Attribute{
				Description: "Number of virtual CPU sockets.",
				Computed:    true,
			},
			"cores": schema.Int64Attribute{
				Description: "Number of cores per socket.",
				Computed:    true,
			},
			"threads": schema.Int64Attribute{
				Description: "Number of threads per core.",
				Computed:    true,
			},
			"memory": schema.Int64Attribute{
				Description: "Memory in bytes.",
				Computed:    true,
			},
			"min_memory": schema.Int64Attribute{
				Description: "Minimum memory in bytes (for ballooning).",
				Computed:    true,
			},
			"bootloader": schema.StringAttribute{
				Description: "Bootloader type.",
				Computed:    true,
			},
			"bootloader_ovmf": schema.StringAttribute{
				Description: "OVMF firmware variant.",
				Computed:    true,
			},
			"autostart": schema.BoolAttribute{
				Description: "Whether the VM starts on boot.",
				Computed:    true,
			},
			"time": schema.StringAttribute{
				Description: "Clock configuration (LOCAL, UTC).",
				Computed:    true,
			},
			"shutdown_timeout": schema.Int64Attribute{
				Description: "Shutdown timeout in seconds.",
				Computed:    true,
			},
			"cpu_mode": schema.StringAttribute{
				Description: "CPU mode.",
				Computed:    true,
			},
			"cpu_model": schema.StringAttribute{
				Description: "Specific CPU model (when cpu_mode is CUSTOM).",
				Computed:    true,
			},
			"enable_secure_boot": schema.BoolAttribute{
				Description: "Whether secure boot is enabled.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "Current runtime state (RUNNING, STOPPED, etc.).",
				Computed:    true,
			},
			"uuid": schema.StringAttribute{
				Description: "VM UUID.",
				Computed:    true,
			},
		},
	}
}

func (d *VMDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *VMDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config VMDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	vm, err := d.client.GetVM(ctx, int(config.ID.ValueInt64()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading VM",
			fmt.Sprintf("Could not read VM with ID %d: %s", config.ID.ValueInt64(), err),
		)
		return
	}

	config.ID = types.Int64Value(int64(vm.ID))
	config.Name = types.StringValue(vm.Name)
	config.Description = types.StringValue(vm.Description)
	config.Vcpus = types.Int64Value(int64(vm.Vcpus))
	config.Cores = types.Int64Value(int64(vm.Cores))
	config.Threads = types.Int64Value(int64(vm.Threads))
	config.Memory = types.Int64Value(vm.Memory)
	if vm.MinMemory != nil {
		config.MinMemory = types.Int64Value(*vm.MinMemory)
	} else {
		config.MinMemory = types.Int64Null()
	}
	config.Bootloader = types.StringValue(vm.Bootloader)
	config.BootloaderOvmf = types.StringValue(vm.BootloaderOvmf)
	config.Autostart = types.BoolValue(vm.Autostart)
	config.Time = types.StringValue(vm.Time)
	config.ShutdownTimeout = types.Int64Value(int64(vm.ShutdownTimeout))
	config.CPUMode = types.StringValue(vm.CPUMode)
	if vm.CPUModel != nil {
		config.CPUModel = types.StringValue(*vm.CPUModel)
	} else {
		config.CPUModel = types.StringNull()
	}
	config.EnableSecureBoot = types.BoolValue(vm.EnableSecureBoot)

	if vm.Status != nil {
		config.State = types.StringValue(vm.Status.State)
	} else {
		config.State = types.StringNull()
	}
	if vm.UUID != nil {
		config.UUID = types.StringValue(*vm.UUID)
	} else {
		config.UUID = types.StringNull()
	}

	diags = resp.State.Set(ctx, config)
	resp.Diagnostics.Append(diags...)
}
