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

// Zvol reuses the /pool/dataset endpoint with type=VOLUME, so tests focus on
// CreateZvol auto-forcing Type, plus delegation to dataset CRUD.
func TestZvol_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Create forces VOLUME type", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/pool/dataset") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req["type"].(string) != "VOLUME" {
				t.Errorf("type = %v, want VOLUME", req["type"])
			}
			if req["name"].(string) != "tank/vol1" {
				t.Errorf("name = %v", req["name"])
			}
			if req["volsize"].(float64) != float64(1024*1024*1024) {
				t.Errorf("volsize = %v", req["volsize"])
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{
				ID:   "tank/vol1",
				Name: "vol1",
				Pool: "tank",
				Type: "VOLUME",
			})
		}))

		got, err := c.CreateZvol(ctx, &client.ZvolCreateRequest{
			Name:        "tank/vol1",
			Volsize:     1024 * 1024 * 1024,
			Compression: "lz4",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "tank/vol1" {
			t.Errorf("ID = %q", got.ID)
		}
		if got.Type != "VOLUME" {
			t.Errorf("Type = %q", got.Type)
		}
	})

	t.Run("Create sends volblocksize and comments", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			s := string(body)
			if !strings.Contains(s, `"volblocksize":"64K"`) {
				t.Errorf("body missing volblocksize: %s", s)
			}
			if !strings.Contains(s, `"comments":"test zvol"`) {
				t.Errorf("body missing comments: %s", s)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{ID: "tank/vol2", Type: "VOLUME"})
		}))

		_, err := c.CreateZvol(ctx, &client.ZvolCreateRequest{
			Name:         "tank/vol2",
			Volsize:      2 * 1024 * 1024 * 1024,
			Volblocksize: "64K",
			Comments:     "test zvol",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid volsize"})
		}))
		_, err := c.CreateZvol(ctx, &client.ZvolCreateRequest{Name: "tank/v", Volsize: 0})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("Create 500 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "err"})
		}))
		_, err := c.CreateZvol(ctx, &client.ZvolCreateRequest{Name: "tank/v", Volsize: 1024})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("Create invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bogus"))
		}))
		_, err := c.CreateZvol(ctx, &client.ZvolCreateRequest{Name: "tank/v", Volsize: 1024})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Get delegates to GetDataset", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.Contains(r.URL.Path, "/pool/dataset/id/") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{
				ID:   "tank/vol1",
				Type: "VOLUME",
			})
		}))

		got, err := c.GetZvol(ctx, "tank/vol1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "tank/vol1" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("Get 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetZvol(ctx, "tank/missing")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bogus"))
		}))
		_, err := c.GetZvol(ctx, "tank/v")
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update success sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.Contains(r.URL.Path, "/pool/dataset/id/") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"compression":"zstd"`) {
				t.Errorf("body missing compression: %s", body)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{
				ID:   "tank/vol1",
				Type: "VOLUME",
			})
		}))

		got, err := c.UpdateZvol(ctx, "tank/vol1", &client.ZvolUpdateRequest{
			Compression: "zstd",
			Volsize:     4 * 1024 * 1024 * 1024,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "tank/vol1" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid"})
		}))
		_, err := c.UpdateZvol(ctx, "tank/vol1", &client.ZvolUpdateRequest{Volsize: 1})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("Update invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bogus"))
		}))
		_, err := c.UpdateZvol(ctx, "tank/vol1", &client.ZvolUpdateRequest{Volsize: 1})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update URL-encodes zvol id", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The raw path should contain the percent-encoded slash.
			if r.URL.RawPath != "" && !strings.Contains(r.URL.RawPath, "tank%2Fvol1") {
				t.Errorf("raw path = %s, want tank%%2Fvol1", r.URL.RawPath)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{ID: "tank/vol1", Type: "VOLUME"})
		}))
		_, err := c.UpdateZvol(ctx, "tank/vol1", &client.ZvolUpdateRequest{Compression: "lz4"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete delegates to DeleteDataset", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.Contains(r.URL.Path, "/pool/dataset/id/") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteZvol(ctx, "tank/vol1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteZvol(ctx, "tank/vol1")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
