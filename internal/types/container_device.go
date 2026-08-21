package types

// Device type discriminators, from the upstream ContainerDeviceType
// discriminated union.
const (
	ContainerDeviceFilesystem = "FILESYSTEM"
	ContainerDeviceGPU        = "GPU"
	ContainerDeviceNIC        = "NIC"
	ContainerDeviceUSB        = "USB"
)

// ContainerDevice is a device attached to a container, new in TrueNAS
// 26.0 alongside the container namespace itself.
//
// Attributes is a discriminated union keyed on dtype. It is kept as a map
// on the wire because the four member shapes share no fields, and the
// upstream update model types it as a bare dict for the same reason: the
// server validates it against the right member once dtype is known.
type ContainerDevice struct {
	ID         int                    `json:"id"`
	Container  int                    `json:"container"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ContainerDeviceCreateRequest is the body for container.device.create.
type ContainerDeviceCreateRequest struct {
	Container  int                    `json:"container"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ContainerDeviceUpdateRequest is the body for container.device.update.
//
// Attributes is sent whole rather than merged: the union member is chosen
// by dtype, so a partial map would be validated against a shape it does
// not fully describe.
type ContainerDeviceUpdateRequest struct {
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// ContainerDeviceDeleteOptions controls how far a device removal reaches.
//
// RawFile and Zvol destroy the storage behind the device. The provider
// never sets either: the backing storage has its own lifecycle and is not
// something removing a device attachment was asked to touch.
type ContainerDeviceDeleteOptions struct {
	Force   bool `json:"force"`
	RawFile bool `json:"raw_file"`
	Zvol    bool `json:"zvol"`
}

// ContainerNICAttachChoices lists the interfaces a NIC device can attach
// to, split by how the attachment is made.
type ContainerNICAttachChoices struct {
	Bridge  []string `json:"BRIDGE"`
	Macvlan []string `json:"MACVLAN"`
}

// ContainerImage is one image available from the LXC image registry, with
// every version currently published for it. Both fields of a
// truenas_container image block come from here.
type ContainerImage struct {
	Name     string                  `json:"name"`
	Versions []ContainerImageVersion `json:"versions"`
}

// ContainerImageVersion is one datestamped publication of an image. The
// registry keeps only the most recent few, so a version pinned today stops
// resolving within days.
type ContainerImageVersion struct {
	Version string `json:"version"`
}
