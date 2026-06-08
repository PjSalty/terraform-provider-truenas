package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SSHConfig represents the SSH service configuration.
type SSHConfig struct {
	ID              int      `json:"id"`
	TCPPort         int      `json:"tcpport"`
	PasswordAuth    bool     `json:"passwordauth"`
	KerberosAuth    bool     `json:"kerberosauth"`
	TCPFwd          bool     `json:"tcpfwd"`
	Compression     bool     `json:"compression"`
	SFTPLogLevel    string   `json:"sftp_log_level"`
	SFTPLogFacility string   `json:"sftp_log_facility"`
	WeakCiphers     []string `json:"weak_ciphers"`
}

// SSHConfigUpdateRequest represents the request to update SSH configuration.
type SSHConfigUpdateRequest struct {
	TCPPort         *int      `json:"tcpport,omitempty"`
	PasswordAuth    *bool     `json:"passwordauth,omitempty"`
	KerberosAuth    *bool     `json:"kerberosauth,omitempty"`
	TCPFwd          *bool     `json:"tcpfwd,omitempty"`
	Compression     *bool     `json:"compression,omitempty"`
	SFTPLogLevel    *string   `json:"sftp_log_level,omitempty"`
	SFTPLogFacility *string   `json:"sftp_log_facility,omitempty"`
	WeakCiphers     *[]string `json:"weak_ciphers,omitempty"`
}

// GetSSHConfig retrieves the SSH service configuration.
func (c *Client) GetSSHConfig(ctx context.Context) (*SSHConfig, error) {
	tflog.Trace(ctx, "GetSSHConfig start")

	resp, err := c.Get(ctx, "/ssh")
	if err != nil {
		return nil, fmt.Errorf("getting SSH config: %w", err)
	}

	var config SSHConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SSH config response: %w", err)
	}

	tflog.Trace(ctx, "GetSSHConfig success")
	return &config, nil
}

// UpdateSSHConfig updates the SSH service configuration.
func (c *Client) UpdateSSHConfig(ctx context.Context, req *SSHConfigUpdateRequest) (*SSHConfig, error) {
	tflog.Trace(ctx, "UpdateSSHConfig start")

	resp, err := c.Put(ctx, "/ssh", req)
	if err != nil {
		return nil, fmt.Errorf("updating SSH config: %w", err)
	}

	var config SSHConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SSH config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSSHConfig success")
	return &config, nil
}
