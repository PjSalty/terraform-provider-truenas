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

func TestVM_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("ListVMs success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.VM{
				{ID: 1, Name: "alpha", Memory: 2048},
				{ID: 2, Name: "beta", Memory: 4096},
			})
		}))
		list, err := c.ListVMs(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("ListVMs invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{"))
		}))
		_, err := c.ListVMs(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetVM success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/vm/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.VM{
				ID: 5, Name: "gamma", Memory: 8192, Vcpus: 2, Cores: 2,
			})
		}))
		vm, err := c.GetVM(ctx, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if vm.Name != "gamma" {
			t.Errorf("Name = %q", vm.Name)
		}
	})

	t.Run("GetVM 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetVM(ctx, 999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetVM invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetVM(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("CreateVM sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.VMCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Name != "testvm" {
				t.Errorf("Name = %q", req.Name)
			}
			if req.Memory != 4096 {
				t.Errorf("Memory = %d, want 4096", req.Memory)
			}
			writeJSON(w, http.StatusOK, client.VM{ID: 10, Name: req.Name, Memory: req.Memory})
		}))
		resp, err := c.CreateVM(ctx, &client.VMCreateRequest{
			Name: "testvm", Memory: 4096,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 10 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("CreateVM server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad config"})
		}))
		_, err := c.CreateVM(ctx, &client.VMCreateRequest{Name: "x", Memory: 1024})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("CreateVM invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := c.CreateVM(ctx, &client.VMCreateRequest{Name: "x", Memory: 1024})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateVM sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.VMUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Name == nil || *req.Name != "renamed" {
				t.Errorf("Name = %v", req.Name)
			}
			if req.Memory == nil || *req.Memory != 8192 {
				t.Errorf("Memory = %v", req.Memory)
			}
			writeJSON(w, http.StatusOK, client.VM{ID: 7, Name: "renamed", Memory: 8192})
		}))
		name := "renamed"
		var mem int64 = 8192
		resp, err := c.UpdateVM(ctx, 7, &client.VMUpdateRequest{Name: &name, Memory: &mem})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "renamed" {
			t.Errorf("Name = %q", resp.Name)
		}
	})

	t.Run("UpdateVM server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateVM(ctx, 1, &client.VMUpdateRequest{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("DeleteVM with nil opts defaults to force true", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm/id/11") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var opts client.VMDeleteOptions
			if err := json.Unmarshal(body, &opts); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if !opts.Force {
				t.Errorf("Force = false, want true (default)")
			}
			if opts.Zvols {
				t.Errorf("Zvols = true, want false (default)")
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteVM(ctx, 11, nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteVM with explicit opts", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var opts client.VMDeleteOptions
			_ = json.Unmarshal(body, &opts)
			if !opts.Zvols {
				t.Errorf("Zvols = false, want true")
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteVM(ctx, 1, &client.VMDeleteOptions{Zvols: true, Force: false}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteVM 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteVM(ctx, 1, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("StartVM success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm/id/3/start") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`null`))
		}))
		if err := c.StartVM(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("StopVM with force true", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm/id/3/stop") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			_ = json.Unmarshal(body, &req)
			if force, ok := req["force"].(bool); !ok || !force {
				t.Errorf("force = %v, want true", req["force"])
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`null`))
		}))
		if err := c.StopVM(ctx, 3, true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("StopVM server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "no"})
		}))
		err := c.StopVM(ctx, 1, false)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	// --- VM Device tests ---

	t.Run("GetVMDevice success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/vm/device/id/22") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.VMDevice{
				ID: 22, VM: 5,
				Attributes: map[string]interface{}{"dtype": "NIC"},
			})
		}))
		dev, err := c.GetVMDevice(ctx, 22)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dev.VM != 5 {
			t.Errorf("VM = %d", dev.VM)
		}
	})

	t.Run("GetVMDevice 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetVMDevice(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("CreateVMDevice sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm/device") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.VMDeviceCreateRequest
			_ = json.Unmarshal(body, &req)
			if req.VM != 5 {
				t.Errorf("VM = %d", req.VM)
			}
			writeJSON(w, http.StatusOK, client.VMDevice{ID: 30, VM: req.VM, Attributes: req.Attributes})
		}))
		resp, err := c.CreateVMDevice(ctx, &client.VMDeviceCreateRequest{
			VM: 5, Attributes: map[string]interface{}{"dtype": "DISK"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 30 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("UpdateVMDevice success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/vm/device/id/30") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.VMDevice{ID: 30, VM: 5})
		}))
		order := 5
		resp, err := c.UpdateVMDevice(ctx, 30, &client.VMDeviceUpdateRequest{Order: &order})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 30 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("DeleteVMDevice success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteVMDevice(ctx, 30); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteVMDevice 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteVMDevice(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})
}
