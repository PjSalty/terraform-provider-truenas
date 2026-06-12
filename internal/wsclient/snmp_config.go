package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for SNMP service: snmp.{config,update}.

// GetSNMPConfig retrieves the SNMP service configuration.
func (c *Client) GetSNMPConfig(ctx context.Context) (*types.SNMPConfig, error) {
	tflog.Trace(ctx, "GetSNMPConfig (ws) start")

	result, err := c.Call(ctx, "snmp.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting SNMP config: %w", err)
	}

	var config types.SNMPConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SNMP config response: %w", err)
	}

	tflog.Trace(ctx, "GetSNMPConfig (ws) success")
	return &config, nil
}

// UpdateSNMPConfig updates the SNMP service configuration.
func (c *Client) UpdateSNMPConfig(ctx context.Context, req *types.SNMPConfigUpdateRequest) (*types.SNMPConfig, error) {
	tflog.Trace(ctx, "UpdateSNMPConfig (ws) start")

	result, err := c.Call(ctx, "snmp.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating SNMP config: %w", err)
	}

	var config types.SNMPConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing SNMP config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSNMPConfig (ws) success")
	return &config, nil
}
