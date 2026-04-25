package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestPool_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetPool success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/pool/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Pool{
				ID: 3, Name: "tank", GUID: "12345", Path: "/mnt/tank",
				Status: "ONLINE", Healthy: true, IsDecrypted: true,
			})
		}))
		p, err := c.GetPool(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Name != "tank" {
			t.Errorf("Name = %q", p.Name)
		}
		if !p.Healthy {
			t.Errorf("Healthy = false")
		}
	})

	t.Run("GetPool 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetPool(ctx, 999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetPool invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetPool(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetPool server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "oops"})
		}))
		_, err := c.GetPool(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("ListPools success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/pool") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.Pool{
				{ID: 1, Name: "tank", Healthy: true},
				{ID: 2, Name: "backup", Healthy: true},
			})
		}))
		list, err := c.ListPools(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("ListPools invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.ListPools(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("ListPools server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "oops"})
		}))
		_, err := c.ListPools(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetPoolByName success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Pool{
				{ID: 1, Name: "tank"},
				{ID: 2, Name: "backup"},
			})
		}))
		p, err := c.GetPoolByName(ctx, "backup")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ID != 2 {
			t.Errorf("ID = %d", p.ID)
		}
	})

	t.Run("GetPoolByName not found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Pool{{ID: 1, Name: "tank"}})
		}))
		_, err := c.GetPoolByName(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetPoolByName list error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "oops"})
		}))
		_, err := c.GetPoolByName(ctx, "tank")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreatePool direct pool response", func(t *testing.T) {
		// When the API returns a Pool object directly instead of a job ID,
		// CreatePool should unmarshal it and return immediately (no job polling).
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/pool") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.PoolCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Name != "newpool" {
				t.Errorf("Name = %q", req.Name)
			}
			writeJSON(w, http.StatusOK, client.Pool{
				ID: 42, Name: "newpool", GUID: "abc",
			})
		}))
		resp, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name:     "newpool",
			Topology: json.RawMessage(`{"data":[{"type":"STRIPE","disks":["da1"]}]}`),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 42 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("CreatePool server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad topology"})
		}))
		_, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "bad", Topology: json.RawMessage(`{}`),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("CreatePool invalid JSON", func(t *testing.T) {
		// Response is neither a job ID nor a Pool object.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`"not-a-number-or-object"`))
		}))
		_, err := c.CreatePool(ctx, &client.PoolCreateRequest{
			Name: "x", Topology: json.RawMessage(`{}`),
		})
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("ExportPool immediate completion", func(t *testing.T) {
		// Non-numeric response means export is complete without polling.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/pool/id/5/export") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.PoolExportRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if !req.Destroy {
				t.Errorf("Destroy = false, want true")
			}
			if !req.Cascade {
				t.Errorf("Cascade = false, want true")
			}
			w.WriteHeader(http.StatusOK)
			// Non-numeric response: the int unmarshal fails, so ExportPool
			// treats the call as already complete without job polling.
			_, _ = w.Write([]byte(`"ok"`))
		}))
		err := c.ExportPool(ctx, 5, &client.PoolExportRequest{
			Cascade: true, Destroy: true, RestartServices: false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ExportPool 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.ExportPool(ctx, 1, &client.PoolExportRequest{Destroy: true})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("ExportPool server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "nope"})
		}))
		err := c.ExportPool(ctx, 1, &client.PoolExportRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
