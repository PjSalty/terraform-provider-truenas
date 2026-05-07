package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ISCSITargetExtent, ISCSITargetExtentCreateRequest,
// ISCSITargetExtentUpdateRequest moved to
// internal/types/iscsi_targetextent.go in the v2.0 transport-migration prep.
type (
	ISCSITargetExtent              = types.ISCSITargetExtent
	ISCSITargetExtentCreateRequest = types.ISCSITargetExtentCreateRequest
	ISCSITargetExtentUpdateRequest = types.ISCSITargetExtentUpdateRequest
)

// GetISCSITargetExtent retrieves an iSCSI target-extent association by ID.
func (c *Client) GetISCSITargetExtent(ctx context.Context, id int) (*types.ISCSITargetExtent, error) {
	tflog.Trace(ctx, "GetISCSITargetExtent start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/targetextent/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI target-extent %d: %w", id, err)
	}

	var te types.ISCSITargetExtent
	if err := json.Unmarshal(resp, &te); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target-extent response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSITargetExtent success")
	return &te, nil
}

// CreateISCSITargetExtent creates a new iSCSI target-extent association.
func (c *Client) CreateISCSITargetExtent(ctx context.Context, req *types.ISCSITargetExtentCreateRequest) (*types.ISCSITargetExtent, error) {
	tflog.Trace(ctx, "CreateISCSITargetExtent start")

	resp, err := c.Post(ctx, "/iscsi/targetextent", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI target-extent: %w", err)
	}

	var te types.ISCSITargetExtent
	if err := json.Unmarshal(resp, &te); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target-extent create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSITargetExtent success")
	return &te, nil
}

// UpdateISCSITargetExtent updates an existing iSCSI target-extent association.
func (c *Client) UpdateISCSITargetExtent(ctx context.Context, id int, req *types.ISCSITargetExtentUpdateRequest) (*types.ISCSITargetExtent, error) {
	tflog.Trace(ctx, "UpdateISCSITargetExtent start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/targetextent/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI target-extent %d: %w", id, err)
	}

	var te types.ISCSITargetExtent
	if err := json.Unmarshal(resp, &te); err != nil {
		return nil, fmt.Errorf("parsing iSCSI target-extent update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSITargetExtent success")
	return &te, nil
}

// DeleteISCSITargetExtent deletes an iSCSI target-extent association.
func (c *Client) DeleteISCSITargetExtent(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSITargetExtent start")

	_, err := c.Delete(ctx, fmt.Sprintf("/iscsi/targetextent/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting iSCSI target-extent %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteISCSITargetExtent success")
	return nil
}
