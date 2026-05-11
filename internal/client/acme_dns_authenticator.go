package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- ACME DNS Authenticator API ---

// ACMEDNSAuthenticator represents an ACME DNS authenticator in TrueNAS.
type ACMEDNSAuthenticator struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ACMEDNSAuthenticatorCreateRequest represents the request body for creating an ACME DNS authenticator.
// The "authenticator" field (e.g., "cloudflare") is placed inside the attributes map.
type ACMEDNSAuthenticatorCreateRequest struct {
	Name       string                 `json:"name"`
	Attributes map[string]interface{} `json:"attributes"`
}

// ACMEDNSAuthenticatorUpdateRequest represents the request body for updating an ACME DNS authenticator.
type ACMEDNSAuthenticatorUpdateRequest struct {
	Name       string                 `json:"name,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// GetACMEDNSAuthenticator retrieves an ACME DNS authenticator by ID.
func (c *Client) GetACMEDNSAuthenticator(ctx context.Context, id int) (*ACMEDNSAuthenticator, error) {
	tflog.Trace(ctx, "GetACMEDNSAuthenticator start")

	resp, err := c.Get(ctx, fmt.Sprintf("/acme/dns/authenticator/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting ACME DNS authenticator %d: %w", id, err)
	}

	var auth ACMEDNSAuthenticator
	if err := json.Unmarshal(resp, &auth); err != nil {
		return nil, fmt.Errorf("parsing ACME DNS authenticator response: %w", err)
	}

	tflog.Trace(ctx, "GetACMEDNSAuthenticator success")
	return &auth, nil
}

// CreateACMEDNSAuthenticator creates a new ACME DNS authenticator.
func (c *Client) CreateACMEDNSAuthenticator(ctx context.Context, req *ACMEDNSAuthenticatorCreateRequest) (*ACMEDNSAuthenticator, error) {
	tflog.Trace(ctx, "CreateACMEDNSAuthenticator start")

	resp, err := c.Post(ctx, "/acme/dns/authenticator", req)
	if err != nil {
		return nil, fmt.Errorf("creating ACME DNS authenticator %q: %w", req.Name, err)
	}

	var auth ACMEDNSAuthenticator
	if err := json.Unmarshal(resp, &auth); err != nil {
		return nil, fmt.Errorf("parsing ACME DNS authenticator create response: %w", err)
	}

	tflog.Trace(ctx, "CreateACMEDNSAuthenticator success")
	return &auth, nil
}

// UpdateACMEDNSAuthenticator updates an existing ACME DNS authenticator.
func (c *Client) UpdateACMEDNSAuthenticator(ctx context.Context, id int, req *ACMEDNSAuthenticatorUpdateRequest) (*ACMEDNSAuthenticator, error) {
	tflog.Trace(ctx, "UpdateACMEDNSAuthenticator start")

	resp, err := c.Put(ctx, fmt.Sprintf("/acme/dns/authenticator/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating ACME DNS authenticator %d: %w", id, err)
	}

	var auth ACMEDNSAuthenticator
	if err := json.Unmarshal(resp, &auth); err != nil {
		return nil, fmt.Errorf("parsing ACME DNS authenticator update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateACMEDNSAuthenticator success")
	return &auth, nil
}

// DeleteACMEDNSAuthenticator deletes an ACME DNS authenticator.
func (c *Client) DeleteACMEDNSAuthenticator(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteACMEDNSAuthenticator start")

	_, err := c.Delete(ctx, fmt.Sprintf("/acme/dns/authenticator/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting ACME DNS authenticator %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteACMEDNSAuthenticator success")
	return nil
}
