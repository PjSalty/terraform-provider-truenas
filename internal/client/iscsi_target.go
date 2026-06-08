package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- iSCSI Target API ---

// ISCSITarget represents an iSCSI target.
type ISCSITarget struct {
	ID     int                `json:"id"`
	Name   string             `json:"name"`
	Alias  string             `json:"alias,omitempty"`
	Mode   string             `json:"mode"`
	Groups []ISCSITargetGroup `json:"groups,omitempty"`
}

// ISCSITargetGroup represents an iSCSI target group.
type ISCSITargetGroup struct {
	Portal     int    `json:"portal"`
	Initiator  int    `json:"initiator"`
	AuthMethod string `json:"authmethod"`
	Auth       int    `json:"auth"`
}

// ISCSITargetCreateRequest represents the request to create an iSCSI target.
type ISCSITargetCreateRequest struct {
	Name   string             `json:"name"`
	Alias  string             `json:"alias,omitempty"`
	Mode   string             `json:"mode"`
	Groups []ISCSITargetGroup `json:"groups,omitempty"`
}

// ISCSITargetUpdateRequest represents the request to update an iSCSI target.
type ISCSITargetUpdateRequest struct {
	Name   string             `json:"name,omitempty"`
	Alias  string             `json:"alias,omitempty"`
	Mode   string             `json:"mode,omitempty"`
	Groups []ISCSITargetGroup `json:"groups,omitempty"`
}

// GetISCSITarget retrieves an iSCSI target by ID.
func (c *Client) GetISCSITarget(ctx context.Context, id int) (*ISCSITarget, error) {
	tflog.Trace(ctx, "GetISCSITarget start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/target/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI target %d: %w", id, err)
	}

	var target ISCSITarget
	if err := json.Unmarshal(resp, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSITarget success")
	return &target, nil
}

// CreateISCSITarget creates a new iSCSI target.
func (c *Client) CreateISCSITarget(ctx context.Context, req *ISCSITargetCreateRequest) (*ISCSITarget, error) {
	tflog.Trace(ctx, "CreateISCSITarget start")

	resp, err := c.Post(ctx, "/iscsi/target", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI target: %w", err)
	}

	var target ISCSITarget
	if err := json.Unmarshal(resp, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSITarget success")
	return &target, nil
}

// UpdateISCSITarget updates an existing iSCSI target.
func (c *Client) UpdateISCSITarget(ctx context.Context, id int, req *ISCSITargetUpdateRequest) (*ISCSITarget, error) {
	tflog.Trace(ctx, "UpdateISCSITarget start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/target/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI target %d: %w", id, err)
	}

	var target ISCSITarget
	if err := json.Unmarshal(resp, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSITarget success")
	return &target, nil
}

// DeleteISCSITarget deletes an iSCSI target.
func (c *Client) DeleteISCSITarget(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSITarget start")

	_, err := c.Delete(ctx, fmt.Sprintf("/iscsi/target/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting iSCSI target %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteISCSITarget success")
	return nil
}
