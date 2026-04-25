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

func TestISCSIInitiator_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/initiator/id/2") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.ISCSIInitiator{
				ID:         2,
				Initiators: []string{"iqn.1992-08.com.netapp:sn.1", "iqn.1998-01.com.vmware:esxi01"},
				Comment:    "vmware hosts",
			})
		}))

		got, err := c.GetISCSIInitiator(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 2 || len(got.Initiators) != 2 || got.Comment != "vmware hosts" {
			t.Errorf("unexpected: %+v", got)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "missing"})
		}))
		_, err := c.GetISCSIInitiator(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := c.GetISCSIInitiator(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Create success sends initiators array", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/initiator") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("body decode: %v", err)
			}
			arr, _ := decoded["initiators"].([]interface{})
			if len(arr) != 2 {
				t.Errorf("initiators len = %d", len(arr))
			}
			if decoded["comment"].(string) != "test" {
				t.Errorf("comment = %v", decoded["comment"])
			}
			writeJSON(w, http.StatusOK, client.ISCSIInitiator{
				ID:         5,
				Initiators: []string{"iqn.a", "iqn.b"},
				Comment:    "test",
			})
		}))

		got, err := c.CreateISCSIInitiator(ctx, &client.ISCSIInitiatorCreateRequest{
			Initiators: []string{"iqn.a", "iqn.b"},
			Comment:    "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 5 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid iqn"})
		}))
		_, err := c.CreateISCSIInitiator(ctx, &client.ISCSIInitiatorCreateRequest{Comment: "bad"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "server"})
		}))
		_, err := c.CreateISCSIInitiator(ctx, &client.ISCSIInitiatorCreateRequest{Comment: "x"})
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
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.CreateISCSIInitiator(ctx, &client.ISCSIInitiatorCreateRequest{Comment: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update success sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/initiator/id/9") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"comment":"updated"`) {
				t.Errorf("body missing comment: %s", body)
			}
			writeJSON(w, http.StatusOK, client.ISCSIInitiator{ID: 9, Comment: "updated"})
		}))

		got, err := c.UpdateISCSIInitiator(ctx, 9, &client.ISCSIInitiatorUpdateRequest{Comment: "updated"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Comment != "updated" {
			t.Errorf("Comment = %q", got.Comment)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateISCSIInitiator(ctx, 9, &client.ISCSIInitiatorUpdateRequest{Comment: "x"})
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
		_, err := c.UpdateISCSIInitiator(ctx, 9, &client.ISCSIInitiatorUpdateRequest{Comment: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/initiator/id/4") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteISCSIInitiator(ctx, 4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteISCSIInitiator(ctx, 4)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
