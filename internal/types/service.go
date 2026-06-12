package types

// Service represents a TrueNAS system service (smb, nfs, ssh, ...).
type Service struct {
	ID      int    `json:"id"`
	Service string `json:"service"`
	Enable  bool   `json:"enable"`
	State   string `json:"state"`
	Pids    []int  `json:"pids"`
}

// ServiceUpdateRequest is the body for PUT /service/id/{id} /
// service.update.
type ServiceUpdateRequest struct {
	Enable bool `json:"enable"`
}

// ServiceStartStopRequest is the body for POST /service/{start,stop} /
// service.{start, stop}.
type ServiceStartStopRequest struct {
	Service string `json:"service"`
}
