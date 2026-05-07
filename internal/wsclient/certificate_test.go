package wsclient

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// =============================================================================
// ListCertificates
// =============================================================================

func TestListCertificates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "certificate.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{
				"id": 1, "name": "wildcard", "type": 8,
				"key_type": "RSA", "key_length": 2048,
			},
			map[string]interface{}{
				"id": 2, "name": "internal-ca", "type": 4,
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	certs, err := c.ListCertificates(ctx)
	if err != nil {
		t.Fatalf("ListCertificates: %v", err)
	}
	if len(certs) != 2 || certs[0].Name != "wildcard" || certs[1].Name != "internal-ca" {
		t.Errorf("got %+v", certs)
	}
}

func TestListCertificates_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListCertificates(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing certificates") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListCertificates_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListCertificates(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

// =============================================================================
// GetCertificate
// =============================================================================

func TestGetCertificate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "certificate.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 5, "name": "wildcard", "type": 8,
			"DN": "CN=*.example.com",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cert, err := c.GetCertificate(ctx, 5)
	if err != nil {
		t.Fatalf("GetCertificate: %v", err)
	}
	if cert.ID != 5 || cert.Name != "wildcard" || cert.DN != "CN=*.example.com" {
		t.Errorf("got %+v", cert)
	}
}

func TestGetCertificate_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCertificate(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting certificate") {
		t.Errorf("got %v", err)
	}
}

func TestGetCertificate_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-a-cert", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCertificate(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// GetCertificateByName
// =============================================================================

func TestGetCertificateByName_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "certificate.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		// Verify the filter param is the name=value triple.
		if len(params) != 1 {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "want 1 param"}
		}
		return []interface{}{map[string]interface{}{"id": 7, "name": "wildcard"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	cert, err := c.GetCertificateByName(ctx, "wildcard")
	if err != nil {
		t.Fatalf("GetCertificateByName: %v", err)
	}
	if cert.ID != 7 {
		t.Errorf("got %+v", cert)
	}
}

func TestGetCertificateByName_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCertificateByName(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found, got %v", err)
	}
}

func TestGetCertificateByName_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCertificateByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "listing certificates") {
		t.Errorf("got %v", err)
	}
}

func TestGetCertificateByName_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCertificateByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// CreateCertificate
// =============================================================================

// certificateJobServer fakes a certificate.{create,update,delete}
// response: returns a job ID immediately, then on core.get_jobs returns
// the job's terminal state with `result` set to the supplied object.
// jobError, when non-empty, marks the job FAILED with that message.
func certificateJobServer(t *testing.T, expectMethod string, jobResult interface{}, jobError string) *TestServer {
	t.Helper()
	pollCount := atomic.Int64{}
	const jobID = int64(42)
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case expectMethod:
			return jobID, nil
		case "core.get_jobs":
			pollCount.Add(1)
			state := "SUCCESS"
			if jobError != "" {
				state = "FAILED"
			}
			job := map[string]interface{}{
				"id":     jobID,
				"state":  state,
				"result": jobResult,
				"error":  jobError,
			}
			return []interface{}{job}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
}

func TestCreateCertificate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.create",
		map[string]interface{}{"id": 9, "name": "new-cert", "type": 8}, "")
	c, _ := ts.NewClient(ctx)
	cert, err := c.CreateCertificate(ctx, &types.CertificateCreateRequest{
		Name: "new-cert", CreateType: "CERTIFICATE_CREATE_IMPORTED",
	})
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	if cert.ID != 9 || cert.Name != "new-cert" {
		t.Errorf("got %+v", cert)
	}
}

func TestCreateCertificate_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.create", nil, "key generation failed")
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCertificate(ctx, &types.CertificateCreateRequest{Name: "bad"})
	if err == nil || !strings.Contains(err.Error(), "key generation failed") {
		t.Errorf("expected job failure, got %v", err)
	}
}

func TestCreateCertificate_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "missing required field"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCertificate(ctx, &types.CertificateCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating certificate") {
		t.Errorf("got %v", err)
	}
}

func TestCreateCertificate_resultDecodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.create", "not-a-cert-object", "")
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCertificate(ctx, &types.CertificateCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing certificate from job result") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// UpdateCertificate
// =============================================================================

func TestUpdateCertificate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.update",
		map[string]interface{}{"id": 9, "name": "renamed"}, "")
	c, _ := ts.NewClient(ctx)
	cert, err := c.UpdateCertificate(ctx, 9, &types.CertificateUpdateRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateCertificate: %v", err)
	}
	if cert.Name != "renamed" {
		t.Errorf("got %+v", cert)
	}
}

func TestUpdateCertificate_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.update", nil, "name conflict")
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCertificate(ctx, 9, &types.CertificateUpdateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "name conflict") {
		t.Errorf("expected job failure, got %v", err)
	}
}

func TestUpdateCertificate_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "no such id"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCertificate(ctx, 99, &types.CertificateUpdateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "updating certificate") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateCertificate_resultDecodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.update", "not-a-cert-object", "")
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCertificate(ctx, 9, &types.CertificateUpdateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing certificate from job result") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// DeleteCertificate
// =============================================================================

func TestDeleteCertificate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.delete", true, "")
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteCertificate(ctx, 9); err != nil {
		t.Errorf("DeleteCertificate: %v", err)
	}
}

func TestDeleteCertificate_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := certificateJobServer(t, "certificate.delete", nil, "filesystem busy")
	c, _ := ts.NewClient(ctx)
	err := c.DeleteCertificate(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "filesystem busy") {
		t.Errorf("expected job failure, got %v", err)
	}
}

func TestDeleteCertificate_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteCertificate(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "deleting certificate") {
		t.Errorf("got %v", err)
	}
}
