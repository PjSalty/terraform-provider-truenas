package types

import (
	"encoding/json"
	"fmt"
)

// =============================================================================
// NVMe-oF Global (singleton)
// =============================================================================

// NVMetGlobal represents the NVMe-oF global configuration.
type NVMetGlobal struct {
	ID            int    `json:"id"`
	Basenqn       string `json:"basenqn"`
	Kernel        bool   `json:"kernel"`
	Ana           bool   `json:"ana"`
	Rdma          bool   `json:"rdma"`
	XportReferral bool   `json:"xport_referral"`
}

// NVMetGlobalUpdateRequest represents the request to update the global config.
type NVMetGlobalUpdateRequest struct {
	Basenqn       *string `json:"basenqn,omitempty"`
	Kernel        *bool   `json:"kernel,omitempty"`
	Ana           *bool   `json:"ana,omitempty"`
	Rdma          *bool   `json:"rdma,omitempty"`
	XportReferral *bool   `json:"xport_referral,omitempty"`
}

// =============================================================================
// NVMe-oF Host
// =============================================================================

// NVMetHost represents an NVMe-oF host (initiator NQN).
type NVMetHost struct {
	ID            int     `json:"id"`
	Hostnqn       string  `json:"hostnqn"`
	DhchapKey     *string `json:"dhchap_key,omitempty"`
	DhchapCtrlKey *string `json:"dhchap_ctrl_key,omitempty"`
	DhchapDhgroup *string `json:"dhchap_dhgroup,omitempty"`
	DhchapHash    *string `json:"dhchap_hash,omitempty"`
}

// NVMetHostCreateRequest represents the request to create an NVMe-oF host.
type NVMetHostCreateRequest struct {
	Hostnqn       string  `json:"hostnqn"`
	DhchapKey     *string `json:"dhchap_key,omitempty"`
	DhchapCtrlKey *string `json:"dhchap_ctrl_key,omitempty"`
	DhchapDhgroup *string `json:"dhchap_dhgroup,omitempty"`
	DhchapHash    *string `json:"dhchap_hash,omitempty"`
}

// NVMetHostUpdateRequest represents the request to update an NVMe-oF host.
type NVMetHostUpdateRequest struct {
	Hostnqn       *string `json:"hostnqn,omitempty"`
	DhchapKey     *string `json:"dhchap_key,omitempty"`
	DhchapCtrlKey *string `json:"dhchap_ctrl_key,omitempty"`
	DhchapDhgroup *string `json:"dhchap_dhgroup,omitempty"`
	DhchapHash    *string `json:"dhchap_hash,omitempty"`
}

// =============================================================================
// NVMe-oF Subsystem
// =============================================================================

// NVMetSubsys represents an NVMe-oF subsystem (target).
type NVMetSubsys struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Subnqn       *string `json:"subnqn,omitempty"`
	Serial       string  `json:"serial,omitempty"`
	AllowAnyHost bool    `json:"allow_any_host"`
	PiEnable     *bool   `json:"pi_enable,omitempty"`
	QidMax       *int    `json:"qid_max,omitempty"`
	IeeeOui      *string `json:"ieee_oui,omitempty"`
	Ana          *bool   `json:"ana,omitempty"`
}

// NVMetSubsysCreateRequest represents the request to create an NVMe-oF subsystem.
type NVMetSubsysCreateRequest struct {
	Name         string  `json:"name"`
	Subnqn       *string `json:"subnqn,omitempty"`
	AllowAnyHost *bool   `json:"allow_any_host,omitempty"`
	PiEnable     *bool   `json:"pi_enable,omitempty"`
	QidMax       *int    `json:"qid_max,omitempty"`
	IeeeOui      *string `json:"ieee_oui,omitempty"`
	Ana          *bool   `json:"ana,omitempty"`
}

// NVMetSubsysUpdateRequest represents the request to update an NVMe-oF subsystem.
type NVMetSubsysUpdateRequest struct {
	Name         *string `json:"name,omitempty"`
	Subnqn       *string `json:"subnqn,omitempty"`
	AllowAnyHost *bool   `json:"allow_any_host,omitempty"`
	PiEnable     *bool   `json:"pi_enable,omitempty"`
	QidMax       *int    `json:"qid_max,omitempty"`
	IeeeOui      *string `json:"ieee_oui,omitempty"`
	Ana          *bool   `json:"ana,omitempty"`
}

// =============================================================================
// NVMe-oF Port
// =============================================================================

// NVMetPort represents an NVMe-oF transport port.
// addr_trsvcid may be returned as int or string depending on transport type; we
// normalize by reading as RawMessage and exposing an int via GetAddrTrsvcid.
type NVMetPort struct {
	ID             int             `json:"id"`
	Index          int             `json:"index"`
	AddrTrtype     string          `json:"addr_trtype"`
	AddrTrsvcid    json.RawMessage `json:"addr_trsvcid,omitempty"`
	AddrTraddr     string          `json:"addr_traddr"`
	InlineDataSize *int            `json:"inline_data_size,omitempty"`
	MaxQueueSize   *int            `json:"max_queue_size,omitempty"`
	PiEnable       *bool           `json:"pi_enable,omitempty"`
	Enabled        bool            `json:"enabled"`
}

// GetAddrTrsvcid returns the addr_trsvcid as an int, accepting int or string.
func (p *NVMetPort) GetAddrTrsvcid() int {
	if len(p.AddrTrsvcid) == 0 || string(p.AddrTrsvcid) == "null" {
		return 0
	}
	var n int
	if err := json.Unmarshal(p.AddrTrsvcid, &n); err == nil {
		return n
	}
	var s string
	if err := json.Unmarshal(p.AddrTrsvcid, &s); err == nil {
		var parsed int
		if _, err := fmt.Sscanf(s, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return 0
}

// NVMetPortCreateRequest represents the request to create an NVMe-oF port.
type NVMetPortCreateRequest struct {
	AddrTrtype     string `json:"addr_trtype"`
	AddrTraddr     string `json:"addr_traddr"`
	AddrTrsvcid    *int   `json:"addr_trsvcid,omitempty"`
	InlineDataSize *int   `json:"inline_data_size,omitempty"`
	MaxQueueSize   *int   `json:"max_queue_size,omitempty"`
	PiEnable       *bool  `json:"pi_enable,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

// NVMetPortUpdateRequest represents the request to update an NVMe-oF port.
type NVMetPortUpdateRequest struct {
	AddrTrtype     *string `json:"addr_trtype,omitempty"`
	AddrTraddr     *string `json:"addr_traddr,omitempty"`
	AddrTrsvcid    *int    `json:"addr_trsvcid,omitempty"`
	InlineDataSize *int    `json:"inline_data_size,omitempty"`
	MaxQueueSize   *int    `json:"max_queue_size,omitempty"`
	PiEnable       *bool   `json:"pi_enable,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

// =============================================================================
// NVMe-oF Namespace
// =============================================================================

// NVMetNamespaceSubsys is the nested subsystem object returned in namespace GET.
type NVMetNamespaceSubsys struct {
	ID int `json:"id"`
}

// NVMetNamespace represents an NVMe-oF namespace (block device within a subsystem).
// GET returns a nested subsys object; we expose both a raw SubsysID and the
// nested wrapper for round-trip convenience.
type NVMetNamespace struct {
	ID         int                   `json:"id"`
	Nsid       *int                  `json:"nsid,omitempty"`
	Subsys     *NVMetNamespaceSubsys `json:"subsys,omitempty"`
	SubsysID   int                   `json:"subsys_id,omitempty"`
	DeviceType string                `json:"device_type"`
	DevicePath string                `json:"device_path"`
	Filesize   *int64                `json:"filesize,omitempty"`
	Enabled    bool                  `json:"enabled"`
}

// EffectiveSubsysID returns the subsys id whether the API replied with the
// nested object form or the flat form.
func (n *NVMetNamespace) EffectiveSubsysID() int {
	if n.Subsys != nil && n.Subsys.ID != 0 {
		return n.Subsys.ID
	}
	return n.SubsysID
}

// NVMetNamespaceCreateRequest represents the request to create an NVMe-oF namespace.
type NVMetNamespaceCreateRequest struct {
	Nsid       *int   `json:"nsid,omitempty"`
	DeviceType string `json:"device_type"`
	DevicePath string `json:"device_path"`
	Filesize   *int64 `json:"filesize,omitempty"`
	Enabled    *bool  `json:"enabled,omitempty"`
	SubsysID   int    `json:"subsys_id"`
}

// NVMetNamespaceUpdateRequest represents the request to update an NVMe-oF namespace.
type NVMetNamespaceUpdateRequest struct {
	Nsid       *int    `json:"nsid,omitempty"`
	DeviceType *string `json:"device_type,omitempty"`
	DevicePath *string `json:"device_path,omitempty"`
	Filesize   *int64  `json:"filesize,omitempty"`
	Enabled    *bool   `json:"enabled,omitempty"`
	SubsysID   *int    `json:"subsys_id,omitempty"`
}

// =============================================================================
// NVMe-oF Host-Subsys association
// =============================================================================

// NVMetHostSubsysHost is a minimal nested host object returned in GET.
type NVMetHostSubsysHost struct {
	ID int `json:"id"`
}

// NVMetHostSubsysSubsys is a minimal nested subsys object returned in GET.
type NVMetHostSubsysSubsys struct {
	ID int `json:"id"`
}

// NVMetHostSubsys represents a host-to-subsystem authorization.
// GET returns nested host/subsys objects; create takes flat host_id/subsys_id.
type NVMetHostSubsys struct {
	ID       int                    `json:"id"`
	Host     *NVMetHostSubsysHost   `json:"host,omitempty"`
	Subsys   *NVMetHostSubsysSubsys `json:"subsys,omitempty"`
	HostID   int                    `json:"host_id,omitempty"`
	SubsysID int                    `json:"subsys_id,omitempty"`
}

// EffectiveHostID returns host id whether API returned nested or flat form.
func (hs *NVMetHostSubsys) EffectiveHostID() int {
	if hs.Host != nil && hs.Host.ID != 0 {
		return hs.Host.ID
	}
	return hs.HostID
}

// EffectiveSubsysID returns subsys id whether API returned nested or flat form.
func (hs *NVMetHostSubsys) EffectiveSubsysID() int {
	if hs.Subsys != nil && hs.Subsys.ID != 0 {
		return hs.Subsys.ID
	}
	return hs.SubsysID
}

// NVMetHostSubsysCreateRequest represents the request to create an association.
type NVMetHostSubsysCreateRequest struct {
	HostID   int `json:"host_id"`
	SubsysID int `json:"subsys_id"`
}

// =============================================================================
// NVMe-oF Port-Subsys association
// =============================================================================

// NVMetPortSubsysPort is a minimal nested port object.
type NVMetPortSubsysPort struct {
	ID int `json:"id"`
}

// NVMetPortSubsysSubsys is a minimal nested subsys object.
type NVMetPortSubsysSubsys struct {
	ID int `json:"id"`
}

// NVMetPortSubsys represents a port-to-subsystem association.
type NVMetPortSubsys struct {
	ID       int                    `json:"id"`
	Port     *NVMetPortSubsysPort   `json:"port,omitempty"`
	Subsys   *NVMetPortSubsysSubsys `json:"subsys,omitempty"`
	PortID   int                    `json:"port_id,omitempty"`
	SubsysID int                    `json:"subsys_id,omitempty"`
}

// EffectivePortID returns port id whether API returned nested or flat form.
func (ps *NVMetPortSubsys) EffectivePortID() int {
	if ps.Port != nil && ps.Port.ID != 0 {
		return ps.Port.ID
	}
	return ps.PortID
}

// EffectiveSubsysID returns subsys id whether API returned nested or flat form.
func (ps *NVMetPortSubsys) EffectiveSubsysID() int {
	if ps.Subsys != nil && ps.Subsys.ID != 0 {
		return ps.Subsys.ID
	}
	return ps.SubsysID
}

// NVMetPortSubsysCreateRequest represents the request to create an association.
type NVMetPortSubsysCreateRequest struct {
	PortID   int `json:"port_id"`
	SubsysID int `json:"subsys_id"`
}
