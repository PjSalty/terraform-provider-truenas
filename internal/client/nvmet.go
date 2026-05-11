package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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

// GetNVMetGlobal retrieves the NVMe-oF global configuration.
func (c *Client) GetNVMetGlobal(ctx context.Context) (*NVMetGlobal, error) {
	tflog.Trace(ctx, "GetNVMetGlobal start")

	resp, err := c.Get(ctx, "/nvmet/global")
	if err != nil {
		return nil, fmt.Errorf("getting nvmet global: %w", err)
	}
	var g NVMetGlobal
	if err := json.Unmarshal(resp, &g); err != nil {
		return nil, fmt.Errorf("parsing nvmet global response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetGlobal success")
	return &g, nil
}

// UpdateNVMetGlobal updates the NVMe-oF global configuration.
func (c *Client) UpdateNVMetGlobal(ctx context.Context, req *NVMetGlobalUpdateRequest) (*NVMetGlobal, error) {
	tflog.Trace(ctx, "UpdateNVMetGlobal start")

	resp, err := c.Put(ctx, "/nvmet/global", req)
	if err != nil {
		return nil, fmt.Errorf("updating nvmet global: %w", err)
	}
	var g NVMetGlobal
	if err := json.Unmarshal(resp, &g); err != nil {
		return nil, fmt.Errorf("parsing nvmet global update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetGlobal success")
	return &g, nil
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

// GetNVMetHost retrieves an NVMe-oF host by ID.
func (c *Client) GetNVMetHost(ctx context.Context, id int) (*NVMetHost, error) {
	tflog.Trace(ctx, "GetNVMetHost start")

	resp, err := c.Get(ctx, fmt.Sprintf("/nvmet/host/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting nvmet host %d: %w", id, err)
	}
	var h NVMetHost
	if err := json.Unmarshal(resp, &h); err != nil {
		return nil, fmt.Errorf("parsing nvmet host response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetHost success")
	return &h, nil
}

// CreateNVMetHost creates a new NVMe-oF host.
func (c *Client) CreateNVMetHost(ctx context.Context, req *NVMetHostCreateRequest) (*NVMetHost, error) {
	tflog.Trace(ctx, "CreateNVMetHost start")

	resp, err := c.Post(ctx, "/nvmet/host", req)
	if err != nil {
		return nil, fmt.Errorf("creating nvmet host: %w", err)
	}
	var h NVMetHost
	if err := json.Unmarshal(resp, &h); err != nil {
		return nil, fmt.Errorf("parsing nvmet host create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetHost success")
	return &h, nil
}

// UpdateNVMetHost updates an NVMe-oF host.
func (c *Client) UpdateNVMetHost(ctx context.Context, id int, req *NVMetHostUpdateRequest) (*NVMetHost, error) {
	tflog.Trace(ctx, "UpdateNVMetHost start")

	resp, err := c.Put(ctx, fmt.Sprintf("/nvmet/host/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating nvmet host %d: %w", id, err)
	}
	var h NVMetHost
	if err := json.Unmarshal(resp, &h); err != nil {
		return nil, fmt.Errorf("parsing nvmet host update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetHost success")
	return &h, nil
}

// DeleteNVMetHost deletes an NVMe-oF host.
func (c *Client) DeleteNVMetHost(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetHost start")

	_, err := c.Delete(ctx, fmt.Sprintf("/nvmet/host/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting nvmet host %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetHost success")
	return nil
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

// GetNVMetSubsys retrieves an NVMe-oF subsystem by ID.
func (c *Client) GetNVMetSubsys(ctx context.Context, id int) (*NVMetSubsys, error) {
	tflog.Trace(ctx, "GetNVMetSubsys start")

	resp, err := c.Get(ctx, fmt.Sprintf("/nvmet/subsys/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting nvmet subsys %d: %w", id, err)
	}
	var s NVMetSubsys
	if err := json.Unmarshal(resp, &s); err != nil {
		return nil, fmt.Errorf("parsing nvmet subsys response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetSubsys success")
	return &s, nil
}

// CreateNVMetSubsys creates a new NVMe-oF subsystem.
func (c *Client) CreateNVMetSubsys(ctx context.Context, req *NVMetSubsysCreateRequest) (*NVMetSubsys, error) {
	tflog.Trace(ctx, "CreateNVMetSubsys start")

	resp, err := c.Post(ctx, "/nvmet/subsys", req)
	if err != nil {
		return nil, fmt.Errorf("creating nvmet subsys: %w", err)
	}
	var s NVMetSubsys
	if err := json.Unmarshal(resp, &s); err != nil {
		return nil, fmt.Errorf("parsing nvmet subsys create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetSubsys success")
	return &s, nil
}

// UpdateNVMetSubsys updates an NVMe-oF subsystem.
func (c *Client) UpdateNVMetSubsys(ctx context.Context, id int, req *NVMetSubsysUpdateRequest) (*NVMetSubsys, error) {
	tflog.Trace(ctx, "UpdateNVMetSubsys start")

	resp, err := c.Put(ctx, fmt.Sprintf("/nvmet/subsys/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating nvmet subsys %d: %w", id, err)
	}
	var s NVMetSubsys
	if err := json.Unmarshal(resp, &s); err != nil {
		return nil, fmt.Errorf("parsing nvmet subsys update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetSubsys success")
	return &s, nil
}

// DeleteNVMetSubsys deletes an NVMe-oF subsystem.
func (c *Client) DeleteNVMetSubsys(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetSubsys start")

	_, err := c.Delete(ctx, fmt.Sprintf("/nvmet/subsys/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting nvmet subsys %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetSubsys success")
	return nil
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

// GetNVMetPort retrieves an NVMe-oF port by ID.
func (c *Client) GetNVMetPort(ctx context.Context, id int) (*NVMetPort, error) {
	tflog.Trace(ctx, "GetNVMetPort start")

	resp, err := c.Get(ctx, fmt.Sprintf("/nvmet/port/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting nvmet port %d: %w", id, err)
	}
	var p NVMetPort
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing nvmet port response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetPort success")
	return &p, nil
}

// CreateNVMetPort creates a new NVMe-oF port.
func (c *Client) CreateNVMetPort(ctx context.Context, req *NVMetPortCreateRequest) (*NVMetPort, error) {
	tflog.Trace(ctx, "CreateNVMetPort start")

	resp, err := c.Post(ctx, "/nvmet/port", req)
	if err != nil {
		return nil, fmt.Errorf("creating nvmet port: %w", err)
	}
	var p NVMetPort
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing nvmet port create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetPort success")
	return &p, nil
}

// UpdateNVMetPort updates an NVMe-oF port.
func (c *Client) UpdateNVMetPort(ctx context.Context, id int, req *NVMetPortUpdateRequest) (*NVMetPort, error) {
	tflog.Trace(ctx, "UpdateNVMetPort start")

	resp, err := c.Put(ctx, fmt.Sprintf("/nvmet/port/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating nvmet port %d: %w", id, err)
	}
	var p NVMetPort
	if err := json.Unmarshal(resp, &p); err != nil {
		return nil, fmt.Errorf("parsing nvmet port update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetPort success")
	return &p, nil
}

// DeleteNVMetPort deletes an NVMe-oF port.
func (c *Client) DeleteNVMetPort(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetPort start")

	_, err := c.Delete(ctx, fmt.Sprintf("/nvmet/port/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting nvmet port %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetPort success")
	return nil
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

// GetNVMetNamespace retrieves an NVMe-oF namespace by ID.
func (c *Client) GetNVMetNamespace(ctx context.Context, id int) (*NVMetNamespace, error) {
	tflog.Trace(ctx, "GetNVMetNamespace start")

	resp, err := c.Get(ctx, fmt.Sprintf("/nvmet/namespace/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting nvmet namespace %d: %w", id, err)
	}
	var n NVMetNamespace
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, fmt.Errorf("parsing nvmet namespace response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetNamespace success")
	return &n, nil
}

// CreateNVMetNamespace creates a new NVMe-oF namespace.
func (c *Client) CreateNVMetNamespace(ctx context.Context, req *NVMetNamespaceCreateRequest) (*NVMetNamespace, error) {
	tflog.Trace(ctx, "CreateNVMetNamespace start")

	resp, err := c.Post(ctx, "/nvmet/namespace", req)
	if err != nil {
		return nil, fmt.Errorf("creating nvmet namespace: %w", err)
	}
	var n NVMetNamespace
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, fmt.Errorf("parsing nvmet namespace create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetNamespace success")
	return &n, nil
}

// UpdateNVMetNamespace updates an NVMe-oF namespace.
func (c *Client) UpdateNVMetNamespace(ctx context.Context, id int, req *NVMetNamespaceUpdateRequest) (*NVMetNamespace, error) {
	tflog.Trace(ctx, "UpdateNVMetNamespace start")

	resp, err := c.Put(ctx, fmt.Sprintf("/nvmet/namespace/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating nvmet namespace %d: %w", id, err)
	}
	var n NVMetNamespace
	if err := json.Unmarshal(resp, &n); err != nil {
		return nil, fmt.Errorf("parsing nvmet namespace update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetNamespace success")
	return &n, nil
}

// DeleteNVMetNamespace deletes an NVMe-oF namespace.
func (c *Client) DeleteNVMetNamespace(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetNamespace start")

	_, err := c.Delete(ctx, fmt.Sprintf("/nvmet/namespace/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting nvmet namespace %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetNamespace success")
	return nil
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

// GetNVMetHostSubsys retrieves a host-subsys association by ID.
func (c *Client) GetNVMetHostSubsys(ctx context.Context, id int) (*NVMetHostSubsys, error) {
	tflog.Trace(ctx, "GetNVMetHostSubsys start")

	resp, err := c.Get(ctx, fmt.Sprintf("/nvmet/host_subsys/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting nvmet host_subsys %d: %w", id, err)
	}
	var hs NVMetHostSubsys
	if err := json.Unmarshal(resp, &hs); err != nil {
		return nil, fmt.Errorf("parsing nvmet host_subsys response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetHostSubsys success")
	return &hs, nil
}

// CreateNVMetHostSubsys creates a host-subsys association.
func (c *Client) CreateNVMetHostSubsys(ctx context.Context, req *NVMetHostSubsysCreateRequest) (*NVMetHostSubsys, error) {
	tflog.Trace(ctx, "CreateNVMetHostSubsys start")

	resp, err := c.Post(ctx, "/nvmet/host_subsys", req)
	if err != nil {
		return nil, fmt.Errorf("creating nvmet host_subsys: %w", err)
	}
	var hs NVMetHostSubsys
	if err := json.Unmarshal(resp, &hs); err != nil {
		return nil, fmt.Errorf("parsing nvmet host_subsys create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetHostSubsys success")
	return &hs, nil
}

// DeleteNVMetHostSubsys deletes a host-subsys association.
func (c *Client) DeleteNVMetHostSubsys(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetHostSubsys start")

	_, err := c.Delete(ctx, fmt.Sprintf("/nvmet/host_subsys/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting nvmet host_subsys %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetHostSubsys success")
	return nil
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

// GetNVMetPortSubsys retrieves a port-subsys association by ID.
func (c *Client) GetNVMetPortSubsys(ctx context.Context, id int) (*NVMetPortSubsys, error) {
	tflog.Trace(ctx, "GetNVMetPortSubsys start")

	resp, err := c.Get(ctx, fmt.Sprintf("/nvmet/port_subsys/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting nvmet port_subsys %d: %w", id, err)
	}
	var ps NVMetPortSubsys
	if err := json.Unmarshal(resp, &ps); err != nil {
		return nil, fmt.Errorf("parsing nvmet port_subsys response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetPortSubsys success")
	return &ps, nil
}

// CreateNVMetPortSubsys creates a port-subsys association.
func (c *Client) CreateNVMetPortSubsys(ctx context.Context, req *NVMetPortSubsysCreateRequest) (*NVMetPortSubsys, error) {
	tflog.Trace(ctx, "CreateNVMetPortSubsys start")

	resp, err := c.Post(ctx, "/nvmet/port_subsys", req)
	if err != nil {
		return nil, fmt.Errorf("creating nvmet port_subsys: %w", err)
	}
	var ps NVMetPortSubsys
	if err := json.Unmarshal(resp, &ps); err != nil {
		return nil, fmt.Errorf("parsing nvmet port_subsys create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetPortSubsys success")
	return &ps, nil
}

// DeleteNVMetPortSubsys deletes a port-subsys association.
func (c *Client) DeleteNVMetPortSubsys(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetPortSubsys start")

	_, err := c.Delete(ctx, fmt.Sprintf("/nvmet/port_subsys/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting nvmet port_subsys %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetPortSubsys success")
	return nil
}
