package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestNFSShare_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusOK, client.NFSShare{
				ID:      11,
				Path:    "/mnt/tank/nfs",
				Enabled: true,
				Hosts:   []string{"10.0.0.0/24"},
			})
		}))

		got, err := c.GetNFSShare(ctx, 11)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 11 || got.Path != "/mnt/tank/nfs" {
			t.Errorf("unexpected share: %+v", got)
		}
		if len(got.Hosts) != 1 || got.Hosts[0] != "10.0.0.0/24" {
			t.Errorf("Hosts = %v", got.Hosts)
		}
	})

	t.Run("Get 422 does not exist -> IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"message": "Object with id=99 does not exist",
			})
		}))

		_, err := c.GetNFSShare(ctx, 99)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true (err=%v)", err)
		}
	})

	t.Run("List", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.NFSShare{
				{ID: 1, Path: "/mnt/tank/a"},
				{ID: 2, Path: "/mnt/tank/b"},
			})
		}))

		list, err := c.ListNFSShares(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.NFSShareCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Path != "/mnt/tank/new" {
				t.Errorf("Path = %q", req.Path)
			}
			writeJSON(w, http.StatusOK, client.NFSShare{
				ID:      42,
				Path:    req.Path,
				Enabled: req.Enabled,
			})
		}))

		resp, err := c.CreateNFSShare(ctx, &client.NFSShareCreateRequest{
			Path:    "/mnt/tank/new",
			Enabled: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 42 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid path"})
		}))

		_, err := c.CreateNFSShare(ctx, &client.NFSShareCreateRequest{Path: "bad"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err is not *APIError: %T", err)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusOK, client.NFSShare{
				ID:      5,
				Path:    "/mnt/tank/shared",
				Enabled: false,
			})
		}))

		enabled := false
		resp, err := c.UpdateNFSShare(ctx, 5, &client.NFSShareUpdateRequest{
			Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Enabled {
			t.Errorf("Enabled = true, want false")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteNFSShare(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))

		err := c.DeleteNFSShare(ctx, 99)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true (err=%v)", err)
		}
	})
}
