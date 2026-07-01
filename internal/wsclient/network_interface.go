package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for network interfaces:
//   interface.{query,get_instance,create,update,delete,commit,checkin}.
//
// As with the REST client, Create/Update/Delete drive the staged commit
// workflow (commit + checkin) so callers see a synchronous API.

// commitAndCheckin applies any pending interface changes and acknowledges them.
// When rollback is true the commit uses a short 5-second rollback timer and
// sends a checkin afterward.  When rollback is false the commit is applied
// immediately with no rollback window — faster but riskier because a bad
// change cannot be recovered automatically.
func (c *Client) commitAndCheckin(ctx context.Context, rollback bool) error {
	commitReq := &types.InterfaceCommitRequest{
		Rollback:       rollback,
		CheckinTimeout: 5,
	}
	if _, err := c.Call(ctx, "interface.commit",
		[]interface{}{commitReq}, CallOptions{}); err != nil {
		return fmt.Errorf("committing interface changes: %w", err)
	}
	if !rollback {
		return nil
	}
	if _, err := c.Call(ctx, "interface.checkin", nil,
		CallOptions{Read: true, Idempotent: true}); err != nil {
		return fmt.Errorf("checking in interface changes: %w", err)
	}
	return nil
}

// GetInterface retrieves a network interface by its ID (name).
func (c *Client) GetInterface(ctx context.Context, id string) (*types.NetworkInterface, error) {
	tflog.Trace(ctx, "GetInterface (ws) start")

	result, err := c.Call(ctx, "interface.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting interface %q: %w", id, err)
	}
	var iface types.NetworkInterface
	if err := json.Unmarshal(result, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface response: %w", err)
	}
	tflog.Trace(ctx, "GetInterface (ws) success")
	return &iface, nil
}

// ListInterfaces retrieves all network interfaces.
func (c *Client) ListInterfaces(ctx context.Context) ([]types.NetworkInterface, error) {
	tflog.Trace(ctx, "ListInterfaces (ws) start")

	result, err := c.Call(ctx, "interface.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing interfaces: %w", err)
	}
	var ifaces []types.NetworkInterface
	if err := json.Unmarshal(result, &ifaces); err != nil {
		return nil, fmt.Errorf("parsing interfaces list: %w", err)
	}
	tflog.Trace(ctx, "ListInterfaces (ws) success")
	return ifaces, nil
}

// CreateInterface creates a virtual interface (BRIDGE, LINK_AGGREGATION, VLAN)
// and applies the change via commit + checkin.
func (c *Client) CreateInterface(ctx context.Context, req *types.NetworkInterfaceCreateRequest, rollback bool) (*types.NetworkInterface, error) {
	tflog.Trace(ctx, "CreateInterface (ws) start")

	result, err := c.Call(ctx, "interface.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating interface: %w", err)
	}

	var iface types.NetworkInterface
	if err := json.Unmarshal(result, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface create response: %w", err)
	}

	if err := c.commitAndCheckin(ctx, rollback); err != nil {
		return nil, err
	}

	tflog.Trace(ctx, "CreateInterface (ws) success")
	return c.GetInterface(ctx, iface.ID)
}

// UpdateInterface updates an existing interface by ID (name) and applies
// the change via commit + checkin.
func (c *Client) UpdateInterface(ctx context.Context, id string, req *types.NetworkInterfaceUpdateRequest, rollback bool) (*types.NetworkInterface, error) {
	tflog.Trace(ctx, "UpdateInterface (ws) start")

	result, err := c.Call(ctx, "interface.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating interface %q: %w", id, err)
	}

	var iface types.NetworkInterface
	if err := json.Unmarshal(result, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface update response: %w", err)
	}

	if err := c.commitAndCheckin(ctx, rollback); err != nil {
		return nil, err
	}

	tflog.Trace(ctx, "UpdateInterface (ws) success")
	return c.GetInterface(ctx, id)
}

// DeleteInterface deletes an interface by ID (name) and applies the change
// via commit + checkin.
func (c *Client) DeleteInterface(ctx context.Context, id string, rollback bool) error {
	tflog.Trace(ctx, "DeleteInterface (ws) start")

	if _, err := c.Call(ctx, "interface.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting interface %q: %w", id, err)
	}

	if err := c.commitAndCheckin(ctx, rollback); err != nil {
		return err
	}

	tflog.Trace(ctx, "DeleteInterface (ws) success")
	return nil
}
