package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for replications: replication.{...}.

// GetReplication retrieves a replication task by ID.
func (c *Client) GetReplication(ctx context.Context, id int) (*types.Replication, error) {
	tflog.Trace(ctx, "GetReplication (ws) start")

	result, err := c.Call(ctx, "replication.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting replication %d: %w", id, err)
	}

	var repl types.Replication
	if err := json.Unmarshal(result, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication response: %w", err)
	}

	tflog.Trace(ctx, "GetReplication (ws) success")
	return &repl, nil
}

// CreateReplication creates a new replication task.
func (c *Client) CreateReplication(ctx context.Context, req *types.ReplicationCreateRequest) (*types.Replication, error) {
	tflog.Trace(ctx, "CreateReplication (ws) start")

	result, err := c.Call(ctx, "replication.create",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("creating replication: %w", err)
	}

	var repl types.Replication
	if err := json.Unmarshal(result, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication create response: %w", err)
	}

	tflog.Trace(ctx, "CreateReplication (ws) success")
	return &repl, nil
}

// UpdateReplication updates an existing replication task.
func (c *Client) UpdateReplication(ctx context.Context, id int, req *types.ReplicationUpdateRequest) (*types.Replication, error) {
	tflog.Trace(ctx, "UpdateReplication (ws) start")

	result, err := c.Call(ctx, "replication.update",
		[]interface{}{id, req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating replication %d: %w", id, err)
	}

	var repl types.Replication
	if err := json.Unmarshal(result, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateReplication (ws) success")
	return &repl, nil
}

// DeleteReplication deletes a replication task.
func (c *Client) DeleteReplication(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteReplication (ws) start")

	if _, err := c.Call(ctx, "replication.delete",
		[]interface{}{id}, CallOptions{}); err != nil {
		return fmt.Errorf("deleting replication %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteReplication (ws) success")
	return nil
}
