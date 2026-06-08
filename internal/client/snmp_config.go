package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SNMPConfig represents the SNMP configuration.
type SNMPConfig struct {
	ID               int     `json:"id"`
	Community        string  `json:"community"`
	Contact          string  `json:"contact"`
	Location         string  `json:"location"`
	V3               bool    `json:"v3"`
	V3Username       string  `json:"v3_username"`
	V3AuthType       string  `json:"v3_authtype"`
	V3Password       string  `json:"v3_password"`
	V3PrivProto      *string `json:"v3_privproto"`
	V3PrivPassphrase *string `json:"v3_privpassphrase"`
}

// SNMPConfigUpdateRequest represents the request to update SNMP configuration.
type SNMPConfigUpdateRequest struct {
	Community        *string `json:"community,omitempty"`
	Contact          *string `json:"contact,omitempty"`
	Location         *string `json:"location,omitempty"`
	V3               *bool   `json:"v3,omitempty"`
	V3Username       *string `json:"v3_username,omitempty"`
	V3AuthType       *string `json:"v3_authtype,omitempty"`
	V3Password       *string `json:"v3_password,omitempty"`
	V3PrivProto      *string `json:"v3_privproto,omitempty"`
	V3PrivPassphrase *string `json:"v3_privpassphrase,omitempty"`
}

// GetSNMPConfig retrieves the SNMP configuration.
func (c *Client) GetSNMPConfig(ctx context.Context) (*SNMPConfig, error) {
	tflog.Trace(ctx, "GetSNMPConfig start")

	resp, err := c.Get(ctx, "/snmp")
	if err != nil {
		return nil, fmt.Errorf("getting SNMP config: %w", err)
	}

	var config SNMPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SNMP config response: %w", err)
	}

	tflog.Trace(ctx, "GetSNMPConfig success")
	return &config, nil
}

// UpdateSNMPConfig updates the SNMP configuration.
func (c *Client) UpdateSNMPConfig(ctx context.Context, req *SNMPConfigUpdateRequest) (*SNMPConfig, error) {
	tflog.Trace(ctx, "UpdateSNMPConfig start")

	resp, err := c.Put(ctx, "/snmp", req)
	if err != nil {
		return nil, fmt.Errorf("updating SNMP config: %w", err)
	}

	var config SNMPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SNMP config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSNMPConfig success")
	return &config, nil
}
