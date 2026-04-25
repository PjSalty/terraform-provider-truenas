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

func TestDataset_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetDataset success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			// URL.Path is auto-decoded — the raw form is in URL.RawPath.
			if !strings.HasSuffix(r.URL.Path, "/pool/dataset/id/tank/child") {
				t.Errorf("path = %s, want .../pool/dataset/id/tank/child", r.URL.Path)
			}
			if r.URL.RawPath != "" && !strings.Contains(r.URL.RawPath, "tank%2Fchild") {
				t.Errorf("raw path = %s, want tank%%2Fchild", r.URL.RawPath)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{
				ID:         "tank/child",
				Name:       "child",
				Pool:       "tank",
				Type:       "FILESYSTEM",
				MountPoint: "/mnt/tank/child",
			})
		}))

		got, err := c.GetDataset(ctx, "tank/child")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "tank/child" {
			t.Errorf("ID = %q", got.ID)
		}
		if got.Pool != "tank" {
			t.Errorf("Pool = %q", got.Pool)
		}
	})

	t.Run("GetDataset 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))

		_, err := c.GetDataset(ctx, "tank/missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true (err=%v)", err)
		}
	})

	t.Run("GetDataset invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))

		_, err := c.GetDataset(ctx, "tank/x")
		if err == nil {
			t.Fatal("expected JSON parse error, got nil")
		}
	})

	t.Run("CreateDataset sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.DatasetCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
			if req.Name != "tank/newds" {
				t.Errorf("Name = %q, want tank/newds", req.Name)
			}
			if req.Compression != "lz4" {
				t.Errorf("Compression = %q, want lz4", req.Compression)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{
				ID:         req.Name,
				Type:       "FILESYSTEM",
				MountPoint: "/mnt/" + req.Name,
			})
		}))

		resp, err := c.CreateDataset(ctx, &client.DatasetCreateRequest{
			Name:        "tank/newds",
			Type:        "FILESYSTEM",
			Compression: "lz4",
			Quota:       1024 * 1024 * 1024,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != "tank/newds" {
			t.Errorf("ID = %q", resp.ID)
		}
	})

	t.Run("CreateDataset server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad input"})
		}))

		_, err := c.CreateDataset(ctx, &client.DatasetCreateRequest{Name: "tank/bad"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err is not *APIError: %T", err)
		}
		if apiErr.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("UpdateDataset success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			if !strings.Contains(r.URL.Path, "/pool/dataset/id/tank/updated") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.DatasetUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding body: %v", err)
			}
			if req.Compression != "zstd" {
				t.Errorf("Compression = %q, want zstd", req.Compression)
			}
			writeJSON(w, http.StatusOK, client.DatasetResponse{
				ID:   "tank/updated",
				Type: "FILESYSTEM",
			})
		}))

		resp, err := c.UpdateDataset(ctx, "tank/updated", &client.DatasetUpdateRequest{
			Compression: "zstd",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != "tank/updated" {
			t.Errorf("ID = %q", resp.ID)
		}
	})

	t.Run("DeleteDataset success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s, want DELETE", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteDataset(ctx, "tank/drop"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteDataset 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))

		err := c.DeleteDataset(ctx, "tank/gone")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true (err=%v)", err)
		}
	})

	t.Run("ListDatasets success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.DatasetResponse{
				{ID: "tank", Type: "FILESYSTEM"},
				{ID: "tank/a", Type: "FILESYSTEM"},
				{ID: "tank/b", Type: "FILESYSTEM"},
			})
		}))

		list, err := c.ListDatasets(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 3 {
			t.Errorf("len = %d, want 3", len(list))
		}
	})
}

func TestDatasetResponse_GetComments(t *testing.T) {
	cases := []struct {
		name string
		resp client.DatasetResponse
		want string
	}{
		{
			name: "empty",
			resp: client.DatasetResponse{},
			want: "",
		},
		{
			name: "legacy 25.04 top-level",
			resp: client.DatasetResponse{
				Comments: &client.PropertyValue{Value: "legacy comment"},
			},
			want: "legacy comment",
		},
		{
			name: "25.10 user_properties",
			resp: client.DatasetResponse{
				UserProperties: map[string]*client.PropertyValue{
					"comments": {Value: "new comment"},
				},
			},
			want: "new comment",
		},
		{
			name: "user_properties preferred over top-level",
			resp: client.DatasetResponse{
				Comments: &client.PropertyValue{Value: "legacy"},
				UserProperties: map[string]*client.PropertyValue{
					"comments": {Value: "new"},
				},
			},
			want: "new",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.resp.GetComments()
			if got != tc.want {
				t.Errorf("GetComments() = %q, want %q", got, tc.want)
			}
		})
	}
}
