package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestSnapshotTask_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.SnapshotTask{
				ID:           4,
				Dataset:      "tank/data",
				Recursive:    true,
				Lifetime:     2,
				LifetimeUnit: "WEEK",
				NamingSchema: "auto-%Y%m%d",
				Schedule: client.Schedule{
					Minute: "0",
					Hour:   "*",
					Dom:    "*",
					Month:  "*",
					Dow:    "*",
				},
				Enabled:    true,
				AllowEmpty: false,
			})
		}))

		got, err := c.GetSnapshotTask(ctx, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Dataset != "tank/data" {
			t.Errorf("Dataset = %q", got.Dataset)
		}
		if got.LifetimeUnit != "WEEK" {
			t.Errorf("LifetimeUnit = %q", got.LifetimeUnit)
		}
		if !got.Recursive {
			t.Errorf("Recursive = false, want true")
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))

		_, err := c.GetSnapshotTask(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})

	t.Run("Create", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			var req client.SnapshotTaskCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Dataset != "tank/new" {
				t.Errorf("Dataset = %q", req.Dataset)
			}
			if req.Schedule.Minute != "0" {
				t.Errorf("Schedule.Minute = %q", req.Schedule.Minute)
			}
			writeJSON(w, http.StatusOK, client.SnapshotTask{
				ID:           15,
				Dataset:      req.Dataset,
				Lifetime:     req.Lifetime,
				LifetimeUnit: req.LifetimeUnit,
				NamingSchema: req.NamingSchema,
				Schedule:     req.Schedule,
				Enabled:      true,
			})
		}))

		resp, err := c.CreateSnapshotTask(ctx, &client.SnapshotTaskCreateRequest{
			Dataset:      "tank/new",
			Recursive:    false,
			Lifetime:     30,
			LifetimeUnit: "DAY",
			NamingSchema: "auto-%Y%m%d-%H%M",
			Schedule: client.Schedule{
				Minute: "0",
				Hour:   "0",
				Dom:    "*",
				Month:  "*",
				Dow:    "*",
			},
			Enabled: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 15 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.SnapshotTask{
				ID:      15,
				Dataset: "tank/new",
				Enabled: false,
			})
		}))

		enabled := false
		resp, err := c.UpdateSnapshotTask(ctx, 15, &client.SnapshotTaskUpdateRequest{
			Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Enabled {
			t.Errorf("Enabled = true, want false")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteSnapshotTask(ctx, 15); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
