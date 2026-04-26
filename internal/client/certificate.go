package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// --- Certificate API ---

// Certificate represents a TLS certificate in TrueNAS.
type Certificate struct {
	ID                 int      `json:"id"`
	Type               int      `json:"type"`
	Name               string   `json:"name"`
	CertificateData    string   `json:"certificate"`
	Privatekey         string   `json:"privatekey"`
	CSR                string   `json:"CSR"`
	KeyLength          int      `json:"key_length"`
	KeyType            string   `json:"key_type"`
	Country            string   `json:"country"`
	State              string   `json:"state"`
	City               string   `json:"city"`
	Organization       string   `json:"organization"`
	OrganizationalUnit string   `json:"organizational_unit"`
	Common             string   `json:"common"`
	Email              string   `json:"email"`
	DigestAlgorithm    string   `json:"digest_algorithm"`
	Lifetime           int      `json:"lifetime"`
	From               string   `json:"from"`
	Until              string   `json:"until"`
	Expired            bool     `json:"expired"`
	Parsed             bool     `json:"parsed"`
	DN                 string   `json:"DN"`
	SAN                []string `json:"san"`
}

// CertificateCreateRequest represents the request body for creating/importing a certificate.
type CertificateCreateRequest struct {
	Name               string   `json:"name"`
	CreateType         string   `json:"create_type"`
	CertificateData    string   `json:"certificate,omitempty"`
	Privatekey         string   `json:"privatekey,omitempty"`
	KeyType            string   `json:"key_type,omitempty"`
	KeyLength          int      `json:"key_length,omitempty"`
	DigestAlgorithm    string   `json:"digest_algorithm,omitempty"`
	Country            string   `json:"country,omitempty"`
	State              string   `json:"state,omitempty"`
	City               string   `json:"city,omitempty"`
	Organization       string   `json:"organization,omitempty"`
	OrganizationalUnit string   `json:"organizational_unit,omitempty"`
	Email              string   `json:"email,omitempty"`
	Common             string   `json:"common,omitempty"`
	SAN                []string `json:"san,omitempty"`
}

// CertificateUpdateRequest represents the request body for updating a certificate.
type CertificateUpdateRequest struct {
	Name string `json:"name,omitempty"`
}

// Job represents a TrueNAS async job.
type Job struct {
	ID       int             `json:"id"`
	Method   string          `json:"method"`
	State    string          `json:"state"`
	Error    string          `json:"error"`
	Result   json.RawMessage `json:"result"`
	Progress JobProgress     `json:"progress"`
}

// JobProgress represents job progress information.
type JobProgress struct {
	Percent     int    `json:"percent"`
	Description string `json:"description"`
}

// WaitForJob polls a job until it reaches a terminal state.
func (c *Client) WaitForJob(ctx context.Context, jobID int) (*Job, error) {
	tflog.Trace(ctx, "WaitForJob start")

	for {
		resp, err := c.Get(ctx, fmt.Sprintf("/core/get_jobs?id=%d", jobID))
		if err != nil {
			return nil, fmt.Errorf("polling job %d: %w", jobID, err)
		}

		var jobs []Job
		if err := json.Unmarshal(resp, &jobs); err != nil {
			return nil, fmt.Errorf("parsing job response: %w", err)
		}
		if len(jobs) == 0 {
			return nil, fmt.Errorf("job %d not found", jobID)
		}

		job := &jobs[0]
		switch job.State {
		case "SUCCESS":
			return job, nil
		case "FAILED", "ABORTED":
			return job, fmt.Errorf("job %d failed: %s", jobID, job.Error)
		}

		// Still running; wait before polling again.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

// ListCertificates retrieves all certificates.
func (c *Client) ListCertificates(ctx context.Context) ([]Certificate, error) {
	tflog.Trace(ctx, "ListCertificates start")

	resp, err := c.Get(ctx, "/certificate")
	if err != nil {
		return nil, fmt.Errorf("listing certificates: %w", err)
	}

	var certs []Certificate
	if err := json.Unmarshal(resp, &certs); err != nil {
		return nil, fmt.Errorf("parsing certificates list: %w", err)
	}

	tflog.Trace(ctx, "ListCertificates success")
	return certs, nil
}

// GetCertificate retrieves a certificate by ID.
func (c *Client) GetCertificate(ctx context.Context, id int) (*Certificate, error) {
	tflog.Trace(ctx, "GetCertificate start")

	resp, err := c.Get(ctx, fmt.Sprintf("/certificate/id/%d", id))
	if err != nil {
		return nil, fmt.Errorf("getting certificate %d: %w", id, err)
	}

	var cert Certificate
	if err := json.Unmarshal(resp, &cert); err != nil {
		return nil, fmt.Errorf("parsing certificate response: %w", err)
	}

	tflog.Trace(ctx, "GetCertificate success")
	return &cert, nil
}

// GetCertificateByName finds a certificate by name.
func (c *Client) GetCertificateByName(ctx context.Context, name string) (*Certificate, error) {
	tflog.Trace(ctx, "GetCertificateByName start")

	certs, err := c.ListCertificates(ctx)
	if err != nil {
		return nil, err
	}

	for _, cert := range certs {
		if cert.Name == name {
			return &cert, nil
		}
	}

	tflog.Trace(ctx, "GetCertificateByName success")
	return nil, &APIError{
		StatusCode: 404,
		Message:    fmt.Sprintf("certificate %q not found", name),
	}
}

// CreateCertificate creates or imports a certificate. Returns the created certificate.
// Certificate creation is async (returns a job ID).
func (c *Client) CreateCertificate(ctx context.Context, req *CertificateCreateRequest) (*Certificate, error) {
	tflog.Trace(ctx, "CreateCertificate start")

	resp, err := c.Post(ctx, "/certificate", req)
	if err != nil {
		return nil, fmt.Errorf("creating certificate %q: %w", req.Name, err)
	}

	// The API returns a job ID as a bare integer
	var jobID int
	if err := json.Unmarshal(resp, &jobID); err != nil {
		return nil, fmt.Errorf("parsing certificate create job ID: %w", err)
	}

	job, err := c.WaitForJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("waiting for certificate creation: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(job.Result, &cert); err != nil {
		return nil, fmt.Errorf("parsing certificate from job result: %w", err)
	}

	tflog.Trace(ctx, "CreateCertificate success")
	return &cert, nil
}

// UpdateCertificate updates an existing certificate.
func (c *Client) UpdateCertificate(ctx context.Context, id int, req *CertificateUpdateRequest) (*Certificate, error) {
	tflog.Trace(ctx, "UpdateCertificate start")

	resp, err := c.Put(ctx, fmt.Sprintf("/certificate/id/%d", id), req)
	if err != nil {
		return nil, fmt.Errorf("updating certificate %d: %w", id, err)
	}

	// Update also returns a job ID
	var jobID int
	if err := json.Unmarshal(resp, &jobID); err != nil {
		// Maybe it returned the cert directly; try that
		var cert Certificate
		if err2 := json.Unmarshal(resp, &cert); err2 != nil {
			return nil, fmt.Errorf("parsing certificate update response: job parse: %w, struct parse: %w", err, err2)
		}
		return &cert, nil
	}

	job, err := c.WaitForJob(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("waiting for certificate update: %w", err)
	}

	var cert Certificate
	if err := json.Unmarshal(job.Result, &cert); err != nil {
		return nil, fmt.Errorf("parsing certificate from job result: %w", err)
	}

	tflog.Trace(ctx, "UpdateCertificate success")
	return &cert, nil
}

// DeleteCertificate deletes a certificate.
func (c *Client) DeleteCertificate(ctx context.Context, id int) error {
	tflog.Trace(ctx, "DeleteCertificate start")

	resp, err := c.Delete(ctx, fmt.Sprintf("/certificate/id/%d", id))
	if err != nil {
		return fmt.Errorf("deleting certificate %d: %w", id, err)
	}

	if err := c.waitIfJobResponse(ctx, resp, fmt.Sprintf("delete certificate %d", id)); err != nil {
		return err
	}

	tflog.Trace(ctx, "DeleteCertificate success")
	return nil
}
