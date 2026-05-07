package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// Replication, ReplicationCreateRequest, ReplicationUpdateRequest moved
// to internal/types/replication.go in the v2.0 transport-migration prep.
type (
	Replication              = types.Replication
	ReplicationCreateRequest = types.ReplicationCreateRequest
	ReplicationUpdateRequest = types.ReplicationUpdateRequest
)

// GetReplication retrieves a replication task by ID.
func (c *Client) GetReplication(ctx context.Context, id int) (*types.Replication, error) {
	tflog.Trace(ctx, "GetReplication start")

	resp, err := c.Get(ctx, fmt.Sprintf("/replication/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting replication %d: %w", id, err)
	}

	var repl types.Replication
	if err := json.Unmarshal(resp, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication response: %w", err)
	}

	tflog.Trace(ctx, "GetReplication success")
	return &repl, nil
}

// CreateReplication creates a new replication task.
func (c *Client) CreateReplication(ctx context.Context, req *types.ReplicationCreateRequest) (*types.Replication, error) {
	tflog.Trace(ctx, "CreateReplication start")

	resp, err := c.Post(ctx, "/replication", req)
	if err != nil {
		return nil, fmt.Errorf("creating replication: %w", err)
	}

	var repl types.Replication
	if err := json.Unmarshal(resp, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication create response: %w", err)
	}

	tflog.Trace(ctx, "CreateReplication success")
	return &repl, nil
}

// UpdateReplication updates an existing replication task.
func (c *Client) UpdateReplication(ctx context.Context, id int, req *types.ReplicationUpdateRequest) (*types.Replication, error) {
	tflog.Trace(ctx, "UpdateReplication start")

	resp, err := c.Put(ctx, fmt.Sprintf("/replication/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating replication %d: %w", id, err)
	}

	var repl types.Replication
	if err := json.Unmarshal(resp, &repl); err != nil {
		return nil, fmt.Errorf("parsing replication update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateReplication success")
	return &repl, nil
}

// DeleteReplication deletes a replication task.
func (c *Client) DeleteReplication(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteReplication start")

	_, err := c.Delete(ctx, fmt.Sprintf("/replication/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting replication %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteReplication success")
	return nil
}
