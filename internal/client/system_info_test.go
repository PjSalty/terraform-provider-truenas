package client_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestSystemInfo_ReadOnly(t *testing.T) {
	ctx := context.Background()

	t.Run("GetSystemInfo success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/system/info") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.SystemInfo{
				Version: "TrueNAS-SCALE-25.04.2.6", Hostname: "truenas",
				PhysicalMem: 16 * 1024 * 1024 * 1024, Model: "Xeon", Cores: 8,
				Uptime: "10 days", UptimeSeconds: 864000.5,
				SystemSerial: "SER123", SystemProduct: "SUPERMICRO",
				Timezone: "UTC", Loadavg: []float64{0.1, 0.2, 0.3},
			})
		}))
		info, err := c.GetSystemInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Hostname != "truenas" {
			t.Errorf("Hostname = %q", info.Hostname)
		}
		if info.Cores != 8 {
			t.Errorf("Cores = %d", info.Cores)
		}
		if len(info.Loadavg) != 3 {
			t.Errorf("Loadavg len = %d", len(info.Loadavg))
		}
	})

	t.Run("GetSystemInfo 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetSystemInfo(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetSystemInfo server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetSystemInfo(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
		if apiErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("GetSystemInfo 503 service unavailable", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"message": "starting"})
		}))
		_, err := c.GetSystemInfo(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("GetSystemInfo 401 unauthorized", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "auth"})
		}))
		_, err := c.GetSystemInfo(ctx)
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

	t.Run("GetSystemInfo invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-json"))
		}))
		_, err := c.GetSystemInfo(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetSystemInfo empty response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{}"))
		}))
		info, err := c.GetSystemInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.Hostname != "" {
			t.Errorf("Hostname = %q, want empty", info.Hostname)
		}
	})

	t.Run("GetSystemInfo parses nested datetime", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"version":"x","hostname":"h","datetime":{"year":2026,"month":4,"day":12,"hour":15,"minute":30,"second":0,"timezone":"UTC"}}`))
		}))
		info, err := c.GetSystemInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.DateTime.Year != 2026 {
			t.Errorf("Year = %d", info.DateTime.Year)
		}
		if info.DateTime.TZ != "UTC" {
			t.Errorf("TZ = %q", info.DateTime.TZ)
		}
	})

	t.Run("GetSystemInfo parses physmem int64", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"physmem":68719476736,"hostname":"h"}`))
		}))
		info, err := c.GetSystemInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.PhysicalMem != 68719476736 {
			t.Errorf("PhysicalMem = %d", info.PhysicalMem)
		}
	})

	t.Run("GetSystemInfo authorization header sent", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				t.Errorf("Authorization header missing")
			}
			writeJSON(w, http.StatusOK, client.SystemInfo{Hostname: "x"})
		}))
		_, err := c.GetSystemInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("GetSystemInfo 400 bad request", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "bad"})
		}))
		_, err := c.GetSystemInfo(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
		if apiErr.StatusCode != http.StatusBadRequest {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("GetSystemInfo 403 forbidden", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusForbidden, map[string]string{"message": "no"})
		}))
		_, err := c.GetSystemInfo(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetSystemInfo zero loadavg", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"loadavg":[]}`))
		}))
		info, err := c.GetSystemInfo(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(info.Loadavg) != 0 {
			t.Errorf("Loadavg len = %d", len(info.Loadavg))
		}
	})
}
