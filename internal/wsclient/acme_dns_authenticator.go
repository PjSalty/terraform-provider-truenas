package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for ACME DNS authenticators:
// acme.dns.authenticator.{get_instance, create, update, delete}.
//
// All operations are synchronous (the actual ACME challenge
// negotiation happens later as part of certificate.create).

// GetACMEDNSAuthenticator retrieves an ACME DNS authenticator by ID.
func (c *Client) GetACMEDNSAuthenticator(ctx context.Context, id int) (*types.ACMEDNSAuthenticator, error) {
	tflog.Trace(ctx, "GetACMEDNSAuthenticator (ws) start")

	result, err := c.Call(ctx, "acme.dns.authenticator.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting ACME DNS authenticator %d: %w", id, err)
	}

	var auth types.ACMEDNSAuthenticator
	if err := json.Unmarshal(result, &auth); err != nil {
		return nil, fmt.Errorf("parsing ACME DNS authenticator response: %w", err)
	}

	tflog.Trace(ctx, "GetACMEDNSAuthenticator (ws) success")
	return &auth, nil
}

// CreateACMEDNSAuthenticator creates a new ACME DNS authenticator.
func (c *Client) CreateACMEDNSAuthenticator(ctx context.Context, req *types.ACMEDNSAuthenticatorCreateRequest) (*types.ACMEDNSAuthenticator, error) {
	tflog.Trace(ctx, "CreateACMEDNSAuthenticator (ws) start")

	result, err := c.Call(ctx, "acme.dns.authenticator.create",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("creating ACME DNS authenticator %q: %w", req.Name, err)
	}

	var auth types.ACMEDNSAuthenticator
	if err := json.Unmarshal(result, &auth); err != nil {
		return nil, fmt.Errorf("parsing ACME DNS authenticator create response: %w", err)
	}

	tflog.Trace(ctx, "CreateACMEDNSAuthenticator (ws) success")
	return &auth, nil
}

// UpdateACMEDNSAuthenticator updates an existing ACME DNS authenticator.
func (c *Client) UpdateACMEDNSAuthenticator(ctx context.Context, id int, req *types.ACMEDNSAuthenticatorUpdateRequest) (*types.ACMEDNSAuthenticator, error) {
	tflog.Trace(ctx, "UpdateACMEDNSAuthenticator (ws) start")

	result, err := c.Call(ctx, "acme.dns.authenticator.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating ACME DNS authenticator %d: %w", id, err)
	}

	var auth types.ACMEDNSAuthenticator
	if err := json.Unmarshal(result, &auth); err != nil {
		return nil, fmt.Errorf("parsing ACME DNS authenticator update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateACMEDNSAuthenticator (ws) success")
	return &auth, nil
}

// DeleteACMEDNSAuthenticator deletes an ACME DNS authenticator.
func (c *Client) DeleteACMEDNSAuthenticator(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteACMEDNSAuthenticator (ws) start")

	if _, err := c.Call(ctx, "acme.dns.authenticator.delete",
		[]interface{}{id},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("deleting ACME DNS authenticator %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteACMEDNSAuthenticator (ws) success")
	return nil
}
