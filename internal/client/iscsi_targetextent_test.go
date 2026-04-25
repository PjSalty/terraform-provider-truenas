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

func TestISCSITargetExtent_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/targetextent/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.ISCSITargetExtent{
				ID: 3, Target: 10, Extent: 20, LunID: 0,
			})
		}))

		got, err := c.GetISCSITargetExtent(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 3 || got.Target != 10 || got.Extent != 20 {
			t.Errorf("unexpected: %+v", got)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetISCSITargetExtent(ctx, 99)
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
			_, _ = w.Write([]byte("xx"))
		}))
		_, err := c.GetISCSITargetExtent(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Create sends target/extent/lunid", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/targetextent") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if decoded["target"].(float64) != 1 {
				t.Errorf("target = %v", decoded["target"])
			}
			if decoded["extent"].(float64) != 2 {
				t.Errorf("extent = %v", decoded["extent"])
			}
			if decoded["lunid"].(float64) != 5 {
				t.Errorf("lunid = %v", decoded["lunid"])
			}
			writeJSON(w, http.StatusOK, client.ISCSITargetExtent{ID: 42, Target: 1, Extent: 2, LunID: 5})
		}))

		lun := 5
		got, err := c.CreateISCSITargetExtent(ctx, &client.ISCSITargetExtentCreateRequest{
			Target: 1, Extent: 2, LunID: &lun,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 42 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create omits nil lunid", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "lunid") {
				t.Errorf("body should not contain lunid: %s", body)
			}
			writeJSON(w, http.StatusOK, client.ISCSITargetExtent{ID: 1, Target: 1, Extent: 2})
		}))

		_, err := c.CreateISCSITargetExtent(ctx, &client.ISCSITargetExtentCreateRequest{
			Target: 1, Extent: 2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "dup"})
		}))
		_, err := c.CreateISCSITargetExtent(ctx, &client.ISCSITargetExtentCreateRequest{Target: 1, Extent: 2})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
		if apiErr.StatusCode != 422 {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("Create 500 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "bad"})
		}))
		_, err := c.CreateISCSITargetExtent(ctx, &client.ISCSITargetExtentCreateRequest{Target: 1, Extent: 2})
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
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := c.CreateISCSITargetExtent(ctx, &client.ISCSITargetExtentCreateRequest{Target: 1, Extent: 2})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/targetextent/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"target":5`) {
				t.Errorf("body missing target: %s", body)
			}
			writeJSON(w, http.StatusOK, client.ISCSITargetExtent{ID: 7, Target: 5, Extent: 6})
		}))

		lun := 2
		got, err := c.UpdateISCSITargetExtent(ctx, 7, &client.ISCSITargetExtentUpdateRequest{
			Target: 5, Extent: 6, LunID: &lun,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Target != 5 {
			t.Errorf("Target = %d", got.Target)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateISCSITargetExtent(ctx, 7, &client.ISCSITargetExtentUpdateRequest{Target: 1, Extent: 2})
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
		_, err := c.UpdateISCSITargetExtent(ctx, 7, &client.ISCSITargetExtentUpdateRequest{Target: 1, Extent: 2})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/targetextent/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteISCSITargetExtent(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteISCSITargetExtent(ctx, 3)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
