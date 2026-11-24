package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for the replication global configuration:
// replication.config.{config,update}.

// GetReplicationConfig retrieves the replication configuration singleton.
func (c *Client) GetReplicationConfig(ctx context.Context) (*types.ReplicationConfig, error) {
	tflog.Trace(ctx, "GetReplicationConfig (ws) start")

	result, err := c.Call(ctx, "replication.config.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting replication config: %w", err)
	}

	var cfg types.ReplicationConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing replication config response: %w", err)
	}

	tflog.Trace(ctx, "GetReplicationConfig (ws) success")
	return &cfg, nil
}

// UpdateReplicationConfig applies the replication configuration and returns
// the stored entry. A change is passed to the running replication engine at
// once; nothing is restarted.
func (c *Client) UpdateReplicationConfig(ctx context.Context, req *types.ReplicationConfigUpdateRequest) (*types.ReplicationConfig, error) {
	tflog.Trace(ctx, "UpdateReplicationConfig (ws) start")

	result, err := c.Call(ctx, "replication.config.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating replication config: %w", err)
	}

	var cfg types.ReplicationConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing replication config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateReplicationConfig (ws) success")
	return &cfg, nil
}
