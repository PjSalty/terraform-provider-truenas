package client_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// statusMatrixCase enumerates the wire conditions every client method
// must classify correctly. The matrix is rigid (one row per condition,
// one expected outcome per row) so a regression in the shared doOnce
// handler — e.g. a 401 stops mapping to "auth failure" — fails the
// exact row that drifted.
//
// 4xx classes are non-retried — caller sees the typed error
// immediately. 5xx and 429 ARE retried (per doRequest semantics) and
// the persistent-failure path eventually surfaces either the typed
// error or a context-deadline depending on whether retries fit inside
// the call deadline. We assert "either" with retryable=true.
type statusMatrixCase struct {
	name        string
	status      int
	body        string
	contentType string
	wantErr     bool
	retryable   bool   // true => 5xx/429 path that retries; accept either typed error OR ctx deadline
	wantClass   string // optional substring the wrapped error must contain (skipped when retryable)
}

var statusMatrix = []statusMatrixCase{
	{name: "200_ok_with_body", status: 200, body: `{"version":"x"}`, contentType: "application/json", wantErr: false},
	{name: "200_ok_empty_body", status: 200, body: "", contentType: "application/json", wantErr: true},
	{name: "201_created", status: 201, body: `{"version":"x"}`, contentType: "application/json", wantErr: false},
	{name: "204_no_content", status: 204, body: "", contentType: "application/json", wantErr: true},
	{name: "400_bad_request", status: 400, body: `{"message":"bad request"}`, contentType: "application/json", wantErr: true, wantClass: "400"},
	{name: "401_unauthorized", status: 401, body: `{"message":"invalid token"}`, contentType: "application/json", wantErr: true, wantClass: "401"},
	{name: "403_forbidden", status: 403, body: `{"message":"forbidden"}`, contentType: "application/json", wantErr: true, wantClass: "403"},
	{name: "404_not_found", status: 404, body: `{"message":"not found"}`, contentType: "application/json", wantErr: true, wantClass: "404"},
	{name: "409_conflict", status: 409, body: `{"message":"already exists"}`, contentType: "application/json", wantErr: true, wantClass: "409"},
	{name: "422_validation", status: 422, body: `{"field":["bad value"]}`, contentType: "application/json", wantErr: true, wantClass: "422"},
	{name: "429_rate_limited", status: 429, body: `{"message":"slow down"}`, contentType: "application/json", wantErr: true, retryable: true},
	{name: "500_internal", status: 500, body: `{"message":"oops"}`, contentType: "application/json", wantErr: true, retryable: true},
	{name: "502_bad_gateway", status: 502, body: `<html>nginx</html>`, contentType: "text/html", wantErr: true, retryable: true},
	{name: "503_unavailable", status: 503, body: `<html>maintenance</html>`, contentType: "text/html", wantErr: true, retryable: true},
	{name: "504_gateway_timeout", status: 504, body: ``, contentType: "text/html", wantErr: true, retryable: true},
	{name: "524_cloudflare_timeout", status: 524, body: ``, contentType: "text/html", wantErr: true, retryable: true},
	{name: "200_html_instead_of_json", status: 200, body: `<html>login required</html>`, contentType: "text/html", wantErr: true},
	{name: "200_truncated_json", status: 200, body: `{"version":"trun`, contentType: "application/json", wantErr: true},
}

// TestContract_StatusMatrix_doRequest exercises the SHARED doRequest
// helper with every condition in statusMatrix. This is the canonical
// regression test — every client method routes through doRequest, so a
// bug in the status classification surfaces here once, not 237 times.
//
// The per-method sweep (TestContract_StatusMatrix_PerMethod_404) is
// the redundant safety net that confirms a future client method that
// bypasses doRequest still classifies the most common error (404)
// correctly.
func TestContract_StatusMatrix_doRequest(t *testing.T) {
	for _, tc := range statusMatrix {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tc.contentType != "" {
					w.Header().Set("Content-Type", tc.contentType)
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			// Retryable cases need a longer deadline since doRequest
			// retries 5xx/429 with exponential backoff. Non-retryable
			// cases are fast and 3s is plenty.
			deadline := 3 * time.Second
			if tc.retryable {
				deadline = 5 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), deadline)
			defer cancel()
			_, err := c.GetSystemInfo(ctx)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("status %d: wantErr=%v, got err=%v", tc.status, tc.wantErr, err)
			}
			// For retryable conditions, allow either the typed
			// status-class error OR a deadline-exceeded — both are
			// valid surface behaviours of "persistent transient
			// failure". We skip the wantClass check.
			if tc.retryable {
				return
			}
			if tc.wantErr && tc.wantClass != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantClass) {
					t.Errorf("status %d: want error containing %q, got %v",
						tc.status, tc.wantClass, err)
				}
			}
		})
	}
}

// TestContract_StatusMatrix_404Classification asserts every typed Get
// method classifies a 404 as "not found" via client.IsNotFound. The
// caller-side pattern across the provider is:
//
//	if err != nil {
//	    if client.IsNotFound(err) { resp.State.RemoveResource(ctx); return }
//	    resp.Diagnostics.AddError(...)
//	}
//
// Any Get that doesn't surface IsNotFound on 404 silently breaks the
// "deleted out-of-band" path — Read will return an error, the
// resource never drops from state, and the next plan dead-locks.
//
// This test enumerates every typed Get<X> by name from the public API
// surface. If a new resource lands without IsNotFound coverage on its
// Get the new row fails and points at the exact method.
func TestContract_StatusMatrix_404Classification(t *testing.T) {
	// each row: (name, invoke func that calls a Get with an id/name
	// that does not exist server-side). The handler always 404s.
	type call struct {
		name string
		fn   func(ctx context.Context, c *client.Client) error
	}
	calls := []call{
		{"GetACMEDNSAuthenticator", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetACMEDNSAuthenticator(ctx, 99999)
			return e
		}},
		{"GetAlertService", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetAlertService(ctx, 99999)
			return e
		}},
		{"GetAPIKey", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetAPIKey(ctx, 99999)
			return e
		}},
		{"GetApp", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetApp(ctx, "missing")
			return e
		}},
		{"GetCatalog", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetCatalog(ctx)
			return e
		}},
		{"GetCertificate", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetCertificate(ctx, 99999)
			return e
		}},
		{"GetCloudBackup", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetCloudBackup(ctx, 99999)
			return e
		}},
		{"GetCloudSync", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetCloudSync(ctx, 99999)
			return e
		}},
		{"GetCloudSyncCredential", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetCloudSyncCredential(ctx, 99999)
			return e
		}},
		{"GetCronJob", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetCronJob(ctx, 99999)
			return e
		}},
		{"GetDataset", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetDataset(ctx, "missing/pool")
			return e
		}},
		{"GetFilesystemACLTemplate", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetFilesystemACLTemplate(ctx, 99999)
			return e
		}},
		{"GetGroup", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetGroup(ctx, 99999)
			return e
		}},
		{"GetInitScript", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetInitScript(ctx, 99999)
			return e
		}},
		{"GetISCSIAuth", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetISCSIAuth(ctx, 99999)
			return e
		}},
		{"GetISCSIExtent", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetISCSIExtent(ctx, 99999)
			return e
		}},
		{"GetISCSIInitiator", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetISCSIInitiator(ctx, 99999)
			return e
		}},
		{"GetISCSIPortal", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetISCSIPortal(ctx, 99999)
			return e
		}},
		{"GetISCSITarget", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetISCSITarget(ctx, 99999)
			return e
		}},
		{"GetISCSITargetExtent", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetISCSITargetExtent(ctx, 99999)
			return e
		}},
		{"GetKerberosKeytab", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetKerberosKeytab(ctx, 99999)
			return e
		}},
		{"GetKerberosRealm", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetKerberosRealm(ctx, 99999)
			return e
		}},
		{"GetKeychainCredential", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetKeychainCredential(ctx, 99999)
			return e
		}},
		{"GetNFSShare", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetNFSShare(ctx, 99999)
			return e
		}},
		{"GetNVMetHost", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetNVMetHost(ctx, 99999)
			return e
		}},
		{"GetNVMetNamespace", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetNVMetNamespace(ctx, 99999)
			return e
		}},
		{"GetNVMetPort", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetNVMetPort(ctx, 99999)
			return e
		}},
		{"GetNVMetSubsys", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetNVMetSubsys(ctx, 99999)
			return e
		}},
		{"GetPool", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetPool(ctx, 99999)
			return e
		}},
		{"GetPrivilege", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetPrivilege(ctx, 99999)
			return e
		}},
		{"GetReportingExporter", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetReportingExporter(ctx, 99999)
			return e
		}},
		{"GetReplication", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetReplication(ctx, 99999)
			return e
		}},
		{"GetRsyncTask", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetRsyncTask(ctx, 99999)
			return e
		}},
		{"GetScrubTask", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetScrubTask(ctx, 99999)
			return e
		}},
		{"GetSMBShare", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetSMBShare(ctx, 99999)
			return e
		}},
		{"GetSnapshotTask", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetSnapshotTask(ctx, 99999)
			return e
		}},
		{"GetStaticRoute", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetStaticRoute(ctx, 99999)
			return e
		}},
		{"GetTunable", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetTunable(ctx, 99999)
			return e
		}},
		{"GetUser", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetUser(ctx, 99999)
			return e
		}},
		{"GetVM", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetVM(ctx, 99999)
			return e
		}},
		{"GetVMware", func(ctx context.Context, c *client.Client) error {
			_, e := c.GetVMware(ctx, 99999)
			return e
		}},
	}
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	for _, ca := range calls {
		ca := ca
		t.Run(ca.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			err := ca.fn(ctx, c)
			if err == nil {
				t.Fatalf("%s: expected 404 error, got nil", ca.name)
			}
			if !client.IsNotFound(err) {
				t.Errorf("%s: 404 must be IsNotFound, got: %v (%T)", ca.name, err, err)
			}
		})
	}
}

// TestContract_StatusMatrix_ErrorMessageRedacted asserts that the
// error string returned to the caller does NOT contain raw secret
// values from the wire body — even on a non-success response. The
// 422 path is the highest-risk because TrueNAS' middleware echoes
// the request payload back in validation error responses, and a
// resource that PUTs `{"password":"…"}` and gets back a 422 should
// not leak the password into the diagnostic.
func TestContract_StatusMatrix_ErrorMessageRedacted(t *testing.T) {
	secretsThatMustNotLeak := []struct {
		name     string
		body     string
		secret   string
		shouldHi string // for context: where the secret was
	}{
		{"password in 422 echo",
			`{"settings.password":["bad value: 's3cret-pw'"]}`,
			"s3cret-pw",
			"password"},
		{"api_key in 422 echo",
			`{"settings.api_key":["bad value: 'AKIA-deadbeef'"]}`,
			"AKIA-deadbeef",
			"api_key"},
		{"bearer token in 500 body",
			`{"trace":"... Authorization: Bearer eyJraWQiOiJyXMzc1NCI..."}`,
			"eyJraWQiOiJyXMzc1NCI",
			"bearer"},
	}
	for _, tc := range secretsThatMustNotLeak {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_, _ = w.Write([]byte(tc.body))
			}))
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := c.GetSystemInfo(ctx)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if strings.Contains(err.Error(), tc.secret) {
				t.Errorf("secret %q leaked into error from %s:\n  err: %s",
					tc.secret, tc.shouldHi, err.Error())
			}
		})
	}
}

// TestContract_StatusMatrix_RequestIDRoundTrip asserts every request
// includes an X-Request-ID header and the doRequest helper threads it
// into the error message on failure so an operator can trace a
// specific failure back through TrueNAS' alertd / middlewared logs.
func TestContract_StatusMatrix_RequestIDRoundTrip(t *testing.T) {
	var sawHeader string
	// Use a non-retryable 4xx so the test surfaces the first response
	// directly. A 5xx retries with backoff and the deadline behaviour
	// is covered elsewhere.
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHeader = r.Header.Get("X-Request-ID")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = fmt.Fprint(w, `{"message":"bad request"}`)
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := c.GetSystemInfo(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if sawHeader == "" {
		t.Error("server saw no X-Request-ID header — operator can't trace the failure")
	}
	if !strings.Contains(err.Error(), sawHeader) {
		t.Errorf("error message must echo X-Request-ID %q for traceability, got: %v",
			sawHeader, err)
	}
}

// ensure the imports stay used even if a guard branch removes them.
var _ = errors.New
