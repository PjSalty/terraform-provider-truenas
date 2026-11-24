package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for the iSCSI global configuration:
// iscsi.global.{config,update}.

// GetISCSIGlobal retrieves the iSCSI global configuration singleton.
func (c *Client) GetISCSIGlobal(ctx context.Context) (*types.ISCSIGlobal, error) {
	tflog.Trace(ctx, "GetISCSIGlobal (ws) start")

	result, err := c.Call(ctx, "iscsi.global.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting iSCSI global config: %w", err)
	}

	var cfg types.ISCSIGlobal
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing iSCSI global config response: %w", err)
	}

	tflog.Trace(ctx, "GetISCSIGlobal (ws) success")
	return &cfg, nil
}

// UpdateISCSIGlobal applies the iSCSI global configuration and returns the
// stored entry. TrueNAS first connects to every iSNS server in the request,
// then reloads the iSCSI service without waiting for it; a change of iSER
// restarts the service if it is running.
func (c *Client) UpdateISCSIGlobal(ctx context.Context, req *types.ISCSIGlobalUpdateRequest) (*types.ISCSIGlobal, error) {
	tflog.Trace(ctx, "UpdateISCSIGlobal (ws) start")

	result, err := c.Call(ctx, "iscsi.global.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating iSCSI global config: %w", err)
	}

	var cfg types.ISCSIGlobal
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing iSCSI global config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateISCSIGlobal (ws) success")
	return &cfg, nil
}
