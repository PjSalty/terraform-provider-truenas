package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for SSH service: ssh.{config,update}.

// GetSSHConfig retrieves the SSH service configuration.
func (c *Client) GetSSHConfig(ctx context.Context) (*types.SSHConfig, error) {
	tflog.Trace(ctx, "GetSSHConfig (ws) start")

	result, err := c.Call(ctx, "ssh.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting SSH config: %w", err)
	}

	var config types.SSHConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SSH config response: %w", err)
	}

	tflog.Trace(ctx, "GetSSHConfig (ws) success")
	return &config, nil
}

// UpdateSSHConfig updates the SSH service configuration.
func (c *Client) UpdateSSHConfig(ctx context.Context, req *types.SSHConfigUpdateRequest) (*types.SSHConfig, error) {
	tflog.Trace(ctx, "UpdateSSHConfig (ws) start")

	result, err := c.Call(ctx, "ssh.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating SSH config: %w", err)
	}

	var config types.SSHConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SSH config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSSHConfig (ws) success")
	return &config, nil
}
