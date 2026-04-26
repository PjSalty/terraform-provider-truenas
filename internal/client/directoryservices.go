package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// DirectoryServicesConfig represents the directory services singleton config.
//
// The API uses polymorphic credential and configuration objects whose shape
// depends on service_type (ACTIVEDIRECTORY|IPA|LDAP) and credential_type. We
// capture these as raw JSON maps so the terraform resource can round-trip
// them without us having to model every variant.
type DirectoryServicesConfig struct {
	ID                 int                    `json:"id"`
	ServiceType        *string                `json:"service_type,omitempty"`
	Credential         map[string]interface{} `json:"credential,omitempty"`
	Enable             bool                   `json:"enable"`
	EnableAccountCache bool                   `json:"enable_account_cache"`
	EnableDNSUpdates   bool                   `json:"enable_dns_updates"`
	Timeout            int                    `json:"timeout"`
	KerberosRealm      *string                `json:"kerberos_realm,omitempty"`
	Configuration      map[string]interface{} `json:"configuration,omitempty"`
}

// DirectoryServicesUpdateRequest is the body for PUT /directoryservices.
//
// Any field left as nil is omitted — matching the singleton-patch pattern
// used by other config resources in this provider.
type DirectoryServicesUpdateRequest struct {
	ServiceType        *string                 `json:"service_type,omitempty"`
	Credential         *map[string]interface{} `json:"credential,omitempty"`
	Enable             *bool                   `json:"enable,omitempty"`
	EnableAccountCache *bool                   `json:"enable_account_cache,omitempty"`
	EnableDNSUpdates   *bool                   `json:"enable_dns_updates,omitempty"`
	Timeout            *int                    `json:"timeout,omitempty"`
	KerberosRealm      *string                 `json:"kerberos_realm,omitempty"`
	Configuration      *map[string]interface{} `json:"configuration,omitempty"`
	Force              *bool                   `json:"force,omitempty"`
}

// GetDirectoryServicesConfig retrieves the directory services config.
func (c *Client) GetDirectoryServicesConfig(ctx context.Context) (*DirectoryServicesConfig, error) {
	tflog.Trace(ctx, "GetDirectoryServicesConfig start")

	resp, err := c.Get(ctx, "/directoryservices")
	if err != nil {
		return nil, fmt.Errorf("getting directory services config: %w", err)
	}

	var cfg DirectoryServicesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		return nil, fmt.Errorf("parsing directory services response: %w", err)
	}
	tflog.Trace(ctx, "GetDirectoryServicesConfig success")
	return &cfg, nil
}

// UpdateDirectoryServicesConfig updates the directory services singleton.
func (c *Client) UpdateDirectoryServicesConfig(ctx context.Context, req *DirectoryServicesUpdateRequest) (*DirectoryServicesConfig, error) {
	tflog.Trace(ctx, "UpdateDirectoryServicesConfig start")

	resp, err := c.Put(ctx, "/directoryservices", req)
	if err != nil {
		return nil, fmt.Errorf("updating directory services config: %w", err)
	}

	// PUT may return a job ID (int) for AD join, or the config object.
	// If it's a number we re-fetch the config.
	var asInt int
	if err := json.Unmarshal(resp, &asInt); err == nil {
		return c.GetDirectoryServicesConfig(ctx)
	}

	var cfg DirectoryServicesConfig
	if err := json.Unmarshal(resp, &cfg); err != nil {
		// Fallback: re-fetch if we can't decode (polymorphic shape may confuse us).
		return c.GetDirectoryServicesConfig(ctx)
	}
	tflog.Trace(ctx, "UpdateDirectoryServicesConfig success")
	return &cfg, nil
}

// LeaveDirectoryServices instructs TrueNAS to leave the currently-joined
// directory service (used during AD disable flows). Returns the raw job
// response. Errors from the API are propagated unchanged.
func (c *Client) LeaveDirectoryServices(ctx context.Context, body map[string]interface{}) ([]byte, error) {
	tflog.Trace(ctx, "LeaveDirectoryServices start")

	resp, err := c.Post(ctx, "/directoryservices/leave", body)
	if err != nil {
		return nil, fmt.Errorf("leaving directory services: %w", err)
	}
	tflog.Trace(ctx, "LeaveDirectoryServices success")
	return resp, nil
}
