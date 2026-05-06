package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// MailConfig, MailConfigUpdateRequest moved to internal/types/mail_config.go
// in the v2.0 transport-migration prep.
type (
	MailConfig              = types.MailConfig
	MailConfigUpdateRequest = types.MailConfigUpdateRequest
)

// GetMailConfig retrieves the mail configuration.
func (c *Client) GetMailConfig(ctx context.Context) (*types.MailConfig, error) {
	tflog.Trace(ctx, "GetMailConfig start")

	resp, err := c.Get(ctx, "/mail")
	if err != nil {
		return nil, fmt.Errorf("getting mail config: %w", err)
	}

	var config types.MailConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing mail config response: %w", err)
	}

	tflog.Trace(ctx, "GetMailConfig success")
	return &config, nil
}

// UpdateMailConfig updates the mail configuration.
func (c *Client) UpdateMailConfig(ctx context.Context, req *types.MailConfigUpdateRequest) (*types.MailConfig, error) {
	tflog.Trace(ctx, "UpdateMailConfig start")

	resp, err := c.Put(ctx, "/mail", req)
	if err != nil {
		return nil, fmt.Errorf("updating mail config: %w", err)
	}

	var config types.MailConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing mail config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateMailConfig success")
	return &config, nil
}
