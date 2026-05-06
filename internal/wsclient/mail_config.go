package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for mail service: mail.{config,update}.

// GetMailConfig retrieves the mail configuration.
func (c *Client) GetMailConfig(ctx context.Context) (*types.MailConfig, error) {
	tflog.Trace(ctx, "GetMailConfig (ws) start")

	result, err := c.Call(ctx, "mail.config", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("getting mail config: %w", err)
	}

	var config types.MailConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing mail config response: %w", err)
	}

	tflog.Trace(ctx, "GetMailConfig (ws) success")
	return &config, nil
}

// UpdateMailConfig updates the mail configuration.
func (c *Client) UpdateMailConfig(ctx context.Context, req *types.MailConfigUpdateRequest) (*types.MailConfig, error) {
	tflog.Trace(ctx, "UpdateMailConfig (ws) start")

	result, err := c.Call(ctx, "mail.update",
		[]interface{}{req}, CallOptions{})
	if err != nil {
		return nil, fmt.Errorf("updating mail config: %w", err)
	}

	var config types.MailConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return nil, fmt.Errorf("parsing mail config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateMailConfig (ws) success")
	return &config, nil
}
