package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC methods for the LXC container configuration: lxc.{config,update}.
//
// The lxc namespace is new in TrueNAS 26.0 and absent on 25.10 and older,
// so a method-not-found here means "this server is too old" rather than a
// typo. It is translated for the same reason as sharing.webshare: a bare
// -32601 names only the method.

func errLXCUnsupported(err error, what string) error {
	if !isMethodUnknown(err) {
		return err
	}
	return fmt.Errorf("%s: LXC containers require TrueNAS 26.0 or newer. "+
		"This server has no lxc namespace, so truenas_lxc_config cannot be used "+
		"against it (%w)", what, err)
}

// GetLXCConfig retrieves the LXC configuration singleton.
func (c *Client) GetLXCConfig(ctx context.Context) (*types.LXCConfig, error) {
	tflog.Trace(ctx, "GetLXCConfig (ws) start")

	result, err := c.Call(ctx, "lxc.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, errLXCUnsupported(err, "getting LXC config")
	}

	var cfg types.LXCConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing LXC config response: %w", err)
	}

	tflog.Trace(ctx, "GetLXCConfig (ws) success")
	return &cfg, nil
}

// SetLXCConfig applies the LXC configuration.
//
// lxc.update takes a single positional object (the upstream args model is
// @single_argument_args) and returns the full entry.
func (c *Client) SetLXCConfig(ctx context.Context, req *types.LXCConfigUpdateRequest) (*types.LXCConfig, error) {
	tflog.Trace(ctx, "SetLXCConfig (ws) start")

	result, err := c.Call(ctx, "lxc.update",
		[]interface{}{req}, CallOptions{Idempotent: false})
	if err != nil {
		return nil, errLXCUnsupported(err, "updating LXC config")
	}

	var cfg types.LXCConfig
	if err := json.Unmarshal(result, &cfg); err != nil {
		return nil, fmt.Errorf("parsing LXC config update response: %w", err)
	}

	tflog.Trace(ctx, "SetLXCConfig (ws) success")
	return &cfg, nil
}

// GetLXCBridgeChoices lists the network bridges available for container
// networking, keyed by interface name.
func (c *Client) GetLXCBridgeChoices(ctx context.Context) (map[string]string, error) {
	tflog.Trace(ctx, "GetLXCBridgeChoices (ws) start")

	result, err := c.Call(ctx, "lxc.bridge_choices", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, errLXCUnsupported(err, "listing LXC bridge choices")
	}

	var choices map[string]string
	if err := json.Unmarshal(result, &choices); err != nil {
		return nil, fmt.Errorf("parsing LXC bridge choices response: %w", err)
	}

	tflog.Trace(ctx, "GetLXCBridgeChoices (ws) success")
	return choices, nil
}
