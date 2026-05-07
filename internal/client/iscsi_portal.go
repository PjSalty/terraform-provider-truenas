package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ISCSIPortal, ISCSIPortalListen, ISCSIPortalCreateRequest,
// ISCSIPortalUpdateRequest (with their custom MarshalJSON for SCALE
// 25.10 compat) moved to internal/types/iscsi_portal.go in the v2.0
// transport-migration prep.
type (
	ISCSIPortal              = types.ISCSIPortal
	ISCSIPortalListen        = types.ISCSIPortalListen
	ISCSIPortalCreateRequest = types.ISCSIPortalCreateRequest
	ISCSIPortalUpdateRequest = types.ISCSIPortalUpdateRequest
)

// GetISCSIPortal retrieves an iSCSI portal by ID.
func (c *Client) GetISCSIPortal(ctx context.Context, id int) (*types.ISCSIPortal, error) {
	tflog.Trace(ctx, "GetISCSIPortal start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/portal/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI portal %d: %w", id, err)
	}

	var portal types.ISCSIPortal
	if err := json.Unmarshal(resp, &portal); err != nil {
		return nil, fmt.Errorf("parsing iSCSI portal response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIPortal success")
	return &portal, nil
}

// CreateISCSIPortal creates a new iSCSI portal.
func (c *Client) CreateISCSIPortal(ctx context.Context, req *types.ISCSIPortalCreateRequest) (*types.ISCSIPortal, error) {
	tflog.Trace(ctx, "CreateISCSIPortal start")

	resp, err := c.Post(ctx, "/iscsi/portal", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI portal: %w", err)
	}

	var portal types.ISCSIPortal
	if err := json.Unmarshal(resp, &portal); err != nil {
		return nil, fmt.Errorf("parsing iSCSI portal create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIPortal success")
	return &portal, nil
}

// UpdateISCSIPortal updates an existing iSCSI portal.
func (c *Client) UpdateISCSIPortal(ctx context.Context, id int, req *types.ISCSIPortalUpdateRequest) (*types.ISCSIPortal, error) {
	tflog.Trace(ctx, "UpdateISCSIPortal start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/portal/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI portal %d: %w", id, err)
	}

	var portal types.ISCSIPortal
	if err := json.Unmarshal(resp, &portal); err != nil {
		return nil, fmt.Errorf("parsing iSCSI portal update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIPortal success")
	return &portal, nil
}

// DeleteISCSIPortal deletes an iSCSI portal.
func (c *Client) DeleteISCSIPortal(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIPortal start")

	_, err := c.Delete(ctx, fmt.Sprintf("/iscsi/portal/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting iSCSI portal %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteISCSIPortal success")
	return nil
}
