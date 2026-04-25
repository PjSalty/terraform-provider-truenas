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

func TestSMBShare_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.SMBShare{
				ID:      7,
				Path:    "/mnt/tank/smb",
				Name:    "myshare",
				Enabled: true,
			})
		}))

		got, err := c.GetSMBShare(ctx, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "myshare" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "missing"})
		}))

		_, err := c.GetSMBShare(ctx, 404)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("List", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.SMBShare{
				{ID: 1, Name: "a"},
				{ID: 2, Name: "b"},
			})
		}))

		list, err := c.ListSMBShares(ctx)
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
			var req client.SMBShareCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "backups" {
				t.Errorf("Name = %q", req.Name)
			}
			writeJSON(w, http.StatusOK, client.SMBShare{
				ID:      10,
				Path:    req.Path,
				Name:    req.Name,
				Enabled: req.Enabled,
			})
		}))

		resp, err := c.CreateSMBShare(ctx, &client.SMBShareCreateRequest{
			Path:    "/mnt/tank/backups",
			Name:    "backups",
			Enabled: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 10 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Create error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "name in use"})
		}))

		_, err := c.CreateSMBShare(ctx, &client.SMBShareCreateRequest{Name: "dupe"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Errorf("err not *APIError")
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.SMBShare{
				ID:       5,
				Name:     "renamed",
				ReadOnly: true,
			})
		}))

		ro := true
		resp, err := c.UpdateSMBShare(ctx, 5, &client.SMBShareUpdateRequest{
			Name:     "renamed",
			ReadOnly: &ro,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.ReadOnly {
			t.Errorf("ReadOnly = false, want true")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteSMBShare(ctx, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
