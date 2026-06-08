package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// MailConfig represents the mail/SMTP configuration.
type MailConfig struct {
	ID             int     `json:"id"`
	FromEmail      string  `json:"fromemail"`
	FromName       string  `json:"fromname"`
	OutgoingServer string  `json:"outgoingserver"`
	Port           int     `json:"port"`
	Security       string  `json:"security"`
	SMTP           bool    `json:"smtp"`
	User           *string `json:"user"`
	Pass           string  `json:"pass"`
}

// MailConfigUpdateRequest represents the request to update mail configuration.
type MailConfigUpdateRequest struct {
	FromEmail      *string `json:"fromemail,omitempty"`
	FromName       *string `json:"fromname,omitempty"`
	OutgoingServer *string `json:"outgoingserver,omitempty"`
	Port           *int    `json:"port,omitempty"`
	Security       *string `json:"security,omitempty"`
	SMTP           *bool   `json:"smtp,omitempty"`
	User           *string `json:"user,omitempty"`
	Pass           *string `json:"pass,omitempty"`
}

// GetMailConfig retrieves the mail configuration.
func (c *Client) GetMailConfig(ctx context.Context) (*MailConfig, error) {
	tflog.Trace(ctx, "GetMailConfig start")

	resp, err := c.Get(ctx, "/mail")
	if err != nil {
		return nil, fmt.Errorf("getting mail config: %w", err)
	}

	var config MailConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing mail config response: %w", err)
	}

	tflog.Trace(ctx, "GetMailConfig success")
	return &config, nil
}

// UpdateMailConfig updates the mail configuration.
func (c *Client) UpdateMailConfig(ctx context.Context, req *MailConfigUpdateRequest) (*MailConfig, error) {
	tflog.Trace(ctx, "UpdateMailConfig start")

	resp, err := c.Put(ctx, "/mail", req)
	if err != nil {
		return nil, fmt.Errorf("updating mail config: %w", err)
	}

	var config MailConfig
	if err := json.Unmarshal(resp, &config); err != nil {
		return nil, fmt.Errorf("parsing mail config update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateMailConfig success")
	return &config, nil
}
