package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// SNMPConfig, SNMPConfigUpdateRequest moved to internal/types/snmp_config.go
// in the v2.0 transport-migration prep.
type (
	SNMPConfig              = types.SNMPConfig
	SNMPConfigUpdateRequest = types.SNMPConfigUpdateRequest
)

// GetSNMPConfig retrieves the SNMP configuration.
func (c *Client) GetSNMPConfig(ctx context.Context) (*types.SNMPConfig, error) {
	tflog.Trace(ctx, "GetSNMPConfig start")

	resp, err := c.Get(ctx, "/snmp")
	if err != nil {
		return nil, fmt.Errorf("getting SNMP config: %w", err)
	}

	var config types.SNMPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SNMP config response: %w", err)
	}

	tflog.Trace(ctx, "GetSNMPConfig success")
	return &config, nil
}

// UpdateSNMPConfig updates the SNMP configuration.
func (c *Client) UpdateSNMPConfig(ctx context.Context, req *types.SNMPConfigUpdateRequest) (*types.SNMPConfig, error) {
	tflog.Trace(ctx, "UpdateSNMPConfig start")

	resp, err := c.Put(ctx, "/snmp", req)
	if err != nil {
		return nil, fmt.Errorf("updating SNMP config: %w", err)
	}

	var config types.SNMPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SNMP config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSNMPConfig success")
	return &config, nil
}
