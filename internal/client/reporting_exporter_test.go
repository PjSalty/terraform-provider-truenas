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

func TestReportingExporter_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/reporting/exporters/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"id":3,"enabled":true,"name":"graphite1","attributes":{"exporter_type":"GRAPHITE","destination":"graphite.local","port":2003}}`))
		}))
		got, err := c.GetReportingExporter(ctx, 3)
		if err != nil {
			t.Fatalf("GetReportingExporter: %v", err)
		}
		if got.ID != 3 || !got.Enabled || got.Name != "graphite1" {
			t.Errorf("got = %+v", got)
		}
		if !strings.Contains(string(got.Attributes), "GRAPHITE") {
			t.Errorf("attributes = %s", string(got.Attributes))
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetReportingExporter(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetReportingExporter(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 500 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetReportingExporter(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/reporting/exporters") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["enabled"] != true {
				t.Errorf("enabled = %v", raw["enabled"])
			}
			if raw["name"] != "newexp" {
				t.Errorf("name = %v", raw["name"])
			}
			attrs, ok := raw["attributes"].(map[string]interface{})
			if !ok {
				t.Fatalf("attributes missing: %T", raw["attributes"])
			}
			if attrs["exporter_type"] != "GRAPHITE" {
				t.Errorf("exporter_type = %v", attrs["exporter_type"])
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":      10,
				"enabled": true,
				"name":    "newexp",
			})
		}))
		got, err := c.CreateReportingExporter(ctx, &client.ReportingExporterCreateRequest{
			Enabled:    true,
			Name:       "newexp",
			Attributes: json.RawMessage(`{"exporter_type":"GRAPHITE","destination":"g.local"}`),
		})
		if err != nil {
			t.Fatalf("CreateReportingExporter: %v", err)
		}
		if got.ID != 10 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid exporter_type"})
		}))
		_, err := c.CreateReportingExporter(ctx, &client.ReportingExporterCreateRequest{
			Name:       "bad",
			Attributes: json.RawMessage(`{}`),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Create 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.CreateReportingExporter(ctx, &client.ReportingExporterCreateRequest{
			Name:       "x",
			Attributes: json.RawMessage(`{}`),
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.CreateReportingExporter(ctx, &client.ReportingExporterCreateRequest{
			Name:       "x",
			Attributes: json.RawMessage(`{}`),
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/reporting/exporters/id/10") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["name"] != "renamed" {
				t.Errorf("name = %v", raw["name"])
			}
			writeJSON(w, http.StatusOK, client.ReportingExporter{ID: 10, Name: "renamed", Enabled: false})
		}))
		name := "renamed"
		enabled := false
		got, err := c.UpdateReportingExporter(ctx, 10, &client.ReportingExporterUpdateRequest{
			Name:    &name,
			Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("UpdateReportingExporter: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update omits nil fields", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if _, has := raw["name"]; has {
				t.Errorf("name should be omitted when nil")
			}
			if raw["enabled"] != true {
				t.Errorf("enabled = %v", raw["enabled"])
			}
			writeJSON(w, http.StatusOK, client.ReportingExporter{ID: 10})
		}))
		enabled := true
		_, err := c.UpdateReportingExporter(ctx, 10, &client.ReportingExporterUpdateRequest{Enabled: &enabled})
		if err != nil {
			t.Fatalf("UpdateReportingExporter: %v", err)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		n := "x"
		_, err := c.UpdateReportingExporter(ctx, 10, &client.ReportingExporterUpdateRequest{Name: &n})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		n := "x"
		_, err := c.UpdateReportingExporter(ctx, 10, &client.ReportingExporterUpdateRequest{Name: &n})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/reporting/exporters/id/10") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteReportingExporter(ctx, 10); err != nil {
			t.Fatalf("DeleteReportingExporter: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteReportingExporter(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Delete 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		err := c.DeleteReportingExporter(ctx, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update attributes raw JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			attrs, ok := raw["attributes"].(map[string]interface{})
			if !ok {
				t.Fatalf("attributes = %T", raw["attributes"])
			}
			if attrs["port"] != float64(2004) {
				t.Errorf("port = %v", attrs["port"])
			}
			writeJSON(w, http.StatusOK, client.ReportingExporter{ID: 10})
		}))
		_, err := c.UpdateReportingExporter(ctx, 10, &client.ReportingExporterUpdateRequest{
			Attributes: json.RawMessage(`{"exporter_type":"GRAPHITE","port":2004}`),
		})
		if err != nil {
			t.Fatalf("UpdateReportingExporter: %v", err)
		}
	})
}
