package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for iSCSI initiators: iscsi.initiator.{...}.

// GetISCSIInitiator retrieves an iSCSI initiator by ID.
func (c *Client) GetISCSIInitiator(ctx context.Context, id int) (*types.ISCSIInitiator, error) {
	tflog.Trace(ctx, "GetISCSIInitiator (ws) start")

	result, err := c.Call(ctx, "iscsi.initiator.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI initiator %d: %w", id, err)
	}

	var initiator types.ISCSIInitiator
	if err := json.Unmarshal(result, &initiator); err != nil {
		return nil, fmt.Errorf("parsing iSCSI initiator response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIInitiator (ws) success")
	return &initiator, nil
}

// CreateISCSIInitiator creates a new iSCSI initiator.
func (c *Client) CreateISCSIInitiator(ctx context.Context, req *types.ISCSIInitiatorCreateRequest) (*types.ISCSIInitiator, error) {
	tflog.Trace(ctx, "CreateISCSIInitiator (ws) start")

	result, err := c.Call(ctx, "iscsi.initiator.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating iSCSI initiator: %w", err)
	}

	var initiator types.ISCSIInitiator
	if err := json.Unmarshal(result, &initiator); err != nil {
		return nil, fmt.Errorf("parsing iSCSI initiator create response: %w", err)
	}

	tflog.Trace(ctx, "CreateISCSIInitiator (ws) success")
	return &initiator, nil
}

// UpdateISCSIInitiator updates an existing iSCSI initiator.
func (c *Client) UpdateISCSIInitiator(ctx context.Context, id int, req *types.ISCSIInitiatorUpdateRequest) (*types.ISCSIInitiator, error) {
	tflog.Trace(ctx, "UpdateISCSIInitiator (ws) start")

	result, err := c.Call(ctx, "iscsi.initiator.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI initiator %d: %w", id, err)
	}

	var initiator types.ISCSIInitiator
	if err := json.Unmarshal(result, &initiator); err != nil {
		return nil, fmt.Errorf("parsing iSCSI initiator update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIInitiator (ws) success")
	return &initiator, nil
}

// DeleteISCSIInitiator deletes an iSCSI initiator.
func (c *Client) DeleteISCSIInitiator(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteISCSIInitiator (ws) start")

	if _, err := c.Call(ctx, "iscsi.initiator.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting iSCSI initiator %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteISCSIInitiator (ws) success")
	return nil
}
