package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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
// This client exposes CreateInterface() that performs steps 1-3 in sequence
// so callers (the Terraform resource) get a simple synchronous API. The same
// pattern is followed for UpdateInterface() and DeleteInterface().
//
// Note that the interface ID is a STRING (the interface name, e.g. "br0",
// "vlan10", "bond0") — not a numeric ID.

// NetworkInterface represents a network interface as returned by /interface.
// Only the fields we surface in Terraform state are declared; additional
// fields are ignored via the absence of strict parsing (API returns a lot
// of state/runtime data we don't care about on the client side).
type NetworkInterface struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	Type                string                  `json:"type"`
	Description         string                  `json:"description"`
	IPv4DHCP            bool                    `json:"ipv4_dhcp"`
	IPv6Auto            bool                    `json:"ipv6_auto"`
	MTU                 *int                    `json:"mtu"`
	Aliases             []NetworkInterfaceAlias `json:"aliases"`
	BridgeMembers       []string                `json:"bridge_members"`
	LagProtocol         string                  `json:"lag_protocol"`
	LagPorts            []string                `json:"lag_ports"`
	VlanParentInterface string                  `json:"vlan_parent_interface"`
	VlanTag             *int                    `json:"vlan_tag"`
	VlanPCP             *int                    `json:"vlan_pcp"`
}

// NetworkInterfaceAlias represents an IP alias on an interface.
type NetworkInterfaceAlias struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Netmask int    `json:"netmask"`
}

// NetworkInterfaceCreateRequest is the POST /interface payload.
type NetworkInterfaceCreateRequest struct {
	Name                string                  `json:"name,omitempty"`
	Description         string                  `json:"description,omitempty"`
	Type                string                  `json:"type"`
	IPv4DHCP            bool                    `json:"ipv4_dhcp"`
	IPv6Auto            bool                    `json:"ipv6_auto"`
	Aliases             []NetworkInterfaceAlias `json:"aliases,omitempty"`
	MTU                 *int                    `json:"mtu,omitempty"`
	BridgeMembers       []string                `json:"bridge_members,omitempty"`
	LagProtocol         string                  `json:"lag_protocol,omitempty"`
	LagPorts            []string                `json:"lag_ports,omitempty"`
	VlanParentInterface string                  `json:"vlan_parent_interface,omitempty"`
	VlanTag             *int                    `json:"vlan_tag,omitempty"`
	VlanPCP             *int                    `json:"vlan_pcp,omitempty"`
}

// NetworkInterfaceUpdateRequest is the PUT /interface/id/{id_} payload.
// Mirrors the create request; all fields are optional for partial updates.
type NetworkInterfaceUpdateRequest struct {
	Description         *string                 `json:"description,omitempty"`
	IPv4DHCP            *bool                   `json:"ipv4_dhcp,omitempty"`
	IPv6Auto            *bool                   `json:"ipv6_auto,omitempty"`
	Aliases             []NetworkInterfaceAlias `json:"aliases,omitempty"`
	MTU                 *int                    `json:"mtu,omitempty"`
	BridgeMembers       []string                `json:"bridge_members,omitempty"`
	LagProtocol         *string                 `json:"lag_protocol,omitempty"`
	LagPorts            []string                `json:"lag_ports,omitempty"`
	VlanParentInterface *string                 `json:"vlan_parent_interface,omitempty"`
	VlanTag             *int                    `json:"vlan_tag,omitempty"`
	VlanPCP             *int                    `json:"vlan_pcp,omitempty"`
}

// InterfaceCommitRequest is the POST /interface/commit payload.
type InterfaceCommitRequest struct {
	Rollback       bool `json:"rollback"`
	CheckinTimeout int  `json:"checkin_timeout"`
}

// commitAndCheckin applies any pending interface changes and acknowledges them.
// This is the glue that turns TrueNAS's staged workflow into a single call.
func (c *Client) commitAndCheckin(ctx context.Context) error {
	commitReq := &InterfaceCommitRequest{
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
func (c *Client) GetInterface(ctx context.Context, id string) (*NetworkInterface, error) {
	tflog.Trace(ctx, "GetInterface start")

	encoded := url.PathEscape(id)
	resp, err := c.Get(ctx, fmt.Sprintf("/interface/id/%s", encoded))
	if err != nil {
		return nil, fmt.Errorf("getting interface %q: %w", id, err)
	}
	var iface NetworkInterface
	if err := json.Unmarshal(resp, &iface); err != nil {
		return nil, fmt.Errorf("parsing interface response: %w", err)
	}
	tflog.Trace(ctx, "GetInterface success")
	return &iface, nil
}

// ListInterfaces retrieves all network interfaces.
func (c *Client) ListInterfaces(ctx context.Context) ([]NetworkInterface, error) {
	tflog.Trace(ctx, "ListInterfaces start")

	resp, err := c.Get(ctx, "/interface")
	if err != nil {
		return nil, fmt.Errorf("listing interfaces: %w", err)
	}
	var ifaces []NetworkInterface
	if err := json.Unmarshal(resp, &ifaces); err != nil {
		return nil, fmt.Errorf("parsing interfaces list: %w", err)
	}
	tflog.Trace(ctx, "ListInterfaces success")
	return ifaces, nil
}

// CreateInterface creates a virtual interface (BRIDGE, LINK_AGGREGATION, VLAN)
// and applies the change via commit + checkin.
func (c *Client) CreateInterface(ctx context.Context, req *NetworkInterfaceCreateRequest) (*NetworkInterface, error) {
	tflog.Trace(ctx, "CreateInterface start")

	resp, err := c.Post(ctx, "/interface", req)
	if err != nil {
		return nil, fmt.Errorf("creating interface: %w", err)
	}

	var iface NetworkInterface
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
func (c *Client) UpdateInterface(ctx context.Context, id string, req *NetworkInterfaceUpdateRequest) (*NetworkInterface, error) {
	tflog.Trace(ctx, "UpdateInterface start")

	encoded := url.PathEscape(id)
	resp, err := c.Put(ctx, fmt.Sprintf("/interface/id/%s", encoded), req)
	if err != nil {
		return nil, fmt.Errorf("updating interface %q: %w", id, err)
	}

	var iface NetworkInterface
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
