package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// NFSConfig, NFSConfigUpdateRequest moved to internal/types/nfs_config.go
// in the v2.0 transport-migration prep.
type (
	NFSConfig              = types.NFSConfig
	NFSConfigUpdateRequest = types.NFSConfigUpdateRequest
)

// GetNFSConfig retrieves the NFS service configuration.
func (c *Client) GetNFSConfig(ctx context.Context) (*types.NFSConfig, error) {
	tflog.Trace(ctx, "GetNFSConfig start")

	resp, err := c.Get(ctx, "/nfs")
	if err != nil {
		return nil, fmt.Errorf("getting NFS config: %w", err)
	}

	var config types.NFSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing NFS config response: %w", err)
	}

	tflog.Trace(ctx, "GetNFSConfig success")
	return &config, nil
}

// UpdateNFSConfig updates the NFS service configuration.
func (c *Client) UpdateNFSConfig(ctx context.Context, req *types.NFSConfigUpdateRequest) (*types.NFSConfig, error) {
	tflog.Trace(ctx, "UpdateNFSConfig start")

	resp, err := c.Put(ctx, "/nfs", req)
	if err != nil {
		return nil, fmt.Errorf("updating NFS config: %w", err)
	}

	var config types.NFSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing NFS config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNFSConfig success")
	return &config, nil
}
