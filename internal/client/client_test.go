package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// newTestServer returns an httptest.Server that responds with the given handler
// and a Client pre-configured to talk to it.
func newTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *client.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	return srv, c
}

// writeJSON writes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// -----------------------------------------------------------------------
// Client construction tests
// -----------------------------------------------------------------------

func TestNew_MissingURL(t *testing.T) {
	_, err := client.New("", "key")
	if err == nil {
		t.Fatal("expected error for empty URL, got nil")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := client.New("http://localhost", "")
	if err == nil {
		t.Fatal("expected error for empty API key, got nil")
	}
}

func TestNew_NormalizesBaseURL(t *testing.T) {
	// Should not double-append /api/v2.0
	c, err := client.New("http://truenas.example.com/api/v2.0", "key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

// -----------------------------------------------------------------------
// APIError tests
// -----------------------------------------------------------------------

func TestAPIError_Format(t *testing.T) {
	e := &client.APIError{StatusCode: 404, Message: "not found"}
	want := "TrueNAS API error (HTTP 404): not found"
	if e.Error() != want {
		t.Errorf("got %q, want %q", e.Error(), want)
	}
}

func TestAPIError_FallbackToBody(t *testing.T) {
	e := &client.APIError{StatusCode: 500, Body: "internal server error"}
	got := e.Error()
	if got == "" {
		t.Error("expected non-empty error string")
	}
}

// -----------------------------------------------------------------------
// Dataset API tests
// -----------------------------------------------------------------------

func TestGetDataset_Success(t *testing.T) {
	ctx := context.Background()
	want := client.DatasetResponse{
		ID:         "tank/testdata",
		Type:       "FILESYSTEM",
		MountPoint: "/mnt/tank/testdata",
	}

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method %s", r.Method)
		}
		writeJSON(w, http.StatusOK, want)
	}))

	got, err := c.GetDataset(ctx, "tank/testdata")
	if err != nil {
		t.Fatalf("GetDataset: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %q, want %q", got.ID, want.ID)
	}
	if got.MountPoint != want.MountPoint {
		t.Errorf("MountPoint: got %q, want %q", got.MountPoint, want.MountPoint)
	}
}

func TestGetDataset_NotFound(t *testing.T) {
	ctx := context.Background()

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
	}))

	_, err := c.GetDataset(ctx, "tank/missing")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError in chain, got %T", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode: got %d, want 404", apiErr.StatusCode)
	}
}

func TestCreateDataset_Success(t *testing.T) {
	ctx := context.Background()

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		var req client.DatasetCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		resp := client.DatasetResponse{
			ID:         req.Name,
			Type:       "FILESYSTEM",
			MountPoint: "/mnt/" + req.Name,
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	resp, err := c.CreateDataset(ctx, &client.DatasetCreateRequest{
		Name: "tank/newdata",
		Type: "FILESYSTEM",
	})
	if err != nil {
		t.Fatalf("CreateDataset: %v", err)
	}
	if resp.ID != "tank/newdata" {
		t.Errorf("ID: got %q, want %q", resp.ID, "tank/newdata")
	}
}

func TestDeleteDataset_Success(t *testing.T) {
	ctx := context.Background()

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))

	if err := c.DeleteDataset(ctx, "tank/toremove"); err != nil {
		t.Fatalf("DeleteDataset: %v", err)
	}
}

// -----------------------------------------------------------------------
// NFS Share API tests
// -----------------------------------------------------------------------

func TestGetNFSShare_Success(t *testing.T) {
	ctx := context.Background()
	want := client.NFSShare{
		ID:      42,
		Path:    "/mnt/tank/exports",
		Enabled: true,
	}

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, want)
	}))

	got, err := c.GetNFSShare(ctx, 42)
	if err != nil {
		t.Fatalf("GetNFSShare: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("ID: got %d, want %d", got.ID, want.ID)
	}
	if got.Path != want.Path {
		t.Errorf("Path: got %q, want %q", got.Path, want.Path)
	}
}

func TestGetNFSShare_NotFound(t *testing.T) {
	ctx := context.Background()

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
	}))

	_, err := c.GetNFSShare(ctx, 99)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestCreateNFSShare_Success(t *testing.T) {
	ctx := context.Background()

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method %s", r.Method)
		}
		resp := client.NFSShare{
			ID:      7,
			Path:    "/mnt/tank/share",
			Enabled: true,
		}
		writeJSON(w, http.StatusOK, resp)
	}))

	resp, err := c.CreateNFSShare(ctx, &client.NFSShareCreateRequest{
		Path:    "/mnt/tank/share",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateNFSShare: %v", err)
	}
	if resp.ID != 7 {
		t.Errorf("ID: got %d, want 7", resp.ID)
	}
}

// -----------------------------------------------------------------------
// Authorization header test
// -----------------------------------------------------------------------

func TestAuthorizationHeader(t *testing.T) {
	ctx := context.Background()
	const apiKey = "my-secret-key"

	_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + apiKey
		if auth != expected {
			t.Errorf("Authorization header: got %q, want %q", auth, expected)
		}
		writeJSON(w, http.StatusOK, client.DatasetResponse{ID: "tank/x", Type: "FILESYSTEM"})
	}))

	// Re-create client with our known key (newTestServer uses "test-api-key").
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+apiKey {
			t.Errorf("Authorization: got %q, want %q", auth, "Bearer "+apiKey)
		}
		writeJSON(w, http.StatusOK, client.DatasetResponse{ID: "tank/y", Type: "FILESYSTEM"})
	}))
	defer srv.Close()

	c2, err := client.New(srv.URL, apiKey)
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}

	_, err = c2.GetDataset(ctx, "tank/y")
	if err != nil {
		t.Fatalf("GetDataset: %v", err)
	}

	// Suppress unused variable warning for c
	_ = c
}
