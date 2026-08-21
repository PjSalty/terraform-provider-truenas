package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for WebShare shares: sharing.webshare.{...}.
//
// The whole namespace is new in TrueNAS 26.0. On 25.10 and older it does
// not exist at all, so every call here answers method-not-found. That is
// translated into a message naming the required version rather than left
// as a bare -32601, which names only the method and points nowhere near
// the cause.

// errWebshareUnsupported turns a method-not-found on this namespace into an
// actionable diagnostic.
//
// Only method-not-found is translated. Any other failure keeps its own
// error, or a genuine permission or validation problem would be reported
// as a version problem.
func errWebshareUnsupported(err error, what string) error {
	if !isMethodUnknown(err) {
		return err
	}
	return fmt.Errorf("%s: WebShare requires TrueNAS 26.0 or newer. "+
		"This server has no sharing.webshare namespace, so the truenas_share_webshare "+
		"resource cannot be used against it (%w)", what, err)
}

// GetWebshare retrieves a WebShare share by ID.
func (c *Client) GetWebshare(ctx context.Context, id int) (*types.Webshare, error) {
	tflog.Trace(ctx, "GetWebshare (ws) start")

	result, err := c.Call(ctx, "sharing.webshare.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errWebshareUnsupported(err, fmt.Sprintf("getting WebShare share %d", id))
	}

	var share types.Webshare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing WebShare share response: %w", err)
	}

	tflog.Trace(ctx, "GetWebshare (ws) success")
	return &share, nil
}

// ListWebshares retrieves all WebShare shares.
func (c *Client) ListWebshares(ctx context.Context) ([]types.Webshare, error) {
	tflog.Trace(ctx, "ListWebshares (ws) start")

	result, err := c.Call(ctx, "sharing.webshare.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, errWebshareUnsupported(err, "listing WebShare shares")
	}

	var shares []types.Webshare
	if err := json.Unmarshal(result, &shares); err != nil {
		return nil, fmt.Errorf("parsing WebShare share list: %w", err)
	}

	tflog.Trace(ctx, "ListWebshares (ws) success")
	return shares, nil
}

// CreateWebshare creates a WebShare share.
func (c *Client) CreateWebshare(ctx context.Context, req *types.WebshareCreateRequest) (*types.Webshare, error) {
	tflog.Trace(ctx, "CreateWebshare (ws) start")

	result, err := c.Call(ctx, "sharing.webshare.create",
		[]interface{}{req}, CallOptions{Idempotent: false})
	if err != nil {
		return nil, errWebshareUnsupported(err, fmt.Sprintf("creating WebShare share %q", req.Name))
	}

	var share types.Webshare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing WebShare create response: %w", err)
	}

	tflog.Trace(ctx, "CreateWebshare (ws) success")
	return &share, nil
}

// UpdateWebshare updates a WebShare share.
func (c *Client) UpdateWebshare(ctx context.Context, id int, req *types.WebshareUpdateRequest) (*types.Webshare, error) {
	tflog.Trace(ctx, "UpdateWebshare (ws) start")

	result, err := c.Call(ctx, "sharing.webshare.update",
		[]interface{}{id, req}, CallOptions{Idempotent: false})
	if err != nil {
		return nil, errWebshareUnsupported(err, fmt.Sprintf("updating WebShare share %d", id))
	}

	var share types.Webshare
	if err := json.Unmarshal(result, &share); err != nil {
		return nil, fmt.Errorf("parsing WebShare update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateWebshare (ws) success")
	return &share, nil
}

// DeleteWebshare removes a WebShare share.
func (c *Client) DeleteWebshare(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteWebshare (ws) start")

	if _, err := c.Call(ctx, "sharing.webshare.delete",
		[]interface{}{id}, CallOptions{Idempotent: false}); err != nil {
		return errWebshareUnsupported(err, fmt.Sprintf("deleting WebShare share %d", id))
	}

	tflog.Trace(ctx, "DeleteWebshare (ws) success")
	return nil
}
