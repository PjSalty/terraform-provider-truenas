package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespaces for Kerberos:
//   - kerberos.realm.{query, get_instance, create, update, delete}
//   - kerberos.keytab.{get_instance, create, update, delete}

// =============================================================================
// Realm
// =============================================================================

// ListKerberosRealms retrieves all kerberos realms.
func (c *Client) ListKerberosRealms(ctx context.Context) ([]types.KerberosRealm, error) {
	tflog.Trace(ctx, "ListKerberosRealms (ws) start")

	result, err := c.Call(ctx, "kerberos.realm.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing kerberos realms: %w", err)
	}

	var items []types.KerberosRealm
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("parsing kerberos realms list response: %w", err)
	}

	tflog.Trace(ctx, "ListKerberosRealms (ws) success")
	return items, nil
}

// GetKerberosRealm retrieves a kerberos realm by ID.
func (c *Client) GetKerberosRealm(ctx context.Context, id int) (*types.KerberosRealm, error) {
	tflog.Trace(ctx, "GetKerberosRealm (ws) start")

	result, err := c.Call(ctx, "kerberos.realm.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting kerberos realm %d: %w", id, err)
	}

	var realm types.KerberosRealm
	if err := json.Unmarshal(result, &realm); err != nil {
		return nil, fmt.Errorf("parsing kerberos realm response: %w", err)
	}

	tflog.Trace(ctx, "GetKerberosRealm (ws) success")
	return &realm, nil
}

// CreateKerberosRealm creates a new kerberos realm.
func (c *Client) CreateKerberosRealm(ctx context.Context, req *types.KerberosRealmCreateRequest) (*types.KerberosRealm, error) {
	tflog.Trace(ctx, "CreateKerberosRealm (ws) start")

	result, err := c.Call(ctx, "kerberos.realm.create",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("creating kerberos realm: %w", err)
	}

	var realm types.KerberosRealm
	if err := json.Unmarshal(result, &realm); err != nil {
		return nil, fmt.Errorf("parsing kerberos realm create response: %w", err)
	}

	tflog.Trace(ctx, "CreateKerberosRealm (ws) success")
	return &realm, nil
}

// UpdateKerberosRealm updates an existing kerberos realm.
func (c *Client) UpdateKerberosRealm(ctx context.Context, id int, req *types.KerberosRealmUpdateRequest) (*types.KerberosRealm, error) {
	tflog.Trace(ctx, "UpdateKerberosRealm (ws) start")

	result, err := c.Call(ctx, "kerberos.realm.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating kerberos realm %d: %w", id, err)
	}

	var realm types.KerberosRealm
	if err := json.Unmarshal(result, &realm); err != nil {
		return nil, fmt.Errorf("parsing kerberos realm update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateKerberosRealm (ws) success")
	return &realm, nil
}

// DeleteKerberosRealm deletes a kerberos realm.
func (c *Client) DeleteKerberosRealm(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteKerberosRealm (ws) start")

	if _, err := c.Call(ctx, "kerberos.realm.delete",
		[]interface{}{id},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("deleting kerberos realm %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteKerberosRealm (ws) success")
	return nil
}

// =============================================================================
// Keytab
// =============================================================================

// GetKerberosKeytab retrieves a kerberos keytab entry by ID.
func (c *Client) GetKerberosKeytab(ctx context.Context, id int) (*types.KerberosKeytab, error) {
	tflog.Trace(ctx, "GetKerberosKeytab (ws) start")

	result, err := c.Call(ctx, "kerberos.keytab.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting kerberos keytab %d: %w", id, err)
	}

	var k types.KerberosKeytab
	if err := json.Unmarshal(result, &k); err != nil {
		return nil, fmt.Errorf("parsing kerberos keytab response: %w", err)
	}

	tflog.Trace(ctx, "GetKerberosKeytab (ws) success")
	return &k, nil
}

// CreateKerberosKeytab creates a kerberos keytab entry.
func (c *Client) CreateKerberosKeytab(ctx context.Context, req *types.KerberosKeytabCreateRequest) (*types.KerberosKeytab, error) {
	tflog.Trace(ctx, "CreateKerberosKeytab (ws) start")

	result, err := c.Call(ctx, "kerberos.keytab.create",
		[]interface{}{req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("creating kerberos keytab: %w", err)
	}

	var k types.KerberosKeytab
	if err := json.Unmarshal(result, &k); err != nil {
		return nil, fmt.Errorf("parsing kerberos keytab create response: %w", err)
	}

	tflog.Trace(ctx, "CreateKerberosKeytab (ws) success")
	return &k, nil
}

// UpdateKerberosKeytab updates a kerberos keytab entry.
func (c *Client) UpdateKerberosKeytab(ctx context.Context, id int, req *types.KerberosKeytabUpdateRequest) (*types.KerberosKeytab, error) {
	tflog.Trace(ctx, "UpdateKerberosKeytab (ws) start")

	result, err := c.Call(ctx, "kerberos.keytab.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false})
	if err != nil {
		return nil, fmt.Errorf("updating kerberos keytab %d: %w", id, err)
	}

	var k types.KerberosKeytab
	if err := json.Unmarshal(result, &k); err != nil {
		return nil, fmt.Errorf("parsing kerberos keytab update response: %w", err)
	}

	tflog.Trace(ctx, "UpdateKerberosKeytab (ws) success")
	return &k, nil
}

// DeleteKerberosKeytab deletes a kerberos keytab entry.
func (c *Client) DeleteKerberosKeytab(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteKerberosKeytab (ws) start")

	if _, err := c.Call(ctx, "kerberos.keytab.delete",
		[]interface{}{id},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("deleting kerberos keytab %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteKerberosKeytab (ws) success")
	return nil
}
