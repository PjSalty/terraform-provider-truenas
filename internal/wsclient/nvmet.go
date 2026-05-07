package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespaces for NVMe-oF:
//   nvmet.global.{config,update}                  (singleton)
//   nvmet.host.{get_instance,create,update,delete}
//   nvmet.subsys.{get_instance,create,update,delete}
//   nvmet.port.{get_instance,create,update,delete}
//   nvmet.namespace.{get_instance,create,update,delete}
//   nvmet.host_subsys.{get_instance,create,delete}    (no update)
//   nvmet.port_subsys.{get_instance,create,delete}    (no update)

// =============================================================================
// NVMe-oF Global (singleton)
// =============================================================================

// GetNVMetGlobal retrieves the NVMe-oF global configuration.
func (c *Client) GetNVMetGlobal(ctx context.Context) (*types.NVMetGlobal, error) {
	tflog.Trace(ctx, "GetNVMetGlobal (ws) start")

	result, err := c.Call(ctx, "nvmet.global.config", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet global: %w", err)
	}
	var g types.NVMetGlobal
	if err := json.Unmarshal(result, &g); err != nil {
		return nil, fmt.Errorf("parsing nvmet global response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetGlobal (ws) success")
	return &g, nil
}

// UpdateNVMetGlobal updates the NVMe-oF global configuration.
func (c *Client) UpdateNVMetGlobal(ctx context.Context, req *types.NVMetGlobalUpdateRequest) (*types.NVMetGlobal, error) {
	tflog.Trace(ctx, "UpdateNVMetGlobal (ws) start")

	result, err := c.Call(ctx, "nvmet.global.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating nvmet global: %w", err)
	}
	var g types.NVMetGlobal
	if err := json.Unmarshal(result, &g); err != nil {
		return nil, fmt.Errorf("parsing nvmet global update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetGlobal (ws) success")
	return &g, nil
}

// =============================================================================
// NVMe-oF Host
// =============================================================================

// GetNVMetHost retrieves an NVMe-oF host by ID.
func (c *Client) GetNVMetHost(ctx context.Context, id int) (*types.NVMetHost, error) {
	tflog.Trace(ctx, "GetNVMetHost (ws) start")

	result, err := c.Call(ctx, "nvmet.host.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet host %d: %w", id, err)
	}
	var h types.NVMetHost
	if err := json.Unmarshal(result, &h); err != nil {
		return nil, fmt.Errorf("parsing nvmet host response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetHost (ws) success")
	return &h, nil
}

// CreateNVMetHost creates a new NVMe-oF host.
func (c *Client) CreateNVMetHost(ctx context.Context, req *types.NVMetHostCreateRequest) (*types.NVMetHost, error) {
	tflog.Trace(ctx, "CreateNVMetHost (ws) start")

	result, err := c.Call(ctx, "nvmet.host.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating nvmet host: %w", err)
	}
	var h types.NVMetHost
	if err := json.Unmarshal(result, &h); err != nil {
		return nil, fmt.Errorf("parsing nvmet host create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetHost (ws) success")
	return &h, nil
}

// UpdateNVMetHost updates an NVMe-oF host.
func (c *Client) UpdateNVMetHost(ctx context.Context, id int, req *types.NVMetHostUpdateRequest) (*types.NVMetHost, error) {
	tflog.Trace(ctx, "UpdateNVMetHost (ws) start")

	result, err := c.Call(ctx, "nvmet.host.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating nvmet host %d: %w", id, err)
	}
	var h types.NVMetHost
	if err := json.Unmarshal(result, &h); err != nil {
		return nil, fmt.Errorf("parsing nvmet host update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetHost (ws) success")
	return &h, nil
}

// DeleteNVMetHost deletes an NVMe-oF host.
func (c *Client) DeleteNVMetHost(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetHost (ws) start")

	if _, err := c.Call(ctx, "nvmet.host.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting nvmet host %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetHost (ws) success")
	return nil
}

// =============================================================================
// NVMe-oF Subsystem
// =============================================================================

// GetNVMetSubsys retrieves an NVMe-oF subsystem by ID.
func (c *Client) GetNVMetSubsys(ctx context.Context, id int) (*types.NVMetSubsys, error) {
	tflog.Trace(ctx, "GetNVMetSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.subsys.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet subsys %d: %w", id, err)
	}
	var s types.NVMetSubsys
	if err := json.Unmarshal(result, &s); err != nil {
		return nil, fmt.Errorf("parsing nvmet subsys response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetSubsys (ws) success")
	return &s, nil
}

// CreateNVMetSubsys creates a new NVMe-oF subsystem.
func (c *Client) CreateNVMetSubsys(ctx context.Context, req *types.NVMetSubsysCreateRequest) (*types.NVMetSubsys, error) {
	tflog.Trace(ctx, "CreateNVMetSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.subsys.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating nvmet subsys: %w", err)
	}
	var s types.NVMetSubsys
	if err := json.Unmarshal(result, &s); err != nil {
		return nil, fmt.Errorf("parsing nvmet subsys create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetSubsys (ws) success")
	return &s, nil
}

// UpdateNVMetSubsys updates an NVMe-oF subsystem.
func (c *Client) UpdateNVMetSubsys(ctx context.Context, id int, req *types.NVMetSubsysUpdateRequest) (*types.NVMetSubsys, error) {
	tflog.Trace(ctx, "UpdateNVMetSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.subsys.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating nvmet subsys %d: %w", id, err)
	}
	var s types.NVMetSubsys
	if err := json.Unmarshal(result, &s); err != nil {
		return nil, fmt.Errorf("parsing nvmet subsys update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetSubsys (ws) success")
	return &s, nil
}

// DeleteNVMetSubsys deletes an NVMe-oF subsystem.
func (c *Client) DeleteNVMetSubsys(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetSubsys (ws) start")

	if _, err := c.Call(ctx, "nvmet.subsys.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting nvmet subsys %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetSubsys (ws) success")
	return nil
}

// =============================================================================
// NVMe-oF Port
// =============================================================================

// GetNVMetPort retrieves an NVMe-oF port by ID.
func (c *Client) GetNVMetPort(ctx context.Context, id int) (*types.NVMetPort, error) {
	tflog.Trace(ctx, "GetNVMetPort (ws) start")

	result, err := c.Call(ctx, "nvmet.port.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet port %d: %w", id, err)
	}
	var p types.NVMetPort
	if err := json.Unmarshal(result, &p); err != nil {
		return nil, fmt.Errorf("parsing nvmet port response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetPort (ws) success")
	return &p, nil
}

// CreateNVMetPort creates a new NVMe-oF port.
func (c *Client) CreateNVMetPort(ctx context.Context, req *types.NVMetPortCreateRequest) (*types.NVMetPort, error) {
	tflog.Trace(ctx, "CreateNVMetPort (ws) start")

	result, err := c.Call(ctx, "nvmet.port.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating nvmet port: %w", err)
	}
	var p types.NVMetPort
	if err := json.Unmarshal(result, &p); err != nil {
		return nil, fmt.Errorf("parsing nvmet port create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetPort (ws) success")
	return &p, nil
}

// UpdateNVMetPort updates an NVMe-oF port.
func (c *Client) UpdateNVMetPort(ctx context.Context, id int, req *types.NVMetPortUpdateRequest) (*types.NVMetPort, error) {
	tflog.Trace(ctx, "UpdateNVMetPort (ws) start")

	result, err := c.Call(ctx, "nvmet.port.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating nvmet port %d: %w", id, err)
	}
	var p types.NVMetPort
	if err := json.Unmarshal(result, &p); err != nil {
		return nil, fmt.Errorf("parsing nvmet port update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetPort (ws) success")
	return &p, nil
}

// DeleteNVMetPort deletes an NVMe-oF port.
func (c *Client) DeleteNVMetPort(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetPort (ws) start")

	if _, err := c.Call(ctx, "nvmet.port.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting nvmet port %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetPort (ws) success")
	return nil
}

// =============================================================================
// NVMe-oF Namespace
// =============================================================================

// GetNVMetNamespace retrieves an NVMe-oF namespace by ID.
func (c *Client) GetNVMetNamespace(ctx context.Context, id int) (*types.NVMetNamespace, error) {
	tflog.Trace(ctx, "GetNVMetNamespace (ws) start")

	result, err := c.Call(ctx, "nvmet.namespace.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet namespace %d: %w", id, err)
	}
	var n types.NVMetNamespace
	if err := json.Unmarshal(result, &n); err != nil {
		return nil, fmt.Errorf("parsing nvmet namespace response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetNamespace (ws) success")
	return &n, nil
}

// CreateNVMetNamespace creates a new NVMe-oF namespace.
func (c *Client) CreateNVMetNamespace(ctx context.Context, req *types.NVMetNamespaceCreateRequest) (*types.NVMetNamespace, error) {
	tflog.Trace(ctx, "CreateNVMetNamespace (ws) start")

	result, err := c.Call(ctx, "nvmet.namespace.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating nvmet namespace: %w", err)
	}
	var n types.NVMetNamespace
	if err := json.Unmarshal(result, &n); err != nil {
		return nil, fmt.Errorf("parsing nvmet namespace create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetNamespace (ws) success")
	return &n, nil
}

// UpdateNVMetNamespace updates an NVMe-oF namespace.
func (c *Client) UpdateNVMetNamespace(ctx context.Context, id int, req *types.NVMetNamespaceUpdateRequest) (*types.NVMetNamespace, error) {
	tflog.Trace(ctx, "UpdateNVMetNamespace (ws) start")

	result, err := c.Call(ctx, "nvmet.namespace.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating nvmet namespace %d: %w", id, err)
	}
	var n types.NVMetNamespace
	if err := json.Unmarshal(result, &n); err != nil {
		return nil, fmt.Errorf("parsing nvmet namespace update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateNVMetNamespace (ws) success")
	return &n, nil
}

// DeleteNVMetNamespace deletes an NVMe-oF namespace.
func (c *Client) DeleteNVMetNamespace(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetNamespace (ws) start")

	if _, err := c.Call(ctx, "nvmet.namespace.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting nvmet namespace %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetNamespace (ws) success")
	return nil
}

// =============================================================================
// NVMe-oF Host-Subsys association
// =============================================================================

// GetNVMetHostSubsys retrieves a host-subsys association by ID.
func (c *Client) GetNVMetHostSubsys(ctx context.Context, id int) (*types.NVMetHostSubsys, error) {
	tflog.Trace(ctx, "GetNVMetHostSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.host_subsys.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet host_subsys %d: %w", id, err)
	}
	var hs types.NVMetHostSubsys
	if err := json.Unmarshal(result, &hs); err != nil {
		return nil, fmt.Errorf("parsing nvmet host_subsys response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetHostSubsys (ws) success")
	return &hs, nil
}

// CreateNVMetHostSubsys creates a host-subsys association.
func (c *Client) CreateNVMetHostSubsys(ctx context.Context, req *types.NVMetHostSubsysCreateRequest) (*types.NVMetHostSubsys, error) {
	tflog.Trace(ctx, "CreateNVMetHostSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.host_subsys.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating nvmet host_subsys: %w", err)
	}
	var hs types.NVMetHostSubsys
	if err := json.Unmarshal(result, &hs); err != nil {
		return nil, fmt.Errorf("parsing nvmet host_subsys create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetHostSubsys (ws) success")
	return &hs, nil
}

// DeleteNVMetHostSubsys deletes a host-subsys association.
func (c *Client) DeleteNVMetHostSubsys(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetHostSubsys (ws) start")

	if _, err := c.Call(ctx, "nvmet.host_subsys.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting nvmet host_subsys %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetHostSubsys (ws) success")
	return nil
}

// =============================================================================
// NVMe-oF Port-Subsys association
// =============================================================================

// GetNVMetPortSubsys retrieves a port-subsys association by ID.
func (c *Client) GetNVMetPortSubsys(ctx context.Context, id int) (*types.NVMetPortSubsys, error) {
	tflog.Trace(ctx, "GetNVMetPortSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.port_subsys.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting nvmet port_subsys %d: %w", id, err)
	}
	var ps types.NVMetPortSubsys
	if err := json.Unmarshal(result, &ps); err != nil {
		return nil, fmt.Errorf("parsing nvmet port_subsys response: %w", err)
	}
	tflog.Trace(ctx, "GetNVMetPortSubsys (ws) success")
	return &ps, nil
}

// CreateNVMetPortSubsys creates a port-subsys association.
func (c *Client) CreateNVMetPortSubsys(ctx context.Context, req *types.NVMetPortSubsysCreateRequest) (*types.NVMetPortSubsys, error) {
	tflog.Trace(ctx, "CreateNVMetPortSubsys (ws) start")

	result, err := c.Call(ctx, "nvmet.port_subsys.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating nvmet port_subsys: %w", err)
	}
	var ps types.NVMetPortSubsys
	if err := json.Unmarshal(result, &ps); err != nil {
		return nil, fmt.Errorf("parsing nvmet port_subsys create response: %w", err)
	}
	tflog.Trace(ctx, "CreateNVMetPortSubsys (ws) success")
	return &ps, nil
}

// DeleteNVMetPortSubsys deletes a port-subsys association.
func (c *Client) DeleteNVMetPortSubsys(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteNVMetPortSubsys (ws) start")

	if _, err := c.Call(ctx, "nvmet.port_subsys.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting nvmet port_subsys %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteNVMetPortSubsys (ws) success")
	return nil
}
