package client_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestDisk_ReadOnly(t *testing.T) {
	ctx := context.Background()

	t.Run("ListDisks success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/disk") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.Disk{
				{Identifier: "{serial}ABC123", Name: "sda", Size: 1000000000000, Type: "HDD", Model: "WDC"},
				{Identifier: "{serial}XYZ999", Name: "sdb", Size: 500000000000, Type: "SSD", Model: "Samsung"},
			})
		}))
		list, err := c.ListDisks(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d, want 2", len(list))
		}
		if list[0].Name != "sda" {
			t.Errorf("Name = %q", list[0].Name)
		}
	})

	t.Run("ListDisks server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.ListDisks(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("ListDisks invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<not json>"))
		}))
		_, err := c.ListDisks(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("ListDisks empty list", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Disk{})
		}))
		list, err := c.ListDisks(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 0 {
			t.Errorf("len = %d, want 0", len(list))
		}
	})

	t.Run("GetDisk success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			// Path has been URL-decoded by the time it reaches the handler.
			if !strings.HasSuffix(r.URL.Path, "/disk/id/{serial}ABC123") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Disk{
				Identifier: "{serial}ABC123", Name: "sda", Size: 1000000000000,
				Type: "HDD", Model: "WDC", Serial: "ABC123",
			})
		}))
		d, err := c.GetDisk(ctx, "{serial}ABC123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Serial != "ABC123" {
			t.Errorf("Serial = %q", d.Serial)
		}
	})

	t.Run("GetDisk escapes special characters in path", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The braces and slashes must be percent-encoded in the raw path.
			// url.PathEscape escapes '{','}','/' — check the raw path contains encoded form.
			if r.URL.RawPath == "" {
				// some stdlib builds only populate RawPath when it differs.
				// Fall back to reconstructed EscapedPath.
				if !strings.Contains(r.URL.EscapedPath(), "%7B") &&
					!strings.Contains(r.URL.EscapedPath(), "%2F") {
					t.Errorf("EscapedPath = %s, want percent-encoded", r.URL.EscapedPath())
				}
			} else if !strings.Contains(r.URL.RawPath, "%7B") && !strings.Contains(r.URL.RawPath, "%2F") {
				t.Errorf("RawPath = %s, want percent-encoded", r.URL.RawPath)
			}
			writeJSON(w, http.StatusOK, client.Disk{Identifier: "{serial_lunid}X/Y", Name: "sdc"})
		}))
		_, err := c.GetDisk(ctx, "{serial_lunid}X/Y")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("GetDisk 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetDisk(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetDisk server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "oops"})
		}))
		_, err := c.GetDisk(ctx, "abc")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("GetDisk invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetDisk(ctx, "abc")
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetDisk pool pointer populated", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"identifier":"x","name":"sda","pool":"tank","zfs_guid":"12345"}`))
		}))
		d, err := c.GetDisk(ctx, "x")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Pool == nil || *d.Pool != "tank" {
			t.Errorf("Pool = %v, want tank", d.Pool)
		}
		if d.ZFSGuid == nil || *d.ZFSGuid != "12345" {
			t.Errorf("ZFSGuid = %v, want 12345", d.ZFSGuid)
		}
	})

	t.Run("GetDisk unassigned pool is nil", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"identifier":"y","name":"sdb","pool":null,"zfs_guid":null}`))
		}))
		d, err := c.GetDisk(ctx, "y")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if d.Pool != nil {
			t.Errorf("Pool = %v, want nil", d.Pool)
		}
	})

	t.Run("GetDisk empty identifier", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/disk/id/") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetDisk(ctx, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
