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

func TestNFSConfig_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetNFSConfig success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/nfs") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.NFSConfig{
				ID: 1, Servers: 4, AllowNonroot: false,
				Protocols: []string{"NFSV3", "NFSV4"},
				V4Krb:     false, V4Domain: "example.com",
				BindIP: []string{"0.0.0.0"},
			})
		}))
		cfg, err := c.GetNFSConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Servers != 4 {
			t.Errorf("Servers = %d", cfg.Servers)
		}
		if len(cfg.Protocols) != 2 {
			t.Errorf("Protocols len = %d", len(cfg.Protocols))
		}
	})

	t.Run("GetNFSConfig 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetNFSConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetNFSConfig server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetNFSConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("GetNFSConfig invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := c.GetNFSConfig(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetNFSConfig with port pointers", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1,"mountd_port":618,"rpcstatd_port":619,"rpclockd_port":620}`))
		}))
		cfg, err := c.GetNFSConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.MountdPort == nil || *cfg.MountdPort != 618 {
			t.Errorf("MountdPort = %v", cfg.MountdPort)
		}
		if cfg.RpcstatdPort == nil || *cfg.RpcstatdPort != 619 {
			t.Errorf("RpcstatdPort = %v", cfg.RpcstatdPort)
		}
	})

	t.Run("UpdateNFSConfig sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/nfs") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.NFSConfigUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Servers == nil || *req.Servers != 8 {
				t.Errorf("Servers = %v, want 8", req.Servers)
			}
			if len(req.Protocols) != 1 || req.Protocols[0] != "NFSV4" {
				t.Errorf("Protocols = %v", req.Protocols)
			}
			writeJSON(w, http.StatusOK, client.NFSConfig{
				ID: 1, Servers: *req.Servers, Protocols: req.Protocols,
			})
		}))
		servers := 8
		resp, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{
			Servers:   &servers,
			Protocols: []string{"NFSV4"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Servers != 8 {
			t.Errorf("Servers = %d", resp.Servers)
		}
	})

	t.Run("UpdateNFSConfig 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad config"})
		}))
		servers := 0
		_, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{Servers: &servers})
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

	t.Run("UpdateNFSConfig server 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("UpdateNFSConfig invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateNFSConfig with AllowNonroot pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.NFSConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.AllowNonroot == nil || *req.AllowNonroot != true {
				t.Errorf("AllowNonroot = %v, want true", req.AllowNonroot)
			}
			writeJSON(w, http.StatusOK, client.NFSConfig{ID: 1, AllowNonroot: true})
		}))
		allow := true
		resp, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{AllowNonroot: &allow})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.AllowNonroot {
			t.Errorf("AllowNonroot = false, want true")
		}
	})

	t.Run("UpdateNFSConfig with V4Krb pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.NFSConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.V4Krb == nil || *req.V4Krb != true {
				t.Errorf("V4Krb = %v", req.V4Krb)
			}
			writeJSON(w, http.StatusOK, client.NFSConfig{ID: 1, V4Krb: true})
		}))
		krb := true
		_, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{V4Krb: &krb})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpdateNFSConfig BindIP list", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.NFSConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if len(req.BindIP) != 2 {
				t.Errorf("BindIP len = %d", len(req.BindIP))
			}
			writeJSON(w, http.StatusOK, client.NFSConfig{ID: 1, BindIP: req.BindIP})
		}))
		_, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{
			BindIP: []string{"10.0.0.1", "10.0.0.2"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpdateNFSConfig with port pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.NFSConfigUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.MountdPort == nil || *req.MountdPort != 618 {
				t.Errorf("MountdPort = %v", req.MountdPort)
			}
			writeJSON(w, http.StatusOK, client.NFSConfig{ID: 1})
		}))
		port := 618
		_, err := c.UpdateNFSConfig(ctx, &client.NFSConfigUpdateRequest{MountdPort: &port})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
