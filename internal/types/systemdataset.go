package types

// SystemDataset represents the TrueNAS system dataset configuration —
// a singleton describing which pool hosts internal system data
// (samba, reports, syslog, ...).
//
// Returned by REST GET /systemdataset and JSON-RPC systemdataset.config.
type SystemDataset struct {
	ID       int    `json:"id"`
	Pool     string `json:"pool"`
	PoolSet  bool   `json:"pool_set"`
	UUID     string `json:"uuid"`
	Basename string `json:"basename"`
	Path     string `json:"path"`
}

// SystemDatasetUpdateRequest is the payload for PUT /systemdataset and
// JSON-RPC systemdataset.update. Either `pool` (target) or
// `pool_exclude` (avoid this pool) may be supplied.
type SystemDatasetUpdateRequest struct {
	Pool        *string `json:"pool,omitempty"`
	PoolExclude *string `json:"pool_exclude,omitempty"`
}
