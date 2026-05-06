package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// FTPConfig, FTPConfigUpdateRequest moved to internal/types/ftp_config.go
// in the v2.0 transport-migration prep.
type (
	FTPConfig              = types.FTPConfig
	FTPConfigUpdateRequest = types.FTPConfigUpdateRequest
)

// GetFTPConfig retrieves the FTP service configuration.
func (c *Client) GetFTPConfig(ctx context.Context) (*types.FTPConfig, error) {
	tflog.Trace(ctx, "GetFTPConfig start")

	resp, err := c.Get(ctx, "/ftp")
	if err != nil {
		return nil, fmt.Errorf("getting FTP config: %w", err)
	}

	var config types.FTPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing FTP config response: %w", err)
	}

	tflog.Trace(ctx, "GetFTPConfig success")
	return &config, nil
}

// UpdateFTPConfig updates the FTP service configuration.
func (c *Client) UpdateFTPConfig(ctx context.Context, req *types.FTPConfigUpdateRequest) (*types.FTPConfig, error) {
	tflog.Trace(ctx, "UpdateFTPConfig start")

	resp, err := c.Put(ctx, "/ftp", req)
	if err != nil {
		return nil, fmt.Errorf("updating FTP config: %w", err)
	}

	var config types.FTPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing FTP config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFTPConfig success")
	return &config, nil
}
