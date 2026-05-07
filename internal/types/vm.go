package types

// VM represents a TrueNAS SCALE virtual machine.
type VM struct {
	ID                         int        `json:"id"`
	Name                       string     `json:"name"`
	Description                string     `json:"description"`
	Vcpus                      int        `json:"vcpus"`
	Cores                      int        `json:"cores"`
	Threads                    int        `json:"threads"`
	Memory                     int64      `json:"memory"`
	MinMemory                  *int64     `json:"min_memory"`
	Bootloader                 string     `json:"bootloader"`
	BootloaderOvmf             string     `json:"bootloader_ovmf"`
	Autostart                  bool       `json:"autostart"`
	HideFromMsr                bool       `json:"hide_from_msr"`
	EnsureDisplayDevice        bool       `json:"ensure_display_device"`
	Time                       string     `json:"time"`
	ShutdownTimeout            int        `json:"shutdown_timeout"`
	ArchType                   *string    `json:"arch_type"`
	MachineType                *string    `json:"machine_type"`
	UUID                       *string    `json:"uuid"`
	CommandLineArgs            string     `json:"command_line_args"`
	CPUMode                    string     `json:"cpu_mode"`
	CPUModel                   *string    `json:"cpu_model"`
	Cpuset                     *string    `json:"cpuset"`
	Nodeset                    *string    `json:"nodeset"`
	EnableCPUTopologyExtension bool       `json:"enable_cpu_topology_extension"`
	PinVcpus                   bool       `json:"pin_vcpus"`
	SuspendOnSnapshot          bool       `json:"suspend_on_snapshot"`
	TrustedPlatformModule      bool       `json:"trusted_platform_module"`
	HypervEnlightenments       bool       `json:"hyperv_enlightenments"`
	EnableSecureBoot           bool       `json:"enable_secure_boot"`
	Status                     *VMStatus  `json:"status"`
	Devices                    []VMDevice `json:"devices"`
	DisplayAvailable           bool       `json:"display_available"`
}

// VMStatus represents the runtime status of a VM.
type VMStatus struct {
	State       string `json:"state"`
	PID         *int   `json:"pid"`
	DomainState string `json:"domain_state"`
}

// VMCreateRequest represents the request body for creating a VM.
// Fields are pointers where the TrueNAS API treats absence differently from zero value.
type VMCreateRequest struct {
	Name                       string  `json:"name"`
	Description                *string `json:"description,omitempty"`
	Vcpus                      *int    `json:"vcpus,omitempty"`
	Cores                      *int    `json:"cores,omitempty"`
	Threads                    *int    `json:"threads,omitempty"`
	Memory                     int64   `json:"memory"`
	MinMemory                  *int64  `json:"min_memory,omitempty"`
	Bootloader                 *string `json:"bootloader,omitempty"`
	BootloaderOvmf             *string `json:"bootloader_ovmf,omitempty"`
	Autostart                  *bool   `json:"autostart,omitempty"`
	HideFromMsr                *bool   `json:"hide_from_msr,omitempty"`
	EnsureDisplayDevice        *bool   `json:"ensure_display_device,omitempty"`
	Time                       *string `json:"time,omitempty"`
	ShutdownTimeout            *int    `json:"shutdown_timeout,omitempty"`
	ArchType                   *string `json:"arch_type,omitempty"`
	MachineType                *string `json:"machine_type,omitempty"`
	UUID                       *string `json:"uuid,omitempty"`
	CommandLineArgs            *string `json:"command_line_args,omitempty"`
	CPUMode                    *string `json:"cpu_mode,omitempty"`
	CPUModel                   *string `json:"cpu_model,omitempty"`
	Cpuset                     *string `json:"cpuset,omitempty"`
	Nodeset                    *string `json:"nodeset,omitempty"`
	EnableCPUTopologyExtension *bool   `json:"enable_cpu_topology_extension,omitempty"`
	PinVcpus                   *bool   `json:"pin_vcpus,omitempty"`
	SuspendOnSnapshot          *bool   `json:"suspend_on_snapshot,omitempty"`
	TrustedPlatformModule      *bool   `json:"trusted_platform_module,omitempty"`
	HypervEnlightenments       *bool   `json:"hyperv_enlightenments,omitempty"`
	EnableSecureBoot           *bool   `json:"enable_secure_boot,omitempty"`
}

// VMUpdateRequest is identical in shape to VMCreateRequest but memory is optional.
type VMUpdateRequest struct {
	Name                       *string `json:"name,omitempty"`
	Description                *string `json:"description,omitempty"`
	Vcpus                      *int    `json:"vcpus,omitempty"`
	Cores                      *int    `json:"cores,omitempty"`
	Threads                    *int    `json:"threads,omitempty"`
	Memory                     *int64  `json:"memory,omitempty"`
	MinMemory                  *int64  `json:"min_memory,omitempty"`
	Bootloader                 *string `json:"bootloader,omitempty"`
	BootloaderOvmf             *string `json:"bootloader_ovmf,omitempty"`
	Autostart                  *bool   `json:"autostart,omitempty"`
	HideFromMsr                *bool   `json:"hide_from_msr,omitempty"`
	EnsureDisplayDevice        *bool   `json:"ensure_display_device,omitempty"`
	Time                       *string `json:"time,omitempty"`
	ShutdownTimeout            *int    `json:"shutdown_timeout,omitempty"`
	ArchType                   *string `json:"arch_type,omitempty"`
	MachineType                *string `json:"machine_type,omitempty"`
	CommandLineArgs            *string `json:"command_line_args,omitempty"`
	CPUMode                    *string `json:"cpu_mode,omitempty"`
	CPUModel                   *string `json:"cpu_model,omitempty"`
	Cpuset                     *string `json:"cpuset,omitempty"`
	Nodeset                    *string `json:"nodeset,omitempty"`
	EnableCPUTopologyExtension *bool   `json:"enable_cpu_topology_extension,omitempty"`
	PinVcpus                   *bool   `json:"pin_vcpus,omitempty"`
	SuspendOnSnapshot          *bool   `json:"suspend_on_snapshot,omitempty"`
	TrustedPlatformModule      *bool   `json:"trusted_platform_module,omitempty"`
	HypervEnlightenments       *bool   `json:"hyperv_enlightenments,omitempty"`
	EnableSecureBoot           *bool   `json:"enable_secure_boot,omitempty"`
}

// VMDeleteOptions represents the options accepted by the VM delete endpoint.
type VMDeleteOptions struct {
	Zvols bool `json:"zvols"`
	Force bool `json:"force"`
}

// VMDevice represents a device attached to a VM (DISK, NIC, CDROM, DISPLAY, RAW, PCI, USB).
type VMDevice struct {
	ID         int                    `json:"id"`
	VM         int                    `json:"vm"`
	Order      *int                   `json:"order"`
	Attributes map[string]interface{} `json:"attributes"`
}

// VMDeviceCreateRequest represents the request body for creating a VM device.
type VMDeviceCreateRequest struct {
	VM         int                    `json:"vm"`
	Order      *int                   `json:"order,omitempty"`
	Attributes map[string]interface{} `json:"attributes"`
}

// VMDeviceUpdateRequest represents the request body for updating a VM device.
type VMDeviceUpdateRequest struct {
	VM         *int                   `json:"vm,omitempty"`
	Order      *int                   `json:"order,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}
