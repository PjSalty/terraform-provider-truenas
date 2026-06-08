package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// NFSConfig represents the NFS service configuration.
type NFSConfig struct {
	ID           int      `json:"id"`
	Servers      int      `json:"servers"`
	AllowNonroot bool     `json:"allow_nonroot"`
	Protocols    []string `json:"protocols"`
	V4Krb        bool     `json:"v4_krb"`
	V4Domain     string   `json:"v4_domain"`
	BindIP       []string `json:"bindip"`
	MountdPort   *int     `json:"mountd_port"`
	RpcstatdPort *int     `json:"rpcstatd_port"`
	RpclockdPort *int     `json:"rpclockd_port"`
}

// NFSConfigUpdateRequest represents the request to update NFS configuration.
type NFSConfigUpdateRequest struct {
	Servers      *int     `json:"servers,omitempty"`
	AllowNonroot *bool    `json:"allow_nonroot,omitempty"`
	Protocols    []string `json:"protocols,omitempty"`
	V4Krb        *bool    `json:"v4_krb,omitempty"`
	V4Domain     *string  `json:"v4_domain,omitempty"`
	BindIP       []string `json:"bindip,omitempty"`
	MountdPort   *int     `json:"mountd_port,omitempty"`
	RpcstatdPort *int     `json:"rpcstatd_port,omitempty"`
	RpclockdPort *int     `json:"rpclockd_port,omitempty"`
}

// GetNFSConfig retrieves the NFS service configuration.
func (c *Client) GetNFSConfig(ctx context.Context) (*NFSConfig, error) {
	tflog.Trace(ctx, "GetNFSConfig start")

	resp, err := c.Get(ctx, "/nfs")
	if err != nil {
		return nil, fmt.Errorf("getting NFS config: %w", err)
	}

	var config NFSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing NFS config response: %w", err)
	}

	tflog.Trace(ctx, "GetNFSConfig success")
	return &config, nil
}

// UpdateNFSConfig updates the NFS service configuration.
func (c *Client) UpdateNFSConfig(ctx context.Context, req *NFSConfigUpdateRequest) (*NFSConfig, error) {
	tflog.Trace(ctx, "UpdateNFSConfig start")

	resp, err := c.Put(ctx, "/nfs", req)
	if err != nil {
		return nil, fmt.Errorf("updating NFS config: %w", err)
	}

	var config NFSConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing NFS config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateNFSConfig success")
	return &config, nil
}
