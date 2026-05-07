package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// KerberosRealm, KerberosRealmCreateRequest, KerberosRealmUpdateRequest
// moved to internal/types/kerberos.go in the v2.0 transport-migration
// prep.
type (
	KerberosRealm              = types.KerberosRealm
	KerberosRealmCreateRequest = types.KerberosRealmCreateRequest
	KerberosRealmUpdateRequest = types.KerberosRealmUpdateRequest
)

// ListKerberosRealms retrieves all kerberos realms.
func (c *Client) ListKerberosRealms(ctx context.Context) ([]KerberosRealm, error) {
	tflog.Trace(ctx, "ListKerberosRealms start")

	resp, err := c.Get(ctx, "/kerberos/realm")
	if err != nil {
		return nil, fmt.Errorf("listing kerberos realms: %w", err)
	}

	var items []KerberosRealm
	if err := json.Unmarshal(resp, &items); err != nil {
		return nil, fmt.Errorf("parsing kerberos realms list response: %w", err)
	}
	tflog.Trace(ctx, "ListKerberosRealms success")
	return items, nil
}

// GetKerberosRealm retrieves a kerberos realm by ID.
func (c *Client) GetKerberosRealm(ctx context.Context, id int) (*KerberosRealm, error) {
	tflog.Trace(ctx, "GetKerberosRealm start")

	resp, err := c.Get(ctx, fmt.Sprintf("/kerberos/realm/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting kerberos realm %d: %w", id, err)
	}

	var r KerberosRealm
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("parsing kerberos realm response: %w", err)
	}
	tflog.Trace(ctx, "GetKerberosRealm success")
	return &r, nil
}

// CreateKerberosRealm creates a new kerberos realm.
func (c *Client) CreateKerberosRealm(ctx context.Context, req *KerberosRealmCreateRequest) (*KerberosRealm, error) {
	tflog.Trace(ctx, "CreateKerberosRealm start")

	resp, err := c.Post(ctx, "/kerberos/realm", req)
	if err != nil {
		return nil, fmt.Errorf("creating kerberos realm: %w", err)
	}

	var r KerberosRealm
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("parsing kerberos realm create response: %w", err)
	}
	tflog.Trace(ctx, "CreateKerberosRealm success")
	return &r, nil
}

// UpdateKerberosRealm updates an existing kerberos realm.
func (c *Client) UpdateKerberosRealm(ctx context.Context, id int, req *KerberosRealmUpdateRequest) (*KerberosRealm, error) {
	tflog.Trace(ctx, "UpdateKerberosRealm start")

	resp, err := c.Put(ctx, fmt.Sprintf("/kerberos/realm/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating kerberos realm %d: %w", id, err)
	}

	var r KerberosRealm
	if err := json.Unmarshal(resp, &r); err != nil {
		return nil, fmt.Errorf("parsing kerberos realm update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateKerberosRealm success")
	return &r, nil
}

// DeleteKerberosRealm deletes a kerberos realm.
func (c *Client) DeleteKerberosRealm(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteKerberosRealm start")

	_, err := c.Delete(ctx, fmt.Sprintf("/kerberos/realm/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting kerberos realm %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteKerberosRealm success")
	return nil
}

// KerberosKeytab, KerberosKeytabCreateRequest, KerberosKeytab
// UpdateRequest moved to internal/types/kerberos.go.
type (
	KerberosKeytab              = types.KerberosKeytab
	KerberosKeytabCreateRequest = types.KerberosKeytabCreateRequest
	KerberosKeytabUpdateRequest = types.KerberosKeytabUpdateRequest
)

// GetKerberosKeytab retrieves a kerberos keytab by ID.
func (c *Client) GetKerberosKeytab(ctx context.Context, id int) (*KerberosKeytab, error) {
	tflog.Trace(ctx, "GetKerberosKeytab start")

	resp, err := c.Get(ctx, fmt.Sprintf("/kerberos/keytab/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting kerberos keytab %d: %w", id, err)
	}

	var k KerberosKeytab
	if err := json.Unmarshal(resp, &k); err != nil {
		return nil, fmt.Errorf("parsing kerberos keytab response: %w", err)
	}
	tflog.Trace(ctx, "GetKerberosKeytab success")
	return &k, nil
}

// CreateKerberosKeytab creates a new kerberos keytab entry.
func (c *Client) CreateKerberosKeytab(ctx context.Context, req *KerberosKeytabCreateRequest) (*KerberosKeytab, error) {
	tflog.Trace(ctx, "CreateKerberosKeytab start")

	resp, err := c.Post(ctx, "/kerberos/keytab", req)
	if err != nil {
		return nil, fmt.Errorf("creating kerberos keytab: %w", err)
	}

	var k KerberosKeytab
	if err := json.Unmarshal(resp, &k); err != nil {
		return nil, fmt.Errorf("parsing kerberos keytab create response: %w", err)
	}
	tflog.Trace(ctx, "CreateKerberosKeytab success")
	return &k, nil
}

// UpdateKerberosKeytab updates a kerberos keytab entry.
func (c *Client) UpdateKerberosKeytab(ctx context.Context, id int, req *KerberosKeytabUpdateRequest) (*KerberosKeytab, error) {
	tflog.Trace(ctx, "UpdateKerberosKeytab start")

	resp, err := c.Put(ctx, fmt.Sprintf("/kerberos/keytab/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating kerberos keytab %d: %w", id, err)
	}

	var k KerberosKeytab
	if err := json.Unmarshal(resp, &k); err != nil {
		return nil, fmt.Errorf("parsing kerberos keytab update response: %w", err)
	}
	tflog.Trace(ctx, "UpdateKerberosKeytab success")
	return &k, nil
}

// DeleteKerberosKeytab deletes a kerberos keytab entry.
func (c *Client) DeleteKerberosKeytab(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteKerberosKeytab start")

	_, err := c.Delete(ctx, fmt.Sprintf("/kerberos/keytab/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting kerberos keytab %d: %w", id, err)
	}
	tflog.Trace(ctx, "DeleteKerberosKeytab success")
	return nil
}
