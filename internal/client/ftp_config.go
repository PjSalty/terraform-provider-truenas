package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// FTPConfig represents the FTP service configuration.
type FTPConfig struct {
	ID            int    `json:"id"`
	Port          int    `json:"port"`
	Clients       int    `json:"clients"`
	IPConnections int    `json:"ipconnections"`
	LoginAttempt  int    `json:"loginattempt"`
	Timeout       int    `json:"timeout"`
	OnlyAnonymous bool   `json:"onlyanonymous"`
	OnlyLocal     bool   `json:"onlylocal"`
	Banner        string `json:"banner"`
	Filemask      string `json:"filemask"`
	Dirmask       string `json:"dirmask"`
	FXP           bool   `json:"fxp"`
	Resume        bool   `json:"resume"`
	DefaultRoot   bool   `json:"defaultroot"`
	TLS           bool   `json:"tls"`
}

// FTPConfigUpdateRequest represents the request to update FTP configuration.
type FTPConfigUpdateRequest struct {
	Port          *int    `json:"port,omitempty"`
	Clients       *int    `json:"clients,omitempty"`
	IPConnections *int    `json:"ipconnections,omitempty"`
	LoginAttempt  *int    `json:"loginattempt,omitempty"`
	Timeout       *int    `json:"timeout,omitempty"`
	OnlyAnonymous *bool   `json:"onlyanonymous,omitempty"`
	OnlyLocal     *bool   `json:"onlylocal,omitempty"`
	Banner        *string `json:"banner,omitempty"`
	Filemask      *string `json:"filemask,omitempty"`
	Dirmask       *string `json:"dirmask,omitempty"`
	FXP           *bool   `json:"fxp,omitempty"`
	Resume        *bool   `json:"resume,omitempty"`
	DefaultRoot   *bool   `json:"defaultroot,omitempty"`
	TLS           *bool   `json:"tls,omitempty"`
}

// GetFTPConfig retrieves the FTP service configuration.
func (c *Client) GetFTPConfig(ctx context.Context) (*FTPConfig, error) {
	tflog.Trace(ctx, "GetFTPConfig start")

	resp, err := c.Get(ctx, "/ftp")
	if err != nil {
		return nil, fmt.Errorf("getting FTP config: %w", err)
	}

	var config FTPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing FTP config response: %w", err)
	}

	tflog.Trace(ctx, "GetFTPConfig success")
	return &config, nil
}

// UpdateFTPConfig updates the FTP service configuration.
func (c *Client) UpdateFTPConfig(ctx context.Context, req *FTPConfigUpdateRequest) (*FTPConfig, error) {
	tflog.Trace(ctx, "UpdateFTPConfig start")

	resp, err := c.Put(ctx, "/ftp", req)
	if err != nil {
		return nil, fmt.Errorf("updating FTP config: %w", err)
	}

	var config FTPConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing FTP config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateFTPConfig success")
	return &config, nil
}
