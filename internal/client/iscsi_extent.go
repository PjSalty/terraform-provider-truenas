package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ISCSIExtent (with helpers GetDisk/GetFilesize), ISCSIExtentCreateRequest,
// ISCSIExtentUpdateRequest moved to internal/types/iscsi_extent.go in
// the v2.0 transport-migration prep.
type (
	ISCSIExtent              = types.ISCSIExtent
	ISCSIExtentCreateRequest = types.ISCSIExtentCreateRequest
	ISCSIExtentUpdateRequest = types.ISCSIExtentUpdateRequest
)

// GetISCSIExtent retrieves an iSCSI extent by ID.
func (c *Client) GetISCSIExtent(ctx context.Context, id int) (*types.ISCSIExtent, error) {
	tflog.Trace(ctx, "GetISCSIExtent start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/extent/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI extent %d: %w", id, err)
	}

	var extent types.ISCSIExtent
	if err := json.Unmarshal(resp, &extent); err != nil {
		return nil, fmt.Errorf("parsing iSCSI extent response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIExtent success")
	return &extent, nil
}

// CreateISCSIExtent creates a new iSCSI extent.
func (c *Client) CreateISCSIExtent(ctx context.Context, req *types.ISCSIExtentCreateRequest) (*types.ISCSIExtent, error) {
	tflog.Trace(ctx, "CreateISCSIExtent start")

	resp, err := c.Post(ctx, "/iscsi/extent", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI extent: %w", err)
	}

	var extent types.ISCSIExtent
	if err := json.Unmarshal(resp, &extent); err != nil {
		return nil, fmt.Errorf("parsing iSCSI extent create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIExtent success")
	return &extent, nil
}

// UpdateISCSIExtent updates an existing iSCSI extent.
func (c *Client) UpdateISCSIExtent(ctx context.Context, id int, req *types.ISCSIExtentUpdateRequest) (*types.ISCSIExtent, error) {
	tflog.Trace(ctx, "UpdateISCSIExtent start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/extent/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI extent %d: %w", id, err)
	}

	var extent types.ISCSIExtent
	if err := json.Unmarshal(resp, &extent); err != nil {
		return nil, fmt.Errorf("parsing iSCSI extent update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIExtent success")
	return &extent, nil
}

// DeleteISCSIExtent deletes an iSCSI extent.
func (c *Client) DeleteISCSIExtent(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIExtent start")

	_, err := c.Delete(ctx, fmt.Sprintf("/iscsi/extent/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting iSCSI extent %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteISCSIExtent success")
	return nil
}
