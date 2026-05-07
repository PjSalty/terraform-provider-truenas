package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ISCSITarget, ISCSITargetGroup, ISCSITargetCreateRequest,
// ISCSITargetUpdateRequest moved to internal/types/iscsi_target.go in
// the v2.0 transport-migration prep.
type (
	ISCSITarget              = types.ISCSITarget
	ISCSITargetGroup         = types.ISCSITargetGroup
	ISCSITargetCreateRequest = types.ISCSITargetCreateRequest
	ISCSITargetUpdateRequest = types.ISCSITargetUpdateRequest
)

// GetISCSITarget retrieves an iSCSI target by ID.
func (c *Client) GetISCSITarget(ctx context.Context, id int) (*types.ISCSITarget, error) {
	tflog.Trace(ctx, "GetISCSITarget start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/target/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI target %d: %w", id, err)
	}

	var target types.ISCSITarget
	if err := json.Unmarshal(resp, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSITarget success")
	return &target, nil
}

// CreateISCSITarget creates a new iSCSI target.
func (c *Client) CreateISCSITarget(ctx context.Context, req *types.ISCSITargetCreateRequest) (*types.ISCSITarget, error) {
	tflog.Trace(ctx, "CreateISCSITarget start")

	resp, err := c.Post(ctx, "/iscsi/target", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI target: %w", err)
	}

	var target types.ISCSITarget
	if err := json.Unmarshal(resp, &target); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSITarget success")
	return &target, nil
}

// UpdateISCSITarget updates an existing iSCSI target.
func (c *Client) UpdateISCSITarget(ctx context.Context, id int, req *types.ISCSITargetUpdateRequest) (*types.ISCSITarget, error) {
	tflog.Trace(ctx, "UpdateISCSITarget start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/target/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI target %d: %w", id, err)
	}

	var target types.ISCSITarget
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
