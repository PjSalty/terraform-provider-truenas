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

func TestVMware_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vmware/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.VMware{
				ID:         3,
				Datastore:  "ds1",
				Filesystem: "tank/vmware",
				Hostname:   "vcenter.example.com",
				Username:   "administrator@vsphere.local",
				Password:   "secret",
			})
		}))

		got, err := c.GetVMware(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Hostname != "vcenter.example.com" {
			t.Errorf("Hostname = %q", got.Hostname)
		}
		if got.Datastore != "ds1" {
			t.Errorf("Datastore = %q", got.Datastore)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetVMware(ctx, 99)
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
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := c.GetVMware(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Create sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vmware") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded["datastore"].(string) != "ds1" {
				t.Errorf("datastore = %v", decoded["datastore"])
			}
			if decoded["hostname"].(string) != "vc.example.com" {
				t.Errorf("hostname = %v", decoded["hostname"])
			}
			if decoded["password"].(string) != "secret" {
				t.Errorf("password = %v", decoded["password"])
			}
			writeJSON(w, http.StatusOK, client.VMware{
				ID:         17,
				Datastore:  "ds1",
				Filesystem: "tank/vmware",
				Hostname:   "vc.example.com",
				Username:   "admin",
				Password:   "secret",
			})
		}))

		got, err := c.CreateVMware(ctx, &client.VMwareCreateRequest{
			Datastore:  "ds1",
			Filesystem: "tank/vmware",
			Hostname:   "vc.example.com",
			Username:   "admin",
			Password:   "secret",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 17 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad cred"})
		}))
		_, err := c.CreateVMware(ctx, &client.VMwareCreateRequest{Hostname: "h", Username: "u", Password: "p", Datastore: "d", Filesystem: "f"})
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "err"})
		}))
		_, err := c.CreateVMware(ctx, &client.VMwareCreateRequest{Hostname: "h"})
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
			_, _ = w.Write([]byte("bogus"))
		}))
		_, err := c.CreateVMware(ctx, &client.VMwareCreateRequest{Hostname: "h"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update sends body with pointers", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vmware/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"hostname":"new.example.com"`) {
				t.Errorf("body missing hostname: %s", body)
			}
			writeJSON(w, http.StatusOK, client.VMware{ID: 5, Hostname: "new.example.com"})
		}))

		newHost := "new.example.com"
		got, err := c.UpdateVMware(ctx, 5, &client.VMwareUpdateRequest{
			Hostname: &newHost,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Hostname != "new.example.com" {
			t.Errorf("Hostname = %q", got.Hostname)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "nope"})
		}))
		h := "h"
		_, err := c.UpdateVMware(ctx, 5, &client.VMwareUpdateRequest{Hostname: &h})
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
		h := "h"
		_, err := c.UpdateVMware(ctx, 5, &client.VMwareUpdateRequest{Hostname: &h})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vmware/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteVMware(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteVMware(ctx, 3)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
