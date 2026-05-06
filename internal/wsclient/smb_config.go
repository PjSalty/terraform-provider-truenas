package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for SMB service: smb.{config,update}.

// GetSMBConfig retrieves the SMB service configuration.
func (c *Client) GetSMBConfig(ctx context.Context) (*types.SMBConfig, error) {
	tflog.Trace(ctx, "GetSMBConfig (ws) start")

	result, err := c.Call(ctx, "smb.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting SMB config: %w", err)
	}

	var config types.SMBConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config response: %w", err)
	}

	tflog.Trace(ctx, "GetSMBConfig (ws) success")
	return &config, nil
}

// UpdateSMBConfig updates the SMB service configuration.
func (c *Client) UpdateSMBConfig(ctx context.Context, req *types.SMBConfigUpdateRequest) (*types.SMBConfig, error) {
	tflog.Trace(ctx, "UpdateSMBConfig (ws) start")

	result, err := c.Call(ctx, "smb.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}

	var config types.SMBConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSMBConfig (ws) success")
	return &config, nil
}
