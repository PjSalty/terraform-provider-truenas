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

func TestSMBConfig_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetSMBConfig success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/smb") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{
				ID: 1, NetbiosName: "TRUENAS", Workgroup: "WORKGROUP",
				Description: "TrueNAS", EnableSMB1: false,
				UnixCharset: "UTF-8", AAPLExtensions: false, Guest: "nobody",
				Filemask: "0644", Dirmask: "0755",
			})
		}))
		cfg, err := c.GetSMBConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.NetbiosName != "TRUENAS" {
			t.Errorf("NetbiosName = %q", cfg.NetbiosName)
		}
		if cfg.Workgroup != "WORKGROUP" {
			t.Errorf("Workgroup = %q", cfg.Workgroup)
		}
	})

	t.Run("GetSMBConfig 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetSMBConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetSMBConfig server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetSMBConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("GetSMBConfig invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetSMBConfig(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetSMBConfig 401 unauthorized", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "no"})
		}))
		_, err := c.GetSMBConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
		if apiErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("UpdateSMBConfig sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/smb") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.SMBConfigUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.NetbiosName == nil || *req.NetbiosName != "NEWNAME" {
				t.Errorf("NetbiosName = %v", req.NetbiosName)
			}
			if req.Workgroup == nil || *req.Workgroup != "DOMAIN" {
				t.Errorf("Workgroup = %v", req.Workgroup)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{
				ID: 1, NetbiosName: *req.NetbiosName, Workgroup: *req.Workgroup,
			})
		}))
		name := "NEWNAME"
		wg := "DOMAIN"
		resp, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{
			NetbiosName: &name, Workgroup: &wg,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.NetbiosName != "NEWNAME" {
			t.Errorf("NetbiosName = %q", resp.NetbiosName)
		}
	})

	t.Run("UpdateSMBConfig with EnableSMB1 pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.SMBConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.EnableSMB1 == nil || *req.EnableSMB1 != true {
				t.Errorf("EnableSMB1 = %v, want true", req.EnableSMB1)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{ID: 1, EnableSMB1: true})
		}))
		enable := true
		resp, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{EnableSMB1: &enable})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.EnableSMB1 {
			t.Errorf("EnableSMB1 = false, want true")
		}
	})

	t.Run("UpdateSMBConfig with AAPLExtensions pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.SMBConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.AAPLExtensions == nil || *req.AAPLExtensions != true {
				t.Errorf("AAPLExtensions = %v", req.AAPLExtensions)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{ID: 1, AAPLExtensions: true})
		}))
		aapl := true
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{AAPLExtensions: &aapl})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpdateSMBConfig with filemask and dirmask", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.SMBConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.Filemask == nil || *req.Filemask != "0600" {
				t.Errorf("Filemask = %v", req.Filemask)
			}
			if req.Dirmask == nil || *req.Dirmask != "0700" {
				t.Errorf("Dirmask = %v", req.Dirmask)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{
				ID: 1, Filemask: *req.Filemask, Dirmask: *req.Dirmask,
			})
		}))
		fm := "0600"
		dm := "0700"
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{Filemask: &fm, Dirmask: &dm})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpdateSMBConfig 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		name := "bad"
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{NetbiosName: &name})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
		if apiErr.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("UpdateSMBConfig 500 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("UpdateSMBConfig invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateSMBConfig with Guest pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.SMBConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.Guest == nil || *req.Guest != "nobody" {
				t.Errorf("Guest = %v", req.Guest)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{ID: 1, Guest: "nobody"})
		}))
		g := "nobody"
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{Guest: &g})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpdateSMBConfig with UnixCharset", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.SMBConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.UnixCharset == nil || *req.UnixCharset != "UTF-8" {
				t.Errorf("UnixCharset = %v", req.UnixCharset)
			}
			writeJSON(w, http.StatusOK, client.SMBConfig{ID: 1, UnixCharset: "UTF-8"})
		}))
		cs := "UTF-8"
		_, err := c.UpdateSMBConfig(ctx, &client.SMBConfigUpdateRequest{UnixCharset: &cs})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
