package resources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/planhelpers"
)

var (
	_ resource.Resource                = &VMResource{}
	_ resource.ResourceWithImportState = &VMResource{}
	_ resource.ResourceWithModifyPlan  = &VMResource{}
)

// VMResource manages a TrueNAS SCALE virtual machine.
type VMResource struct {
	client *client.Client
}

// VMResourceModel describes the resource data model.
type VMResourceModel struct {
	ID                    types.String   `tfsdk:"id"`
	Name                  types.String   `tfsdk:"name"`
	Description           types.String   `tfsdk:"description"`
	Vcpus                 types.Int64    `tfsdk:"vcpus"`
	Cores                 types.Int64    `tfsdk:"cores"`
	Threads               types.Int64    `tfsdk:"threads"`
	Memory                types.Int64    `tfsdk:"memory"`
	MinMemory             types.Int64    `tfsdk:"min_memory"`
	Bootloader            types.String   `tfsdk:"bootloader"`
	BootloaderOvmf        types.String   `tfsdk:"bootloader_ovmf"`
	Autostart             types.Bool     `tfsdk:"autostart"`
	HideFromMsr           types.Bool     `tfsdk:"hide_from_msr"`
	EnsureDisplayDevice   types.Bool     `tfsdk:"ensure_display_device"`
	Time                  types.String   `tfsdk:"time"`
	ShutdownTimeout       types.Int64    `tfsdk:"shutdown_timeout"`
	ArchType              types.String   `tfsdk:"arch_type"`
	MachineType           types.String   `tfsdk:"machine_type"`
	UUID                  types.String   `tfsdk:"uuid"`
	CommandLineArgs       types.String   `tfsdk:"command_line_args"`
	CPUMode               types.String   `tfsdk:"cpu_mode"`
	CPUModel              types.String   `tfsdk:"cpu_model"`
	Cpuset                types.String   `tfsdk:"cpuset"`
	Nodeset               types.String   `tfsdk:"nodeset"`
	PinVcpus              types.Bool     `tfsdk:"pin_vcpus"`
	SuspendOnSnapshot     types.Bool     `tfsdk:"suspend_on_snapshot"`
	TrustedPlatformModule types.Bool     `tfsdk:"trusted_platform_module"`
	HypervEnlightenments  types.Bool     `tfsdk:"hyperv_enlightenments"`
	EnableSecureBoot      types.Bool     `tfsdk:"enable_secure_boot"`
	Status                types.String   `tfsdk:"status"`
	Timeouts              timeouts.Value `tfsdk:"timeouts"`
}

// NewVMResource returns the resource implementation for truenas_vm.
func NewVMResource() resource.Resource {
	return &VMResource{}
}

func (r *VMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *VMResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a TrueNAS SCALE virtual machine via the /vm API. " +
			"Devices (disks, NICs, CDROMs, displays) are managed independently via truenas_vm_device. " +
			"Default timeouts: 20m for create/update/delete (VM start/stop can be slow).",
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The numeric ID of the VM.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the VM. Only alphanumeric characters are allowed — " +
					"no spaces, hyphens, underscores, or punctuation.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 150),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+$`),
						"VM name must contain only alphanumeric characters",
					),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description for the VM.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Validators: []validator.String{
					stringvalidator.LengthAtMost(1024),
				},
			},
			"vcpus": schema.Int64Attribute{
				Description: "Number of virtual CPU sockets (1-16).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 16),
				},
			},
			"cores": schema.Int64Attribute{
				Description: "Number of cores per CPU socket (1-254).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 254),
				},
			},
			"threads": schema.Int64Attribute{
				Description: "Number of threads per core (1-254).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 254),
				},
			},
			"memory": schema.Int64Attribute{
				Description: "Memory allocated to the VM, in MiB. Minimum 20 MiB. Max 4 TiB.",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.Between(20, 4194304),
				},
			},
			"min_memory": schema.Int64Attribute{
				Description: "Optional minimum memory (MiB) for memory ballooning.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(20, 4194304),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"bootloader": schema.StringAttribute{
				Description: "Bootloader to use: UEFI or UEFI_CSM.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("UEFI"),
				Validators: []validator.String{
					stringvalidator.OneOf("UEFI", "UEFI_CSM"),
				},
			},
			"bootloader_ovmf": schema.StringAttribute{
				Description: "OVMF firmware file name.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("OVMF_CODE.fd"),
			},
			"autostart": schema.BoolAttribute{
				Description: "Whether the VM should start automatically on host boot.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"hide_from_msr": schema.BoolAttribute{
				Description: "Hide the KVM hypervisor from MSR-based discovery (for GPU passthrough).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ensure_display_device": schema.BoolAttribute{
				Description: "Ensure the VM always has a display device attached.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"time": schema.StringAttribute{
				Description: "VM clock source: LOCAL or UTC.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LOCAL"),
				Validators: []validator.String{
					stringvalidator.OneOf("LOCAL", "UTC"),
				},
			},
			"shutdown_timeout": schema.Int64Attribute{
				Description: "Seconds to wait for the VM to cleanly shut down before forcing power off (5-300).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(90),
				Validators: []validator.Int64{
					int64validator.Between(5, 300),
				},
			},
			"arch_type": schema.StringAttribute{
				Description: "Guest architecture (nullable; system chooses default when unset).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"machine_type": schema.StringAttribute{
				Description: "Guest machine type (nullable; system chooses default when unset).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"uuid": schema.StringAttribute{
				Description: "VM UUID. If unset, TrueNAS generates one.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"command_line_args": schema.StringAttribute{
				Description: "Additional QEMU command line arguments.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"cpu_mode": schema.StringAttribute{
				Description: "CPU mode: CUSTOM, HOST-MODEL, or HOST-PASSTHROUGH.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("CUSTOM"),
				Validators: []validator.String{
					stringvalidator.OneOf("CUSTOM", "HOST-MODEL", "HOST-PASSTHROUGH"),
				},
			},
			"cpu_model": schema.StringAttribute{
				Description: "CPU model when cpu_mode is CUSTOM (nullable).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cpuset": schema.StringAttribute{
				Description: "Host CPU set to pin the VM to (nullable).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nodeset": schema.StringAttribute{
				Description: "NUMA node set to pin the VM to (nullable).",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"pin_vcpus": schema.BoolAttribute{
				Description: "Pin vCPUs to host CPUs listed in cpuset.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"suspend_on_snapshot": schema.BoolAttribute{
				Description: "Suspend the VM automatically while periodic snapshots run.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"trusted_platform_module": schema.BoolAttribute{
				Description: "Attach a virtual TPM to the VM.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"hyperv_enlightenments": schema.BoolAttribute{
				Description: "Enable Hyper-V enlightenments for Windows guests.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"enable_secure_boot": schema.BoolAttribute{
				Description: "Enable UEFI Secure Boot.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"status": schema.StringAttribute{
				Description: "Current VM state (RUNNING, STOPPED, etc.). Read-only.",
				Computed:    true,
			},
		},
	}
}

func (r *VMResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	r.client = c
}

// buildCreateRequest assembles a VMCreateRequest from the plan model. Optional fields
// that are unknown/null are omitted so the API applies its own defaults.
func buildVMCreateRequest(plan *VMResourceModel) *client.VMCreateRequest {
	req := &client.VMCreateRequest{
		Name:   plan.Name.ValueString(),
		Memory: plan.Memory.ValueInt64(),
	}

	setStr := func(v types.String, dst **string) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		s := v.ValueString()
		*dst = &s
	}
	setInt := func(v types.Int64, dst **int) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		i := int(v.ValueInt64())
		*dst = &i
	}
	setInt64 := func(v types.Int64, dst **int64) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		i := v.ValueInt64()
		*dst = &i
	}
	setBool := func(v types.Bool, dst **bool) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		b := v.ValueBool()
		*dst = &b
	}

	setStr(plan.Description, &req.Description)
	setInt(plan.Vcpus, &req.Vcpus)
	setInt(plan.Cores, &req.Cores)
	setInt(plan.Threads, &req.Threads)
	setInt64(plan.MinMemory, &req.MinMemory)
	setStr(plan.Bootloader, &req.Bootloader)
	setStr(plan.BootloaderOvmf, &req.BootloaderOvmf)
	setBool(plan.Autostart, &req.Autostart)
	setBool(plan.HideFromMsr, &req.HideFromMsr)
	setBool(plan.EnsureDisplayDevice, &req.EnsureDisplayDevice)
	setStr(plan.Time, &req.Time)
	setInt(plan.ShutdownTimeout, &req.ShutdownTimeout)
	setStr(plan.ArchType, &req.ArchType)
	setStr(plan.MachineType, &req.MachineType)
	setStr(plan.UUID, &req.UUID)
	setStr(plan.CommandLineArgs, &req.CommandLineArgs)
	setStr(plan.CPUMode, &req.CPUMode)
	setStr(plan.CPUModel, &req.CPUModel)
	setStr(plan.Cpuset, &req.Cpuset)
	setStr(plan.Nodeset, &req.Nodeset)
	setBool(plan.PinVcpus, &req.PinVcpus)
	setBool(plan.SuspendOnSnapshot, &req.SuspendOnSnapshot)
	setBool(plan.TrustedPlatformModule, &req.TrustedPlatformModule)
	setBool(plan.HypervEnlightenments, &req.HypervEnlightenments)
	setBool(plan.EnableSecureBoot, &req.EnableSecureBoot)

	return req
}

func buildVMUpdateRequest(plan *VMResourceModel) *client.VMUpdateRequest {
	req := &client.VMUpdateRequest{}

	name := plan.Name.ValueString()
	req.Name = &name
	mem := plan.Memory.ValueInt64()
	req.Memory = &mem

	setStr := func(v types.String, dst **string) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		s := v.ValueString()
		*dst = &s
	}
	setInt := func(v types.Int64, dst **int) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		i := int(v.ValueInt64())
		*dst = &i
	}
	setInt64 := func(v types.Int64, dst **int64) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		i := v.ValueInt64()
		*dst = &i
	}
	setBool := func(v types.Bool, dst **bool) {
		if v.IsNull() || v.IsUnknown() {
			return
		}
		b := v.ValueBool()
		*dst = &b
	}

	setStr(plan.Description, &req.Description)
	setInt(plan.Vcpus, &req.Vcpus)
	setInt(plan.Cores, &req.Cores)
	setInt(plan.Threads, &req.Threads)
	setInt64(plan.MinMemory, &req.MinMemory)
	setStr(plan.Bootloader, &req.Bootloader)
	// bootloader_ovmf and enable_secure_boot are create-time-only on SCALE
	// 25.10 — the /vm/update endpoint rejects them with HTTP 422
	// "Extra inputs are not permitted". They are exposed as Computed
	// attributes for visibility but cannot be modified post-create.
	setBool(plan.Autostart, &req.Autostart)
	setBool(plan.HideFromMsr, &req.HideFromMsr)
	setBool(plan.EnsureDisplayDevice, &req.EnsureDisplayDevice)
	setStr(plan.Time, &req.Time)
	setInt(plan.ShutdownTimeout, &req.ShutdownTimeout)
	setStr(plan.ArchType, &req.ArchType)
	setStr(plan.MachineType, &req.MachineType)
	setStr(plan.CommandLineArgs, &req.CommandLineArgs)
	setStr(plan.CPUMode, &req.CPUMode)
	setStr(plan.CPUModel, &req.CPUModel)
	setStr(plan.Cpuset, &req.Cpuset)
	setStr(plan.Nodeset, &req.Nodeset)
	setBool(plan.PinVcpus, &req.PinVcpus)
	setBool(plan.SuspendOnSnapshot, &req.SuspendOnSnapshot)
	setBool(plan.TrustedPlatformModule, &req.TrustedPlatformModule)
	setBool(plan.HypervEnlightenments, &req.HypervEnlightenments)

	return req
}

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Trace(ctx, "Create VM start")

	var plan VMResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := buildVMCreateRequest(&plan)

	tflog.Debug(ctx, "Creating VM", map[string]interface{}{"name": createReq.Name})

	vm, err := r.client.CreateVM(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating VM",
			fmt.Sprintf("Could not create VM %q: %s", createReq.Name, err))
		return
	}

	r.mapResponseToModel(vm, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Create VM success")
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Trace(ctx, "Read VM start")

	var state VMResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VM ID: %s", err))
		return
	}

	vm, err := r.client.GetVM(ctx, id)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading VM",
			fmt.Sprintf("Could not read VM %d: %s", id, err))
		return
	}

	r.mapResponseToModel(vm, &state)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Read VM success")
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Trace(ctx, "Update VM start")

	var plan VMResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state VMResourceModel
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VM ID: %s", err))
		return
	}

	updateReq := buildVMUpdateRequest(&plan)

	vm, err := r.client.UpdateVM(ctx, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating VM",
			fmt.Sprintf("Could not update VM %d: %s", id, err))
		return
	}

	r.mapResponseToModel(vm, &plan)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	tflog.Trace(ctx, "Update VM success")
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Trace(ctx, "Delete VM start")

	var state VMResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("Could not parse VM ID: %s", err))
		return
	}

	tflog.Debug(ctx, "Deleting VM", map[string]interface{}{"id": id})

	// Force stop a running VM but leave any attached zvols in place; those have
	// their own lifecycle managed by truenas_zvol.
	err = r.client.DeleteVM(ctx, id, &client.VMDeleteOptions{Force: true, Zvols: false})
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Warn(ctx, "VM already deleted, removing from state", map[string]interface{}{"id": id})
			return
		}
		resp.Diagnostics.AddError("Error Deleting VM",
			fmt.Sprintf("Could not delete VM %d: %s", id, err))
		return
	}
	tflog.Trace(ctx, "Delete VM success")
}

// ModifyPlan enforces VM cross-attribute constraints at plan time:
//
//   - `cpu_mode = CUSTOM` requires `cpu_model` to be set.
//   - `min_memory` must be strictly less than `memory` (it's a ballooning
//     floor — equal or greater defeats the purpose and TrueNAS rejects it).
//   - `pin_vcpus = true` requires `cpuset` to be set (you can't pin to
//     nothing).
func (r *VMResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Destroy warning runs BEFORE the early-return on null plan so
	// the operator sees the destructive intent at plan time.
	planhelpers.WarnOnDestroy(ctx, req, resp, "truenas_vm")
	if req.Plan.Raw.IsNull() {
		return
	}

	var config VMResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	cpuModeCustom := !config.CPUMode.IsNull() && !config.CPUMode.IsUnknown() && config.CPUMode.ValueString() == "CUSTOM"
	cpuModelSet := !config.CPUModel.IsNull() && !config.CPUModel.IsUnknown() && config.CPUModel.ValueString() != ""
	if cpuModeCustom && !cpuModelSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("cpu_model"),
			"Missing cpu_model",
			"When cpu_mode is CUSTOM, cpu_model must be set to a specific host CPU model.",
		)
	}

	memSet := !config.Memory.IsNull() && !config.Memory.IsUnknown()
	minMemSet := !config.MinMemory.IsNull() && !config.MinMemory.IsUnknown()
	if memSet && minMemSet && config.MinMemory.ValueInt64() >= config.Memory.ValueInt64() {
		resp.Diagnostics.AddAttributeError(
			path.Root("min_memory"),
			"min_memory must be less than memory",
			fmt.Sprintf("min_memory (%d) must be strictly less than memory (%d). "+
				"min_memory is the ballooning floor and must leave room for growth.",
				config.MinMemory.ValueInt64(), config.Memory.ValueInt64()),
		)
	}

	pinSet := !config.PinVcpus.IsNull() && !config.PinVcpus.IsUnknown() && config.PinVcpus.ValueBool()
	cpusetSet := !config.Cpuset.IsNull() && !config.Cpuset.IsUnknown() && config.Cpuset.ValueString() != ""
	if pinSet && !cpusetSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("cpuset"),
			"Missing cpuset",
			"pin_vcpus=true requires cpuset to be set to a valid host CPU mask "+
				"(e.g., \"0-3\"). Cannot pin vCPUs to an empty CPU set.",
		)
	}
}

func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.Atoi(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid ID", fmt.Sprintf("VM ID must be numeric: %s", err))
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *VMResource) mapResponseToModel(vm *client.VM, model *VMResourceModel) {
	model.ID = types.StringValue(strconv.Itoa(vm.ID))
	model.Name = types.StringValue(vm.Name)
	model.Description = types.StringValue(vm.Description)
	model.Vcpus = types.Int64Value(int64(vm.Vcpus))
	model.Cores = types.Int64Value(int64(vm.Cores))
	model.Threads = types.Int64Value(int64(vm.Threads))
	model.Memory = types.Int64Value(vm.Memory)
	if vm.MinMemory != nil {
		model.MinMemory = types.Int64Value(*vm.MinMemory)
	} else {
		model.MinMemory = types.Int64Null()
	}
	model.Bootloader = types.StringValue(vm.Bootloader)
	model.BootloaderOvmf = types.StringValue(vm.BootloaderOvmf)
	model.Autostart = types.BoolValue(vm.Autostart)
	model.HideFromMsr = types.BoolValue(vm.HideFromMsr)
	model.EnsureDisplayDevice = types.BoolValue(vm.EnsureDisplayDevice)
	model.Time = types.StringValue(vm.Time)
	model.ShutdownTimeout = types.Int64Value(int64(vm.ShutdownTimeout))
	model.ArchType = nullableString(vm.ArchType)
	model.MachineType = nullableString(vm.MachineType)
	model.UUID = nullableString(vm.UUID)
	model.CommandLineArgs = types.StringValue(vm.CommandLineArgs)
	model.CPUMode = types.StringValue(vm.CPUMode)
	model.CPUModel = nullableString(vm.CPUModel)
	model.Cpuset = nullableString(vm.Cpuset)
	model.Nodeset = nullableString(vm.Nodeset)
	model.PinVcpus = types.BoolValue(vm.PinVcpus)
	model.SuspendOnSnapshot = types.BoolValue(vm.SuspendOnSnapshot)
	model.TrustedPlatformModule = types.BoolValue(vm.TrustedPlatformModule)
	model.HypervEnlightenments = types.BoolValue(vm.HypervEnlightenments)
	model.EnableSecureBoot = types.BoolValue(vm.EnableSecureBoot)

	if vm.Status != nil {
		model.Status = types.StringValue(vm.Status.State)
	} else {
		model.Status = types.StringValue("")
	}
}

// nullableString converts an API *string into a types.String, mapping nil to Null.
func nullableString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
