package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// SMBConfig represents the SMB service configuration.
type SMBConfig struct {
	ID             int    `json:"id"`
	NetbiosName    string `json:"netbiosname"`
	Workgroup      string `json:"workgroup"`
	Description    string `json:"description"`
	EnableSMB1     bool   `json:"enable_smb1"`
	UnixCharset    string `json:"unixcharset"`
	AAPLExtensions bool   `json:"aapl_extensions"`
	Guest          string `json:"guest"`
	Filemask       string `json:"filemask"`
	Dirmask        string `json:"dirmask"`
}

// SMBConfigUpdateRequest represents the request to update SMB configuration.
type SMBConfigUpdateRequest struct {
	NetbiosName    *string `json:"netbiosname,omitempty"`
	Workgroup      *string `json:"workgroup,omitempty"`
	Description    *string `json:"description,omitempty"`
	EnableSMB1     *bool   `json:"enable_smb1,omitempty"`
	UnixCharset    *string `json:"unixcharset,omitempty"`
	AAPLExtensions *bool   `json:"aapl_extensions,omitempty"`
	Guest          *string `json:"guest,omitempty"`
	Filemask       *string `json:"filemask,omitempty"`
	Dirmask        *string `json:"dirmask,omitempty"`
}

// GetSMBConfig retrieves the SMB service configuration.
func (c *Client) GetSMBConfig(ctx context.Context) (*SMBConfig, error) {
	tflog.Trace(ctx, "GetSMBConfig start")

	resp, err := c.Get(ctx, "/smb")
	if err != nil {
		return nil, fmt.Errorf("getting SMB config: %w", err)
	}

	var config SMBConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config response: %w", err)
	}

	tflog.Trace(ctx, "GetSMBConfig success")
	return &config, nil
}

// UpdateSMBConfig updates the SMB service configuration.
func (c *Client) UpdateSMBConfig(ctx context.Context, req *SMBConfigUpdateRequest) (*SMBConfig, error) {
	tflog.Trace(ctx, "UpdateSMBConfig start")

	resp, err := c.Put(ctx, "/smb", req)
	if err != nil {
		return nil, fmt.Errorf("updating SMB config: %w", err)
	}

	var config SMBConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing SMB config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateSMBConfig success")
	return &config, nil
}
