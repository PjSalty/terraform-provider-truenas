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

// networkInterfaceRouter routes requests across /interface, /interface/id/X,
// /interface/commit, /interface/checkin so CreateInterface / UpdateInterface /
// DeleteInterface (which internally call commit+checkin+re-Get) can be tested
// against a single handler.
type networkInterfaceRouter struct {
	// onCreate returns the interface object for POST /interface
	onCreate func(r *http.Request) (client.NetworkInterface, int)
	// onUpdate returns the interface object for PUT /interface/id/{id}
	onUpdate func(r *http.Request) (client.NetworkInterface, int)
	// onGet returns the interface object for GET /interface/id/{id}
	onGet func(r *http.Request) (client.NetworkInterface, int)
	// onDelete handles DELETE /interface/id/{id}
	onDelete func(r *http.Request) int
	// failCommit / failCheckin flip commit or checkin to 500
	failCommit  bool
	failCheckin bool
	t           *testing.T
}

func (h *networkInterfaceRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case strings.HasSuffix(path, "/interface/commit") && r.Method == http.MethodPost:
		if h.failCommit {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "commit failed"})
			return
		}
		writeJSON(w, http.StatusOK, nil)
	case strings.HasSuffix(path, "/interface/checkin") && r.Method == http.MethodGet:
		if h.failCheckin {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "checkin failed"})
			return
		}
		writeJSON(w, http.StatusOK, nil)
	case strings.Contains(path, "/interface/id/") && r.Method == http.MethodGet:
		if h.onGet == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "no onGet"})
			return
		}
		obj, status := h.onGet(r)
		writeJSON(w, status, obj)
	case strings.Contains(path, "/interface/id/") && r.Method == http.MethodPut:
		if h.onUpdate == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "no onUpdate"})
			return
		}
		obj, status := h.onUpdate(r)
		writeJSON(w, status, obj)
	case strings.Contains(path, "/interface/id/") && r.Method == http.MethodDelete:
		if h.onDelete == nil {
			writeJSON(w, http.StatusOK, nil)
			return
		}
		status := h.onDelete(r)
		w.WriteHeader(status)
	case strings.HasSuffix(path, "/interface") && r.Method == http.MethodPost:
		if h.onCreate == nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "no onCreate"})
			return
		}
		obj, status := h.onCreate(r)
		writeJSON(w, status, obj)
	case strings.HasSuffix(path, "/interface") && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, []client.NetworkInterface{})
	default:
		h.t.Errorf("unexpected request: %s %s", r.Method, path)
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "router: no route"})
	}
}

func TestNetworkInterface_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onGet: func(r *http.Request) (client.NetworkInterface, int) {
				if !strings.HasSuffix(r.URL.Path, "/interface/id/br0") {
					t.Errorf("path = %s", r.URL.Path)
				}
				mtu := 1500
				return client.NetworkInterface{
					ID:            "br0",
					Name:          "br0",
					Type:          "BRIDGE",
					Description:   "core bridge",
					MTU:           &mtu,
					BridgeMembers: []string{"enp1s0"},
				}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		got, err := c.GetInterface(ctx, "br0")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "br0" || got.Type != "BRIDGE" {
			t.Errorf("unexpected: %+v", got)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onGet: func(_ *http.Request) (client.NetworkInterface, int) {
				return client.NetworkInterface{}, http.StatusNotFound
			},
		}
		_, c := newTestServer(t, router)

		_, err := c.GetInterface(ctx, "missing")
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
			_, _ = w.Write([]byte("bogus"))
		}))
		_, err := c.GetInterface(ctx, "br0")
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("List success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/interface") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.NetworkInterface{
				{ID: "br0", Name: "br0", Type: "BRIDGE"},
				{ID: "vlan10", Name: "vlan10", Type: "VLAN"},
			})
		}))
		list, err := c.ListInterfaces(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("Create success body + commit + checkin", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onCreate: func(r *http.Request) (client.NetworkInterface, int) {
				body, _ := io.ReadAll(r.Body)
				var decoded map[string]interface{}
				if err := json.Unmarshal(body, &decoded); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if decoded["type"].(string) != "BRIDGE" {
					t.Errorf("type = %v", decoded["type"])
				}
				if decoded["name"].(string) != "br0" {
					t.Errorf("name = %v", decoded["name"])
				}
				return client.NetworkInterface{
					ID: "br0", Name: "br0", Type: "BRIDGE",
				}, http.StatusOK
			},
			onGet: func(_ *http.Request) (client.NetworkInterface, int) {
				return client.NetworkInterface{
					ID: "br0", Name: "br0", Type: "BRIDGE",
				}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		got, err := c.CreateInterface(ctx, &client.NetworkInterfaceCreateRequest{
			Name:          "br0",
			Type:          "BRIDGE",
			BridgeMembers: []string{"enp1s0"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != "br0" {
			t.Errorf("ID = %q", got.ID)
		}
	})

	t.Run("Create commit failure", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onCreate: func(_ *http.Request) (client.NetworkInterface, int) {
				return client.NetworkInterface{ID: "br0", Type: "BRIDGE"}, http.StatusOK
			},
			failCommit: true,
		}
		_, c := newTestServer(t, router)

		// Disable retries so 500s don't loop.
		withFastRetries(c, 1)
		_, err := c.CreateInterface(ctx, &client.NetworkInterfaceCreateRequest{
			Name: "br0", Type: "BRIDGE",
		})
		if err == nil {
			t.Fatal("expected commit error")
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid type"})
		}))

		_, err := c.CreateInterface(ctx, &client.NetworkInterfaceCreateRequest{
			Name: "br0", Type: "INVALID",
		})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("Create 500 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "err"})
		}))
		_, err := c.CreateInterface(ctx, &client.NetworkInterfaceCreateRequest{
			Name: "br0", Type: "BRIDGE",
		})
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
		_, err := c.CreateInterface(ctx, &client.NetworkInterfaceCreateRequest{
			Name: "br0", Type: "BRIDGE",
		})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update success sends body + commit + checkin", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onUpdate: func(r *http.Request) (client.NetworkInterface, int) {
				if !strings.HasSuffix(r.URL.Path, "/interface/id/br0") {
					t.Errorf("path = %s", r.URL.Path)
				}
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), `"description":"updated"`) {
					t.Errorf("body missing description: %s", body)
				}
				return client.NetworkInterface{
					ID: "br0", Name: "br0", Type: "BRIDGE", Description: "updated",
				}, http.StatusOK
			},
			onGet: func(_ *http.Request) (client.NetworkInterface, int) {
				return client.NetworkInterface{
					ID: "br0", Name: "br0", Type: "BRIDGE", Description: "updated",
				}, http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		desc := "updated"
		got, err := c.UpdateInterface(ctx, "br0", &client.NetworkInterfaceUpdateRequest{
			Description: &desc,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "updated" {
			t.Errorf("Description = %q", got.Description)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		withFastRetries(c, 1)
		desc := "x"
		_, err := c.UpdateInterface(ctx, "br0", &client.NetworkInterfaceUpdateRequest{
			Description: &desc,
		})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected APIError, got %T", err)
		}
	})

	t.Run("Update invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bogus"))
		}))
		withFastRetries(c, 1)
		desc := "x"
		_, err := c.UpdateInterface(ctx, "br0", &client.NetworkInterfaceUpdateRequest{
			Description: &desc,
		})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success + commit + checkin", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onDelete: func(r *http.Request) int {
				if !strings.HasSuffix(r.URL.Path, "/interface/id/br0") {
					t.Errorf("path = %s", r.URL.Path)
				}
				return http.StatusOK
			},
		}
		_, c := newTestServer(t, router)

		if err := c.DeleteInterface(ctx, "br0"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 returns IsNotFound error", func(t *testing.T) {
		router := &networkInterfaceRouter{
			t: t,
			onDelete: func(_ *http.Request) int {
				return http.StatusNotFound
			},
		}
		_, c := newTestServer(t, router)
		withFastRetries(c, 1)

		err := c.DeleteInterface(ctx, "br0")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
