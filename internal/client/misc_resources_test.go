// Tests for misc small resources (tunable, static_route) to broaden coverage.
package client_test

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestTunable_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.Tunable{
				ID:      1,
				Type:    "SYSCTL",
				Var:     "vm.swappiness",
				Value:   "10",
				Enabled: true,
			})
		}))

		got, err := c.GetTunable(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Var != "vm.swappiness" {
			t.Errorf("Var = %q", got.Var)
		}
	})

	t.Run("List", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Tunable{
				{ID: 1, Var: "vm.swappiness", Value: "10"},
				{ID: 2, Var: "net.core.somaxconn", Value: "1024"},
			})
		}))

		list, err := c.ListTunables(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("FindTunableByVar", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Tunable{
				{ID: 1, Var: "vm.swappiness"},
				{ID: 2, Var: "net.core.somaxconn"},
			})
		}))

		got, err := c.FindTunableByVar(ctx, "net.core.somaxconn")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 2 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("FindTunableByVar not found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Tunable{})
		}))

		_, err := c.FindTunableByVar(ctx, "missing")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create re-fetches by var", func(t *testing.T) {
		var postCount int32
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				atomic.AddInt32(&postCount, 1)
				// The create endpoint returns some internal ID.
				_, _ = w.Write([]byte(`{"id": 999}`))
			case http.MethodGet:
				// ListTunables is used by FindTunableByVar.
				writeJSON(w, http.StatusOK, []client.Tunable{
					{ID: 5, Var: "vm.swappiness", Value: "10"},
				})
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		}))

		resp, err := c.CreateTunable(ctx, &client.TunableCreateRequest{
			Type:  "SYSCTL",
			Var:   "vm.swappiness",
			Value: "10",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 5 {
			t.Errorf("ID = %d, want 5 (from list, not create response)", resp.ID)
		}
		if atomic.LoadInt32(&postCount) != 1 {
			t.Errorf("postCount = %d, want 1", postCount)
		}
	})

	t.Run("Update re-fetches by stable id", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				_, _ = w.Write([]byte(`{"id": 999}`))
			case http.MethodGet:
				writeJSON(w, http.StatusOK, client.Tunable{
					ID:    5,
					Var:   "vm.swappiness",
					Value: "20",
				})
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		}))

		resp, err := c.UpdateTunable(ctx, 5, &client.TunableUpdateRequest{
			Value: "20",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Value != "20" {
			t.Errorf("Value = %q", resp.Value)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteTunable(ctx, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestStaticRoute_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.StaticRoute{
				ID:          1,
				Destination: "10.20.0.0/24",
				Gateway:     "10.10.20.1",
				Description: "lab",
			})
		}))

		got, err := c.GetStaticRoute(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Destination != "10.20.0.0/24" {
			t.Errorf("Destination = %q", got.Destination)
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.StaticRoute{
				ID:          7,
				Destination: "10.30.0.0/24",
				Gateway:     "10.10.20.1",
			})
		}))

		resp, err := c.CreateStaticRoute(ctx, &client.StaticRouteCreateRequest{
			Destination: "10.30.0.0/24",
			Gateway:     "10.10.20.1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 7 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.StaticRoute{
				ID:          7,
				Destination: "10.30.0.0/24",
				Gateway:     "10.10.20.254",
				Description: "updated",
			})
		}))

		resp, err := c.UpdateStaticRoute(ctx, 7, &client.StaticRouteUpdateRequest{
			Gateway:     "10.10.20.254",
			Description: "updated",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Gateway != "10.10.20.254" {
			t.Errorf("Gateway = %q", resp.Gateway)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteStaticRoute(ctx, 7); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))

		err := c.DeleteStaticRoute(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})
}
