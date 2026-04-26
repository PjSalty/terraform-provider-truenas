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

func TestAlertClasses_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetAlertClassesConfig success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/alertclasses") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.AlertClassesConfig{
				ID: 1,
				Classes: map[string]client.AlertClassEntry{
					"PoolDegraded": {Level: "CRITICAL", Policy: "IMMEDIATELY"},
				},
			})
		}))
		cfg, err := c.GetAlertClassesConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.ID != 1 {
			t.Errorf("ID = %d", cfg.ID)
		}
		entry, ok := cfg.Classes["PoolDegraded"]
		if !ok {
			t.Fatalf("PoolDegraded entry missing")
		}
		if entry.Level != "CRITICAL" {
			t.Errorf("Level = %q", entry.Level)
		}
	})

	t.Run("GetAlertClassesConfig null classes normalized", func(t *testing.T) {
		// When the API returns classes: null, GetAlertClassesConfig should
		// initialize an empty (non-nil) map.
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1,"classes":null}`))
		}))
		cfg, err := c.GetAlertClassesConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Classes == nil {
			t.Errorf("Classes = nil, want empty map")
		}
		if len(cfg.Classes) != 0 {
			t.Errorf("len = %d, want 0", len(cfg.Classes))
		}
	})

	t.Run("GetAlertClassesConfig 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetAlertClassesConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetAlertClassesConfig server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetAlertClassesConfig(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("GetAlertClassesConfig invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.GetAlertClassesConfig(ctx)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateAlertClassesConfig sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/alertclasses") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.AlertClassesUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			entry, ok := req.Classes["ZpoolCapacityCritical"]
			if !ok {
				t.Errorf("ZpoolCapacityCritical entry missing")
			}
			if entry.Level != "CRITICAL" {
				t.Errorf("Level = %q", entry.Level)
			}
			writeJSON(w, http.StatusOK, client.AlertClassesConfig{
				ID:      1,
				Classes: req.Classes,
			})
		}))
		resp, err := c.UpdateAlertClassesConfig(ctx, &client.AlertClassesUpdateRequest{
			Classes: map[string]client.AlertClassEntry{
				"ZpoolCapacityCritical": {Level: "CRITICAL", Policy: "IMMEDIATELY"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := resp.Classes["ZpoolCapacityCritical"]; !ok {
			t.Errorf("response missing entry")
		}
	})

	t.Run("UpdateAlertClassesConfig null classes normalized", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1,"classes":null}`))
		}))
		resp, err := c.UpdateAlertClassesConfig(ctx, &client.AlertClassesUpdateRequest{
			Classes: map[string]client.AlertClassEntry{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Classes == nil {
			t.Errorf("Classes = nil, want empty map")
		}
	})

	t.Run("UpdateAlertClassesConfig 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad class"})
		}))
		_, err := c.UpdateAlertClassesConfig(ctx, &client.AlertClassesUpdateRequest{
			Classes: map[string]client.AlertClassEntry{"bad": {}},
		})
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

	t.Run("UpdateAlertClassesConfig invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.UpdateAlertClassesConfig(ctx, &client.AlertClassesUpdateRequest{
			Classes: map[string]client.AlertClassEntry{},
		})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateAlertClassesConfig 500 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.UpdateAlertClassesConfig(ctx, &client.AlertClassesUpdateRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetAlertClassesConfig with proactive_support pointer", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":1,"classes":{"X":{"level":"WARNING","policy":"HOURLY","proactive_support":true}}}`))
		}))
		cfg, err := c.GetAlertClassesConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entry := cfg.Classes["X"]
		if entry.ProactiveSupport == nil {
			t.Fatalf("ProactiveSupport = nil, want non-nil")
		}
		if !*entry.ProactiveSupport {
			t.Errorf("ProactiveSupport = false, want true")
		}
	})

	t.Run("GetAlertClassesConfig empty classes map", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.AlertClassesConfig{ID: 1, Classes: map[string]client.AlertClassEntry{}})
		}))
		cfg, err := c.GetAlertClassesConfig(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Classes) != 0 {
			t.Errorf("len = %d, want 0", len(cfg.Classes))
		}
	})
}
