package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// SSHConfig, SSHConfigUpdateRequest moved to internal/types/ssh_config.go
// in the v2.0 transport-migration prep.
type (
	SSHConfig              = types.SSHConfig
	SSHConfigUpdateRequest = types.SSHConfigUpdateRequest
)

// GetSSHConfig retrieves the SSH service configuration.
func (c *Client) GetSSHConfig(ctx context.Context) (*types.SSHConfig, error) {
	tflog.Trace(ctx, "GetSSHConfig start")

	resp, err := c.Get(ctx, "/ssh")
	if err != nil {
		return nil, fmt.Errorf("getting SSH config: %w", err)
	}

	var config types.SSHConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SSH config response: %w", err)
	}

	tflog.Trace(ctx, "GetSSHConfig success")
	return &config, nil
}

// UpdateSSHConfig updates the SSH service configuration.
func (c *Client) UpdateSSHConfig(ctx context.Context, req *types.SSHConfigUpdateRequest) (*types.SSHConfig, error) {
	tflog.Trace(ctx, "UpdateSSHConfig start")

	resp, err := c.Put(ctx, "/ssh", req)
	if err != nil {
		return nil, fmt.Errorf("updating SSH config: %w", err)
	}

	var config types.SSHConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SSH config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSSHConfig success")
	return &config, nil
}
