package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for NFS service: nfs.{config,update}.

// GetNFSConfig retrieves the NFS service configuration.
func (c *Client) GetNFSConfig(ctx context.Context) (*types.NFSConfig, error) {
	tflog.Trace(ctx, "GetNFSConfig (ws) start")

	result, err := c.Call(ctx, "nfs.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting NFS config: %w", err)
	}

	var config types.NFSConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing NFS config response: %w", err)
	}

	tflog.Trace(ctx, "GetNFSConfig (ws) success")
	return &config, nil
}

// UpdateNFSConfig updates the NFS service configuration.
func (c *Client) UpdateNFSConfig(ctx context.Context, req *types.NFSConfigUpdateRequest) (*types.NFSConfig, error) {
	tflog.Trace(ctx, "UpdateNFSConfig (ws) start")

	result, err := c.Call(ctx, "nfs.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating NFS config: %w", err)
	}

	var config types.NFSConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing NFS config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNFSConfig (ws) success")
	return &config, nil
}
