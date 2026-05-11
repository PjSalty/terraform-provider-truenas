package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// KMIPConfig represents the singleton KMIP configuration.
type KMIPConfig struct {
	ID                   int     `json:"id"`
	Enabled              bool    `json:"enabled"`
	ManageSEDDisks       bool    `json:"manage_sed_disks"`
	ManageZFSKeys        bool    `json:"manage_zfs_keys"`
	Certificate          *int    `json:"certificate"`
	CertificateAuthority *int    `json:"certificate_authority"`
	Port                 int     `json:"port"`
	Server               *string `json:"server"`
	SSLVersion           string  `json:"ssl_version"`
}

// KMIPUpdateRequest is the singleton update payload.
type KMIPUpdateRequest struct {
	Enabled              *bool   `json:"enabled,omitempty"`
	ManageSEDDisks       *bool   `json:"manage_sed_disks,omitempty"`
	ManageZFSKeys        *bool   `json:"manage_zfs_keys,omitempty"`
	Certificate          *int    `json:"certificate,omitempty"`
	CertificateAuthority *int    `json:"certificate_authority,omitempty"`
	Port                 *int    `json:"port,omitempty"`
	Server               *string `json:"server,omitempty"`
	SSLVersion           *string `json:"ssl_version,omitempty"`
	ForceClear           *bool   `json:"force_clear,omitempty"`
	ChangeServer         *bool   `json:"change_server,omitempty"`
	Validate             *bool   `json:"validate,omitempty"`
}

// GetKMIPConfig retrieves the KMIP configuration.
func (c *Client) GetKMIPConfig(ctx context.Context) (*KMIPConfig, error) {
	tflog.Trace(ctx, "GetKMIPConfig start")

	resp, err := c.Get(ctx, "/kmip")
	if err != nil {
		return nil, fmt.Errorf("getting KMIP config: %w", err)
	}

	var cfg KMIPConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing KMIP config response: %w", err)
	}

	tflog.Trace(ctx, "GetKMIPConfig success")
	return &cfg, nil
}

// UpdateKMIPConfig updates the KMIP configuration via PUT.
func (c *Client) UpdateKMIPConfig(ctx context.Context, req *KMIPUpdateRequest) (*KMIPConfig, error) {
	tflog.Trace(ctx, "UpdateKMIPConfig start")

	resp, err := c.Put(ctx, "/kmip", req)
	if err != nil {
		return nil, fmt.Errorf("updating KMIP config: %w", err)
	}

	// TrueNAS may return either the updated object directly or a job ID
	// for async completion. Try decoding as the config first.
	var cfg KMIPConfig
	if err := json.Unmarshal(resp, &cfg); err == nil && cfg.ID != 0 {
		return &cfg, nil
	}

	// Re-fetch the canonical state.
	tflog.Trace(ctx, "UpdateKMIPConfig success")
	return c.GetKMIPConfig(ctx)
}
