package types

// VMware represents a VMware host registration in TrueNAS.
type VMware struct {
	ID         int    `json:"id"`
	Datastore  string `json:"datastore"`
	Filesystem string `json:"filesystem"`
	Hostname   string `json:"hostname"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// VMwareCreateRequest represents a request to create a VMware integration.
type VMwareCreateRequest struct {
	Datastore  string `json:"datastore"`
	Filesystem string `json:"filesystem"`
	Hostname   string `json:"hostname"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

// VMwareUpdateRequest represents a request to update a VMware integration.
type VMwareUpdateRequest struct {
	Datastore  *string `json:"datastore,omitempty"`
	Filesystem *string `json:"filesystem,omitempty"`
	Hostname   *string `json:"hostname,omitempty"`
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
}
