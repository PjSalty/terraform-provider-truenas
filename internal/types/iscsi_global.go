package types

// ISCSIGlobal is the iSCSI global configuration: the target name prefix,
// the iSNS servers and the listen port.
//
// It is a SystemServiceService singleton: one row, read with
// iscsi.global.config and written with iscsi.global.update.
//
// PoolAvailThreshold is null when the check is off. Alua and Iser are
// license and hardware gated, so they are read but never written.
type ISCSIGlobal struct {
	ID                 int      `json:"id"`
	Basename           string   `json:"basename"`
	IsnsServers        []string `json:"isns_servers"`
	ListenPort         int      `json:"listen_port"`
	PoolAvailThreshold *int     `json:"pool_avail_threshold"`
	Alua               bool     `json:"alua"`
	Iser               bool     `json:"iser"`
}

// ISCSIGlobalUpdateRequest is the body for iscsi.global.update. The upstream
// model is ForUpdateMetaclass, so an omitted key leaves the stored value
// alone, with one exception: iscsi.global.update rebuilds the iSNS list from
// the request alone, so an update that leaves isns_servers out empties it.
// IsnsServers therefore has no omitempty and is always sent, and a nil slice
// is made an empty list by the caller.
type ISCSIGlobalUpdateRequest struct {
	Basename           *string       `json:"basename,omitempty"`
	IsnsServers        []string      `json:"isns_servers"`
	ListenPort         *int          `json:"listen_port,omitempty"`
	PoolAvailThreshold Nullable[int] `json:"pool_avail_threshold,omitzero"`
}
