package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for FTP service: ftp.{config,update}.

// GetFTPConfig retrieves the FTP service configuration.
func (c *Client) GetFTPConfig(ctx context.Context) (*types.FTPConfig, error) {
	tflog.Trace(ctx, "GetFTPConfig (ws) start")

	result, err := c.Call(ctx, "ftp.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting FTP config: %w", err)
	}

	var config types.FTPConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing FTP config response: %w", err)
	}

	tflog.Trace(ctx, "GetFTPConfig (ws) success")
	return &config, nil
}

// UpdateFTPConfig updates the FTP service configuration.
func (c *Client) UpdateFTPConfig(ctx context.Context, req *types.FTPConfigUpdateRequest) (*types.FTPConfig, error) {
	tflog.Trace(ctx, "UpdateFTPConfig (ws) start")

	result, err := c.Call(ctx, "ftp.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating FTP config: %w", err)
	}

	var config types.FTPConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing FTP config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFTPConfig (ws) success")
	return &config, nil
}
