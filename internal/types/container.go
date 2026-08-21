package types

// Container models a TrueNAS LXC container, new in 26.0. The container
// namespace does not exist on 25.10 or earlier.
//
// Dataset, DefaultNetwork and Status are excluded_field() on
// ContainerCreate upstream: they are derived by middleware and read-only
// here. Devices is excluded too, and has its own namespace
// (container.device.*), so it is not modeled on this struct.
type Container struct {
	ID                 int               `json:"id"`
	UUID               string            `json:"uuid"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Cpuset             *string           `json:"cpuset"`
	Autostart          bool              `json:"autostart"`
	Time               string            `json:"time"`
	ShutdownTimeout    int               `json:"shutdown_timeout"`
	Dataset            string            `json:"dataset"`
	Init               string            `json:"init"`
	InitDir            *string           `json:"initdir"`
	InitEnv            map[string]string `json:"initenv"`
	InitUser           *string           `json:"inituser"`
	InitGroup          *string           `json:"initgroup"`
	Idmap              *ContainerIdmap   `json:"idmap"`
	CapabilitiesPolicy string            `json:"capabilities_policy"`
	CapabilitiesState  map[string]bool   `json:"capabilities_state"`
	DefaultNetwork     *string           `json:"default_network"`
	Status             ContainerStatus   `json:"status"`
}

// ContainerIdmap is the discriminated union upstream models as DEFAULT or
// ISOLATED. Slice is only meaningful for ISOLATED, where a null asks the
// backend to pick an unused one.
//
// A null idmap on the wire is a THIRD state, not a missing value: it means
// the container applies no user-namespace mapping at all, so container
// root is host root. That is why Container.Idmap is a pointer.
type ContainerIdmap struct {
	Type  string `json:"type"`
	Slice *int   `json:"slice,omitempty"`
}

// ContainerStatus is derived, read-only runtime state.
type ContainerStatus struct {
	State       string  `json:"state"`
	PID         *int    `json:"pid"`
	DomainState *string `json:"domain_state"`
}

// ContainerImageRef selects the image a container is created from.
// Both fields come from container.image.query_registry.
type ContainerImageRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ContainerCreateRequest is the body for container.create.
//
// Pool, Image and Idmap have no counterpart on the update model upstream,
// so they can only be set at creation; the resource marks them
// RequiresReplace.
type ContainerCreateRequest struct {
	Name               string             `json:"name"`
	Image              ContainerImageRef  `json:"image"`
	UUID               *string            `json:"uuid,omitempty"`
	Pool               *string            `json:"pool,omitempty"`
	Description        *string            `json:"description,omitempty"`
	Cpuset             *string            `json:"cpuset,omitempty"`
	Autostart          *bool              `json:"autostart,omitempty"`
	Time               *string            `json:"time,omitempty"`
	ShutdownTimeout    *int               `json:"shutdown_timeout,omitempty"`
	Init               *string            `json:"init,omitempty"`
	InitDir            *string            `json:"initdir,omitempty"`
	InitEnv            *map[string]string `json:"initenv,omitempty"`
	InitUser           *string            `json:"inituser,omitempty"`
	InitGroup          *string            `json:"initgroup,omitempty"`
	Idmap              *ContainerIdmap    `json:"idmap,omitempty"`
	CapabilitiesPolicy *string            `json:"capabilities_policy,omitempty"`
	CapabilitiesState  *map[string]bool   `json:"capabilities_state,omitempty"`
}

// ContainerUpdateRequest is the body for container.update. Every field is
// optional: the upstream model is ForUpdateMetaclass, so an omitted key
// leaves the stored value alone.
//
// pool, image and idmap are absent by design, they are excluded_field()
// on the upstream update model and sending them is rejected.
type ContainerUpdateRequest struct {
	Name               *string            `json:"name,omitempty"`
	Description        *string            `json:"description,omitempty"`
	Cpuset             *string            `json:"cpuset,omitempty"`
	Autostart          *bool              `json:"autostart,omitempty"`
	Time               *string            `json:"time,omitempty"`
	ShutdownTimeout    *int               `json:"shutdown_timeout,omitempty"`
	Init               *string            `json:"init,omitempty"`
	InitDir            *string            `json:"initdir,omitempty"`
	InitEnv            *map[string]string `json:"initenv,omitempty"`
	InitUser           *string            `json:"inituser,omitempty"`
	InitGroup          *string            `json:"initgroup,omitempty"`
	CapabilitiesPolicy *string            `json:"capabilities_policy,omitempty"`
	CapabilitiesState  *map[string]bool   `json:"capabilities_state,omitempty"`
}

// ContainerDeleteOptions controls how far a delete reaches.
//
// Recursive destroys the container's child datasets and snapshots, any
// clones of those snapshots anywhere in the pool, and any holds on them,
// none of it recoverable. The provider never sets it.
type ContainerDeleteOptions struct {
	Force     bool `json:"force"`
	Recursive bool `json:"recursive"`
}

// ContainerStopOptions controls how a stop is escalated.
type ContainerStopOptions struct {
	Force             bool `json:"force"`
	ForceAfterTimeout bool `json:"force_after_timeout"`
}
