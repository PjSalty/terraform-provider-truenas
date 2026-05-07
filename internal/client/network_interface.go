package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// --- Network Interface API ---
//
// TrueNAS interface configuration uses a staged commit workflow:
//
//   1. POST /interface        -> creates a PENDING change
//   2. POST /interface/commit -> applies the pending changes (with a
//                                checkin_timeout; if no checkin arrives
//                                before the timeout, changes are rolled back)
//   3. GET  /interface/checkin -> acknowledges the changes as good, canceling
//                                the rollback timer and making them permanent
//
// CreateInterface() / UpdateInterface() / DeleteInterface() perform steps 1-3
// in sequence so callers (the Terraform resource) get a simple synchronous API.
//
// The interface ID is a STRING (the interface name, e.g. "br0", "vlan10",
// "bond0") — not a numeric ID.

// NetworkInterface, NetworkInterfaceAlias, and the create/update/commit
// requests moved to internal/types/network_interface.go in the v2.0
// transport-migration prep.
type (
	NetworkInterface              = types.NetworkInterface
	NetworkInterfaceAlias         = types.NetworkInterfaceAlias
	NetworkInterfaceCreateRequest = types.NetworkInterfaceCreateRequest
	NetworkInterfaceUpdateRequest = types.NetworkInterfaceUpdateRequest
	InterfaceCommitRequest        = types.InterfaceCommitRequest
)

// commitAndCheckin applies any pending interface changes and acknowledges them.
// This is the glue that turns TrueNAS's staged workflow into a single call.
func (c *Client) commitAndCheckin(ctx context.Context) error {
	commitReq := &types.InterfaceCommitRequest{
		Rollback:       true,
		CheckinTimeout: 60,
	}
	if _, err := c.Post(ctx, "/interface/commit", commitReq); err != nil {
		return fmt.Errorf("committing interface changes: %w", err)
	}
	// Checkin to cancel the rollback timer and make changes permanent.
	if _, err := c.Get(ctx, "/interface/checkin"); err != nil {
		return fmt.Errorf("checking in interface changes: %w", err)
	}
	return nil
}

// GetInterface retrieves a network interface by its ID (name).
func (c *Client) GetInterface(ctx context.Context, id string) (*types.NetworkInterface, error) {
	tflog.Trace(ctx, "GetInterface start")

	encoded := url.PathEscape(id)
	resp, err := c.Get(ctx, fmt.Sprintf("/interface/id/%s", encoded))
	if err != nil {
		return nil, fmt.Errorf("getting interface %q: %w", id, err)
	}
	var iface types.NetworkInterface
	if err := json.Unmarshal(resp, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface response: %w", err)
	}
	tflog.Trace(ctx, "GetInterface success")
	return &iface, nil
}

// ListInterfaces retrieves all network interfaces.
func (c *Client) ListInterfaces(ctx context.Context) ([]types.NetworkInterface, error) {
	tflog.Trace(ctx, "ListInterfaces start")

	resp, err := c.Get(ctx, "/interface")
	if err != nil {
		return nil, fmt.Errorf("listing interfaces: %w", err)
	}
	var ifaces []types.NetworkInterface
	if err := json.Unmarshal(resp, &ifaces); err != nil {
		return nil, fmt.Errorf("parsing interfaces list: %w", err)
	}
	tflog.Trace(ctx, "ListInterfaces success")
	return ifaces, nil
}

// CreateInterface creates a virtual interface (BRIDGE, LINK_AGGREGATION, VLAN)
// and applies the change via commit + checkin.
func (c *Client) CreateInterface(ctx context.Context, req *types.NetworkInterfaceCreateRequest) (*types.NetworkInterface, error) {
	tflog.Trace(ctx, "CreateInterface start")

	resp, err := c.Post(ctx, "/interface", req)
	if err != nil {
		return nil, fmt.Errorf("creating interface: %w", err)
	}

	var iface types.NetworkInterface
	if err := json.Unmarshal(resp, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface create response: %w", err)
	}

	if err := c.commitAndCheckin(ctx); err != nil {
		return nil, err
	}

	// Re-read the interface post-commit so we capture the final persisted state.
	tflog.Trace(ctx, "CreateInterface success")
	return c.GetInterface(ctx, iface.ID)
}

// UpdateInterface updates an existing interface by ID (name) and applies
// the change via commit + checkin.
func (c *Client) UpdateInterface(ctx context.Context, id string, req *types.NetworkInterfaceUpdateRequest) (*types.NetworkInterface, error) {
	tflog.Trace(ctx, "UpdateInterface start")

	encoded := url.PathEscape(id)
	resp, err := c.Put(ctx, fmt.Sprintf("/interface/id/%s", encoded), req)
	if err != nil {
		return nil, fmt.Errorf("updating interface %q: %w", id, err)
	}

	var iface types.NetworkInterface
	if err := json.Unmarshal(resp, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface update response: %w", err)
	}

	if err := c.commitAndCheckin(ctx); err != nil {
		return nil, err
	}

	tflog.Trace(ctx, "UpdateInterface success")
	return c.GetInterface(ctx, id)
}

// DeleteInterface deletes an interface by ID (name) and applies the change
// via commit + checkin.
func (c *Client) DeleteInterface(ctx context.Context, id string) error {
	tflog.Trace(ctx, "DeleteInterface start")

	encoded := url.PathEscape(id)
	if _, err := c.Delete(ctx, fmt.Sprintf("/interface/id/%s", encoded)); err != nil {
		return fmt.Errorf("deleting interface %q: %w", id, err)
	}

	if err := c.commitAndCheckin(ctx); err != nil {
		return err
	}

	tflog.Trace(ctx, "DeleteInterface success")
	return nil
}
