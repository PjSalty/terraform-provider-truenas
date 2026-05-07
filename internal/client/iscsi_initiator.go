package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// ISCSIInitiator, ISCSIInitiatorCreateRequest, ISCSIInitiatorUpdateRequest
// moved to internal/types/iscsi_initiator.go in the v2.0
// transport-migration prep.
type (
	ISCSIInitiator              = types.ISCSIInitiator
	ISCSIInitiatorCreateRequest = types.ISCSIInitiatorCreateRequest
	ISCSIInitiatorUpdateRequest = types.ISCSIInitiatorUpdateRequest
)

// GetISCSIInitiator retrieves an iSCSI initiator by ID.
func (c *Client) GetISCSIInitiator(ctx context.Context, id int) (*types.ISCSIInitiator, error) {
	tflog.Trace(ctx, "GetISCSIInitiator start")

	resp, err := c.Get(ctx, fmt.Sprintf("/iscsi/initiator/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI initiator %d: %w", id, err)
	}

	var initiator types.ISCSIInitiator
	if err := json.Unmarshal(resp, &initiator); err != nil {
		return nil, fmt.Errorf("parsing iSCSI initiator response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIInitiator success")
	return &initiator, nil
}

// CreateISCSIInitiator creates a new iSCSI initiator.
func (c *Client) CreateISCSIInitiator(ctx context.Context, req *types.ISCSIInitiatorCreateRequest) (*types.ISCSIInitiator, error) {
	tflog.Trace(ctx, "CreateISCSIInitiator start")

	resp, err := c.Post(ctx, "/iscsi/initiator", req)
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI initiator: %w", err)
	}

	var initiator types.ISCSIInitiator
	if err := json.Unmarshal(resp, &initiator); err != nil {
		return nil, fmt.Errorf("parsing iSCSI initiator create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIInitiator success")
	return &initiator, nil
}

// UpdateISCSIInitiator updates an existing iSCSI initiator.
func (c *Client) UpdateISCSIInitiator(ctx context.Context, id int, req *types.ISCSIInitiatorUpdateRequest) (*types.ISCSIInitiator, error) {
	tflog.Trace(ctx, "UpdateISCSIInitiator start")

	resp, err := c.Put(ctx, fmt.Sprintf("/iscsi/initiator/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI initiator %d: %w", id, err)
	}

	var initiator types.ISCSIInitiator
	if err := json.Unmarshal(resp, &initiator); err != nil {
		return nil, fmt.Errorf("parsing iSCSI initiator update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIInitiator success")
	return &initiator, nil
}

// DeleteISCSIInitiator deletes an iSCSI initiator.
func (c *Client) DeleteISCSIInitiator(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIInitiator start")

	_, err := c.Delete(ctx, fmt.Sprintf("/iscsi/initiator/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting iSCSI initiator %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteISCSIInitiator success")
	return nil
}
