package client_test

// This file collects additional branch-coverage tests added to push the
// internal/client package to 100% literal line coverage. Tests here focus
// on error/edge paths not exercised by the primary per-file *_test.go
// suites (HTTP errors, invalid JSON, job failures, zero-value getters,
// etc). Tests are grouped by source file.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// --------------------------------------------------------------------
// Shared helpers for this file.
// --------------------------------------------------------------------

// garbageServer returns a server that always replies 200 OK with non-JSON
// bytes, so JSON parse error branches can be exercised.
func garbageServer(t *testing.T) *client.Client {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not json"))
	}))
	return c
}

// errorServer returns a server that always replies with the given status
// and a JSON message body; retries are disabled so errors surface quickly.
func errorServer(t *testing.T, status int) *client.Client {
	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, status, map[string]string{"message": "boom"})
	}))
	withFastRetries(c, 1)
	return c
}

// expectAPIError asserts err is a non-nil *client.APIError.
func expectAPIError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
}

// --------------------------------------------------------------------
// DatasetResponse zvol getters (dataset.go:99,113) — pure helpers.
// --------------------------------------------------------------------

func TestDatasetResponse_GetVolsize(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		d := client.DatasetResponse{}
		if got := d.GetVolsize(); got != 0 {
			t.Errorf("GetVolsize nil = %d, want 0", got)
		}
	})
	t.Run("empty rawvalue", func(t *testing.T) {
		d := client.DatasetResponse{Volsize: &client.PropertyRawVal{}}
		if got := d.GetVolsize(); got != 0 {
			t.Errorf("GetVolsize empty = %d, want 0", got)
		}
	})
	t.Run("valid rawvalue", func(t *testing.T) {
		d := client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: "16777216"}}
		if got := d.GetVolsize(); got != 16777216 {
			t.Errorf("GetVolsize = %d, want 16777216", got)
		}
	})
	t.Run("non-numeric rawvalue falls through to 0", func(t *testing.T) {
		d := client.DatasetResponse{Volsize: &client.PropertyRawVal{Rawvalue: "not-a-number"}}
		if got := d.GetVolsize(); got != 0 {
			t.Errorf("GetVolsize non-numeric = %d, want 0", got)
		}
	})
}

func TestDatasetResponse_GetVolblocksize(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		d := client.DatasetResponse{}
		if got := d.GetVolblocksize(); got != "" {
			t.Errorf("GetVolblocksize nil = %q, want empty", got)
		}
	})
	t.Run("value set", func(t *testing.T) {
		d := client.DatasetResponse{Volblocksize: &client.PropertyValue{Value: "16K"}}
		if got := d.GetVolblocksize(); got != "16K" {
			t.Errorf("GetVolblocksize = %q, want 16K", got)
		}
	})
}

// --------------------------------------------------------------------
// SyncCatalog (app.go:212) — triggers a job via GET /catalog/sync.
// --------------------------------------------------------------------

func TestSyncCatalog(t *testing.T) {
	ctx := context.Background()

	t.Run("non-job response (synchronous)", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/catalog/sync") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		}))
		if err := c.SyncCatalog(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("job success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 55, State: "SUCCESS"}})
			case strings.HasSuffix(r.URL.Path, "/catalog/sync"):
				writeJSON(w, http.StatusOK, 55)
			default:
				t.Errorf("unexpected path: %s", r.URL.Path)
				writeJSON(w, http.StatusNotFound, map[string]string{})
			}
		}))
		if err := c.SyncCatalog(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("job failed", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 56, State: "FAILED", Error: "sync error"}})
			case strings.HasSuffix(r.URL.Path, "/catalog/sync"):
				writeJSON(w, http.StatusOK, 56)
			default:
				writeJSON(w, http.StatusNotFound, map[string]string{})
			}
		}))
		if err := c.SyncCatalog(ctx); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GET error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.SyncCatalog(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// Per-file branch coverage: simple CRUD error paths.
// Each block below targets the remaining uncovered branches in its
// corresponding source file: JSON parse errors on Get/Create/Update,
// HTTP errors on Update/Delete, etc.
// --------------------------------------------------------------------

func TestACMEDNSAuthenticator_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	// Update JSON parse error (line 69.52-71.3).
	c := garbageServer(t)
	_, err := c.UpdateACMEDNSAuthenticator(ctx, 1, &client.ACMEDNSAuthenticatorUpdateRequest{})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestAlertService_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("Get JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetAlertService(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Create JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateAlertService(ctx, &client.AlertServiceCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Update HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateAlertService(ctx, 1, &client.AlertServiceUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("Update JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateAlertService(ctx, 1, &client.AlertServiceUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Delete HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteAlertService(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestApp_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListApps HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListApps(ctx); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("UpdateApp response is raw App (non-job)", func(t *testing.T) {
		// PUT response is a bare App object (not a job ID); exercises the
		// fallback path at app.go:128-134 that returns the unmarshaled App.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusOK, client.App{ID: "directapp", Name: "directapp"})
		}))
		got, err := c.UpdateApp(ctx, "directapp", &client.AppUpdateRequest{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "directapp" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("UpdateApp unparseable response", func(t *testing.T) {
		// Response is neither an int nor an App — should return parse error.
		c := garbageServer(t)
		_, err := c.UpdateApp(ctx, "x", &client.AppUpdateRequest{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("UpdateApp job fails", func(t *testing.T) {
		// Job ID parses OK but WaitForJob reports failure.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 9, State: "FAILED", Error: "nope"}})
			case r.Method == http.MethodPut:
				writeJSON(w, http.StatusOK, 9)
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		if _, err := c.UpdateApp(ctx, "x", &client.AppUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("DeleteApp job fails", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 10, State: "FAILED", Error: "fail"}})
			case r.Method == http.MethodDelete:
				writeJSON(w, http.StatusOK, 10)
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		if err := c.DeleteApp(ctx, "x", &client.AppDeleteRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GetCatalog JSON parse error", func(t *testing.T) {
		// Already covered in app_test.go, but include again for resilience.
		c := garbageServer(t)
		if _, err := c.GetCatalog(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateCatalog HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateCatalog(ctx, &client.CatalogUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("UpdateCatalog JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateCatalog(ctx, &client.CatalogUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestCloudSync_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("GetCloudSync JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetCloudSync(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateCloudSync HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateCloudSync(ctx, &client.CloudSyncCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateCloudSync JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateCloudSync(ctx, &client.CloudSyncCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateCloudSync HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateCloudSync(ctx, 1, &client.CloudSyncUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateCloudSync JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateCloudSync(ctx, 1, &client.CloudSyncUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

func TestCloudsyncCredential_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListCloudSyncCredentials HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListCloudSyncCredentials(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ListCloudSyncCredentials JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListCloudSyncCredentials(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("GetCloudSyncCredentialByName HTTP error", func(t *testing.T) {
		// ListCloudSyncCredentials called internally; use list HTTP error.
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetCloudSyncCredentialByName(ctx, "foo"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDataset_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListDatasets HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListDatasets(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ListDatasets JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListDatasets(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateDataset JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateDataset(ctx, &client.DatasetCreateRequest{Name: "x"}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateDataset HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateDataset(ctx, "x", &client.DatasetUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateDataset JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateDataset(ctx, "x", &client.DatasetUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

// Ensure error helper is reachable in at least one test path.
var _ = expectAPIError
var _ = json.RawMessage{}

// --------------------------------------------------------------------
// App branch coverage: DeleteApp non-job response + GetCatalog HTTP err.
// --------------------------------------------------------------------

func TestApp_ExtraBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("DeleteApp non-job response (sync delete)", func(t *testing.T) {
		// DELETE returns non-int JSON — code path: parse fails and returns nil.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		}))
		if err := c.DeleteApp(ctx, "x", &client.AppDeleteRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("GetCatalog HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetCatalog(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// Certificate branch coverage.
// --------------------------------------------------------------------

func TestCertificate_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListCertificates JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListCertificates(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("GetCertificate JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetCertificate(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("GetCertificateByName list error propagates", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetCertificateByName(ctx, "foo"); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateCertificate raw Certificate (non-job) response", func(t *testing.T) {
		// PUT returns a bare Certificate object, not a job ID.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusOK, client.Certificate{ID: 7, Name: "direct"})
		}))
		got, err := c.UpdateCertificate(ctx, 7, &client.CertificateUpdateRequest{Name: "direct"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 7 || got.Name != "direct" {
			t.Errorf("got %+v", got)
		}
	})
	t.Run("UpdateCertificate unparseable response", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateCertificate(ctx, 1, &client.CertificateUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateCertificate job failed", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 11, State: "FAILED", Error: "nope"}})
			case r.Method == http.MethodPut:
				writeJSON(w, http.StatusOK, 11)
			default:
				t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		if _, err := c.UpdateCertificate(ctx, 1, &client.CertificateUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("DeleteCertificate job ENOENT -> 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 12, State: "FAILED", Error: "[ENOENT] Certificate 1 does not exist"}})
			case r.Method == http.MethodDelete:
				writeJSON(w, http.StatusOK, 12)
			default:
				t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		err := c.DeleteCertificate(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound=false, err=%v", err)
		}
	})
	t.Run("DeleteCertificate job other failure", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 13, State: "FAILED", Error: "internal problem"}})
			case r.Method == http.MethodDelete:
				writeJSON(w, http.StatusOK, 13)
			default:
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		err := c.DeleteCertificate(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
		if client.IsNotFound(err) {
			t.Error("IsNotFound should be false for non-ENOENT errors")
		}
	})
}

// --------------------------------------------------------------------
// Group branch coverage.
// --------------------------------------------------------------------

func TestGroup_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListGroups HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListGroups(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ListGroups JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListGroups(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("GetGroupByName list error propagates", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetGroupByName(ctx, "admins"); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetGroup HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetGroup(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetGroup JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetGroup(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateGroup HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateGroup(ctx, &client.GroupCreateRequest{Name: "x"}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateGroup parse error on ID response", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateGroup(ctx, &client.GroupCreateRequest{Name: "x"}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateGroup HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateGroup(ctx, 1, &client.GroupUpdateRequest{Name: "x"}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateGroup parse error on ID response", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateGroup(ctx, 1, &client.GroupUpdateRequest{Name: "x"}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

// --------------------------------------------------------------------
// VM branch coverage.
// --------------------------------------------------------------------

func TestVM_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListVMs JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListVMs(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateVM JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateVM(ctx, 1, &client.VMUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("StartVM HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.StartVM(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetVMDevice JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetVMDevice(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateVMDevice HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateVMDevice(ctx, &client.VMDeviceCreateRequest{VM: 1}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateVMDevice JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateVMDevice(ctx, &client.VMDeviceCreateRequest{VM: 1}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateVMDevice HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateVMDevice(ctx, 1, &client.VMDeviceUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateVMDevice JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateVMDevice(ctx, 1, &client.VMDeviceUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

// --------------------------------------------------------------------
// SMB share branch coverage.
// --------------------------------------------------------------------

func TestSMBShare_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListSMBShares HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListSMBShares(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ListSMBShares JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListSMBShares(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("GetSMBShare JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetSMBShare(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateSMBShare JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateSMBShare(ctx, &client.SMBShareCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateSMBShare HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateSMBShare(ctx, 1, &client.SMBShareUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateSMBShare JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateSMBShare(ctx, 1, &client.SMBShareUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteSMBShare HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteSMBShare(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// iSCSI extent branch coverage (including GetDisk unmarshal error).
// --------------------------------------------------------------------

func TestISCSIExtent_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("GetDisk unmarshal error", func(t *testing.T) {
		// Disk raw bytes are not a valid JSON string — force the unmarshal
		// error path (iscsi_extent.go:34-36).
		e := client.ISCSIExtent{Disk: json.RawMessage(`{"not":"string"}`)}
		if e.GetDisk() != "" {
			t.Errorf("expected empty from bad disk json, got %q", e.GetDisk())
		}
	})
	t.Run("GetISCSIExtent JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetISCSIExtent(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateISCSIExtent HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateISCSIExtent(ctx, &client.ISCSIExtentCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateISCSIExtent JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateISCSIExtent(ctx, &client.ISCSIExtentCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateISCSIExtent HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateISCSIExtent(ctx, 1, &client.ISCSIExtentUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateISCSIExtent JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateISCSIExtent(ctx, 1, &client.ISCSIExtentUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteISCSIExtent HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteISCSIExtent(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// iSCSI portal branch coverage.
// --------------------------------------------------------------------

func TestISCSIPortal_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("GetISCSIPortal JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetISCSIPortal(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateISCSIPortal HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateISCSIPortal(ctx, &client.ISCSIPortalCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateISCSIPortal JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateISCSIPortal(ctx, &client.ISCSIPortalCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateISCSIPortal HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateISCSIPortal(ctx, 1, &client.ISCSIPortalUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateISCSIPortal JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateISCSIPortal(ctx, 1, &client.ISCSIPortalUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteISCSIPortal HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteISCSIPortal(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// iSCSI target branch coverage.
// --------------------------------------------------------------------

func TestISCSITarget_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("GetISCSITarget JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetISCSITarget(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateISCSITarget HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateISCSITarget(ctx, &client.ISCSITargetCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateISCSITarget JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateISCSITarget(ctx, &client.ISCSITargetCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateISCSITarget HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateISCSITarget(ctx, 1, &client.ISCSITargetUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateISCSITarget JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateISCSITarget(ctx, 1, &client.ISCSITargetUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteISCSITarget HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteISCSITarget(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// Snapshot task branch coverage.
// --------------------------------------------------------------------

func TestSnapshotTask_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("Get JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetSnapshotTask(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Create HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateSnapshotTask(ctx, &client.SnapshotTaskCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("Create JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateSnapshotTask(ctx, &client.SnapshotTaskCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Update HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateSnapshotTask(ctx, 1, &client.SnapshotTaskUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("Update JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateSnapshotTask(ctx, 1, &client.SnapshotTaskUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Delete HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteSnapshotTask(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// NFS share branch coverage.
// --------------------------------------------------------------------

func TestNFSShare_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("Get JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNFSShare(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("List HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListNFSShares(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("List JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListNFSShares(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Create JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNFSShare(ctx, &client.NFSShareCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("Update HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateNFSShare(ctx, 1, &client.NFSShareUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("Update JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateNFSShare(ctx, 1, &client.NFSShareUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

// --------------------------------------------------------------------
// Network interface branch coverage.
// --------------------------------------------------------------------

func TestNetworkInterface_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("commitAndCheckin checkin fails", func(t *testing.T) {
		// commitAndCheckin POST /interface/commit succeeds but GET
		// /interface/checkin fails (line 106-108).
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/interface/id/"):
				writeJSON(w, http.StatusOK, client.NetworkInterface{ID: "br0", Name: "br0"})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/interface/commit"):
				writeJSON(w, http.StatusOK, map[string]string{})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/interface/checkin"):
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "checkin failed"})
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		withFastRetries(c, 1)
		if _, err := c.UpdateInterface(ctx, "br0", &client.NetworkInterfaceUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ListInterfaces HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListInterfaces(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ListInterfaces JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListInterfaces(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateInterface JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateInterface(ctx, "br0", &client.NetworkInterfaceUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteInterface commit fails", func(t *testing.T) {
		// Delete succeeds but commit fails.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/interface/id/"):
				writeJSON(w, http.StatusOK, map[string]string{})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/interface/commit"):
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "commit fail"})
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		withFastRetries(c, 1)
		if err := c.DeleteInterface(ctx, "br0"); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// Kerberos branch coverage.
// --------------------------------------------------------------------

func TestKerberos_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListKerberosRealms JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.ListKerberosRealms(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateKerberosRealm JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateKerberosRealm(ctx, &client.KerberosRealmCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateKerberosRealm JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateKerberosRealm(ctx, 1, &client.KerberosRealmUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateKerberosKeytab JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateKerberosKeytab(ctx, &client.KerberosKeytabCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateKerberosKeytab JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateKerberosKeytab(ctx, 1, &client.KerberosKeytabUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
}

// --------------------------------------------------------------------
// Privilege, reporting_exporter, filesystem_acl_template, directoryservices.
// --------------------------------------------------------------------

func TestMisc_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("CreatePrivilege JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreatePrivilege(ctx, &client.PrivilegeCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdatePrivilege JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdatePrivilege(ctx, 1, &client.PrivilegeUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateReportingExporter JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateReportingExporter(ctx, 1, &client.ReportingExporterUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateFilesystemACLTemplate JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateFilesystemACLTemplate(ctx, 1, &client.FilesystemACLTemplateUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateDirectoryServicesConfig fallback re-fetch", func(t *testing.T) {
		// PUT returns garbage (not int, not struct) -> fallback re-fetch path.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("totally not json"))
			case http.MethodGet:
				writeJSON(w, http.StatusOK, client.DirectoryServicesConfig{})
			}
		}))
		if _, err := c.UpdateDirectoryServicesConfig(ctx, &client.DirectoryServicesUpdateRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// --------------------------------------------------------------------
// NVMe-oF branch coverage (all Get/Create/Update/Delete error paths).
// --------------------------------------------------------------------

func TestNVMet_ErrorBranches(t *testing.T) {
	ctx := context.Background()

	// --- Global ---
	t.Run("GetNVMetGlobal JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetGlobal(ctx); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateNVMetGlobal HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateNVMetGlobal(ctx, &client.NVMetGlobalUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateNVMetGlobal JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateNVMetGlobal(ctx, &client.NVMetGlobalUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})

	// --- Host ---
	t.Run("GetNVMetHost HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetNVMetHost(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetNVMetHost JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetHost(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateNVMetHost HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateNVMetHost(ctx, &client.NVMetHostCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateNVMetHost JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNVMetHost(ctx, &client.NVMetHostCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateNVMetHost HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateNVMetHost(ctx, 1, &client.NVMetHostUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateNVMetHost JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateNVMetHost(ctx, 1, &client.NVMetHostUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteNVMetHost HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteNVMetHost(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	// --- Subsys ---
	t.Run("GetNVMetSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetNVMetSubsys(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetNVMetSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetSubsys(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateNVMetSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateNVMetSubsys(ctx, &client.NVMetSubsysCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateNVMetSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNVMetSubsys(ctx, &client.NVMetSubsysCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateNVMetSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateNVMetSubsys(ctx, 1, &client.NVMetSubsysUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateNVMetSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateNVMetSubsys(ctx, 1, &client.NVMetSubsysUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteNVMetSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteNVMetSubsys(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	// --- Port ---
	t.Run("GetNVMetPort HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetNVMetPort(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetNVMetPort JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetPort(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateNVMetPort HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateNVMetPort(ctx, &client.NVMetPortCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateNVMetPort JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNVMetPort(ctx, &client.NVMetPortCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateNVMetPort HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateNVMetPort(ctx, 1, &client.NVMetPortUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateNVMetPort JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateNVMetPort(ctx, 1, &client.NVMetPortUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteNVMetPort HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteNVMetPort(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	// --- Namespace ---
	t.Run("GetNVMetNamespace HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetNVMetNamespace(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetNVMetNamespace JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetNamespace(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateNVMetNamespace HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateNVMetNamespace(ctx, &client.NVMetNamespaceCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateNVMetNamespace JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNVMetNamespace(ctx, &client.NVMetNamespaceCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("UpdateNVMetNamespace HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.UpdateNVMetNamespace(ctx, 1, &client.NVMetNamespaceUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("UpdateNVMetNamespace JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.UpdateNVMetNamespace(ctx, 1, &client.NVMetNamespaceUpdateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteNVMetNamespace HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteNVMetNamespace(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	// --- Host-Subsys ---
	t.Run("GetNVMetHostSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetNVMetHostSubsys(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetNVMetHostSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetHostSubsys(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateNVMetHostSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateNVMetHostSubsys(ctx, &client.NVMetHostSubsysCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateNVMetHostSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNVMetHostSubsys(ctx, &client.NVMetHostSubsysCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteNVMetHostSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteNVMetHostSubsys(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	// --- Port-Subsys ---
	t.Run("GetNVMetPortSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.GetNVMetPortSubsys(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("GetNVMetPortSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.GetNVMetPortSubsys(ctx, 1); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("CreateNVMetPortSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.CreateNVMetPortSubsys(ctx, &client.NVMetPortSubsysCreateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("CreateNVMetPortSubsys JSON parse error", func(t *testing.T) {
		c := garbageServer(t)
		if _, err := c.CreateNVMetPortSubsys(ctx, &client.NVMetPortSubsysCreateRequest{}); err == nil {
			t.Fatal("expected parse error")
		}
	})
	t.Run("DeleteNVMetPortSubsys HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if err := c.DeleteNVMetPortSubsys(ctx, 1); err == nil {
			t.Fatal("expected error")
		}
	})

	// --- EffectiveSubsysID fallthrough (nested ID == 0) ---
	t.Run("HostSubsys EffectiveSubsysID nested 0", func(t *testing.T) {
		hs := client.NVMetHostSubsys{
			Subsys:   &client.NVMetHostSubsysSubsys{ID: 0},
			SubsysID: 42,
		}
		if got := hs.EffectiveSubsysID(); got != 42 {
			t.Errorf("EffectiveSubsysID = %d, want 42", got)
		}
	})
	t.Run("PortSubsys EffectiveSubsysID nested 0", func(t *testing.T) {
		ps := client.NVMetPortSubsys{
			Subsys:   &client.NVMetPortSubsysSubsys{ID: 0},
			SubsysID: 99,
		}
		if got := ps.EffectiveSubsysID(); got != 99 {
			t.Errorf("EffectiveSubsysID = %d, want 99", got)
		}
	})
}

// --------------------------------------------------------------------
// Kerberos ListKerberosRealms and vm ListVMs HTTP error branches.
// --------------------------------------------------------------------

func TestStragglers_HTTPErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("ListKerberosRealms HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListKerberosRealms(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("ListVMs HTTP error", func(t *testing.T) {
		c := errorServer(t, http.StatusInternalServerError)
		if _, err := c.ListVMs(ctx); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// Certificate job-result parse error branches (lines 132, 163).
// --------------------------------------------------------------------

func TestCertificate_JobResultParseError(t *testing.T) {
	ctx := context.Background()

	t.Run("CreateCertificate job result is garbage", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				// Job succeeded but Result is not a Certificate object.
				writeJSON(w, http.StatusOK, []client.Job{{
					ID:     20,
					State:  "SUCCESS",
					Result: json.RawMessage(`"not an object"`),
				}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/certificate"):
				writeJSON(w, http.StatusOK, 20)
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			}
		}))
		if _, err := c.CreateCertificate(ctx, &client.CertificateCreateRequest{Name: "x"}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("UpdateCertificate job result is garbage", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{
					ID:     21,
					State:  "SUCCESS",
					Result: json.RawMessage(`"garbage"`),
				}})
			case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/certificate/id/"):
				writeJSON(w, http.StatusOK, 21)
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
			}
		}))
		if _, err := c.UpdateCertificate(ctx, 1, &client.CertificateUpdateRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
}

// --------------------------------------------------------------------
// Pool CreatePool/ExportPool job polling branches.
// --------------------------------------------------------------------

func TestPool_JobBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("CreatePool job success with pool in result", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{
					ID:     30,
					State:  "SUCCESS",
					Result: json.RawMessage(`{"id":42,"name":"mypool","guid":"g","path":"/mnt/mypool","status":"ONLINE","healthy":true}`),
				}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool"):
				writeJSON(w, http.StatusOK, 30)
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		pool, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "mypool", Topology: json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pool.ID != 42 || pool.Name != "mypool" {
			t.Errorf("got %+v", pool)
		}
	})

	t.Run("CreatePool job success with null result -> fallback list lookup", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 31, State: "SUCCESS"}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool"):
				writeJSON(w, http.StatusOK, 31)
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pool"):
				// fallback lookup
				writeJSON(w, http.StatusOK, []client.Pool{
					{ID: 44, Name: "fallback-pool", Healthy: true},
				})
			default:
				t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{})
			}
		}))
		pool, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "fallback-pool", Topology: json.RawMessage(`{}`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pool.ID != 44 {
			t.Errorf("ID = %d", pool.ID)
		}
	})

	t.Run("CreatePool job success, result garbage -> parse error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{
					ID:     32,
					State:  "SUCCESS",
					Result: json.RawMessage(`"not-a-pool"`),
				}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool"):
				writeJSON(w, http.StatusOK, 32)
			}
		}))
		if _, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "x", Topology: json.RawMessage(`{}`),
		}); err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("CreatePool fallback list lookup fails", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 33, State: "SUCCESS"}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool"):
				writeJSON(w, http.StatusOK, 33)
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pool"):
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "list fail"})
			}
		}))
		withFastRetries(c, 1)
		if _, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "x", Topology: json.RawMessage(`{}`),
		}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("CreatePool job failed", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 34, State: "FAILED", Error: "no disks"}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool"):
				writeJSON(w, http.StatusOK, 34)
			}
		}))
		if _, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "x", Topology: json.RawMessage(`{}`),
		}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("ExportPool job success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 40, State: "SUCCESS"}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool/id/1/export"):
				writeJSON(w, http.StatusOK, 40)
			}
		}))
		if err := c.ExportPool(ctx, 1, &client.PoolExportRequest{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ExportPool job failed", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/core/get_jobs"):
				writeJSON(w, http.StatusOK, []client.Job{{ID: 41, State: "FAILED", Error: "busy"}})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pool/id/1/export"):
				writeJSON(w, http.StatusOK, 41)
			}
		}))
		if err := c.ExportPool(ctx, 1, &client.PoolExportRequest{}); err == nil {
			t.Fatal("expected error")
		}
	})
}
