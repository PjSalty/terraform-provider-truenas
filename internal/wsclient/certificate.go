package wsclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for certificates: certificate.{...}.
//
// certificate.create / certificate.update / certificate.delete are
// asynchronous — the server returns a job ID and the actual work
// (key generation, ACME flow, file removal) happens in the background.
// CallJob handles the core.get_jobs polling.
//
// certPollInterval bounds the wait between core.get_jobs polls during
// async certificate operations. ACME flows can run for many seconds;
// 1s is conservative and matches the REST WaitForJob cadence.
const certPollInterval = 1 * time.Second

// ListCertificates retrieves all certificates.
func (c *Client) ListCertificates(ctx context.Context) ([]types.Certificate, error) {
	tflog.Trace(ctx, "ListCertificates (ws) start")

	result, err := c.Call(ctx, "certificate.query", nil, CallOptions{
		Read:       true,
		Idempotent: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing certificates: %w", err)
	}

	var certs []types.Certificate
	if err := json.Unmarshal(result, &certs); err != nil {
		return nil, fmt.Errorf("parsing certificates list: %w", err)
	}

	tflog.Trace(ctx, "ListCertificates (ws) success")
	return certs, nil
}

// GetCertificate retrieves a certificate by ID.
func (c *Client) GetCertificate(ctx context.Context, id int) (*types.Certificate, error) {
	tflog.Trace(ctx, "GetCertificate (ws) start")

	result, err := c.Call(ctx, "certificate.get_instance",
		[]interface{}{id}, CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting certificate %d: %w", id, err)
	}

	var cert types.Certificate
	if err := json.Unmarshal(result, &cert); err != nil {
		return nil, fmt.Errorf("parsing certificate response: %w", err)
	}

	tflog.Trace(ctx, "GetCertificate (ws) success")
	return &cert, nil
}

// GetCertificateByName finds a certificate by name. JSON-RPC
// certificate.query supports server-side filtering, so we issue a
// scoped query rather than fetching the whole list and filtering
// client-side (which the REST client has to do).
func (c *Client) GetCertificateByName(ctx context.Context, name string) (*types.Certificate, error) {
	tflog.Trace(ctx, "GetCertificateByName (ws) start")

	filters := []interface{}{[]interface{}{"name", "=", name}}
	result, err := c.Call(ctx, "certificate.query",
		[]interface{}{filters},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing certificates: %w", err)
	}

	var certs []types.Certificate
	if err := json.Unmarshal(result, &certs); err != nil {
		return nil, fmt.Errorf("parsing certificates list: %w", err)
	}

	if len(certs) == 0 {
		return nil, &RPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("certificate %q not found", name),
		}
	}

	tflog.Trace(ctx, "GetCertificateByName (ws) success")
	return &certs[0], nil
}

// CreateCertificate creates or imports a certificate. The server runs
// the work as a job; CallJob polls until terminal state and returns
// the created Certificate from the job's result field.
func (c *Client) CreateCertificate(ctx context.Context, req *types.CertificateCreateRequest) (*types.Certificate, error) {
	tflog.Trace(ctx, "CreateCertificate (ws) start")

	jobResult, err := c.CallJob(ctx, "certificate.create",
		[]interface{}{req},
		CallOptions{Job: true, Idempotent: false},
		certPollInterval)
	if err != nil {
		return nil, fmt.Errorf("creating certificate %q: %w", req.Name, err)
	}

	var cert types.Certificate
	if err := json.Unmarshal(jobResult, &cert); err != nil {
		return nil, fmt.Errorf("parsing certificate from job result: %w", err)
	}

	tflog.Trace(ctx, "CreateCertificate (ws) success")
	return &cert, nil
}

// UpdateCertificate updates an existing certificate. Like create,
// update is async on the server side.
func (c *Client) UpdateCertificate(ctx context.Context, id int, req *types.CertificateUpdateRequest) (*types.Certificate, error) {
	tflog.Trace(ctx, "UpdateCertificate (ws) start")

	jobResult, err := c.CallJob(ctx, "certificate.update",
		[]interface{}{id, req},
		CallOptions{Job: true, Idempotent: false},
		certPollInterval)
	if err != nil {
		return nil, fmt.Errorf("updating certificate %d: %w", id, err)
	}

	var cert types.Certificate
	if err := json.Unmarshal(jobResult, &cert); err != nil {
		return nil, fmt.Errorf("parsing certificate from job result: %w", err)
	}

	tflog.Trace(ctx, "UpdateCertificate (ws) success")
	return &cert, nil
}

// DeleteCertificate deletes a certificate. Async on the server (file
// cleanup + DB row removal); CallJob waits for terminal state.
func (c *Client) DeleteCertificate(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCertificate (ws) start")

	_, err := c.CallJob(ctx, "certificate.delete",
		[]interface{}{id},
		CallOptions{Job: true, Idempotent: false},
		certPollInterval)
	if err != nil {
		return fmt.Errorf("deleting certificate %d: %w", id, err)
	}

	tflog.Trace(ctx, "DeleteCertificate (ws) success")
	return nil
}
