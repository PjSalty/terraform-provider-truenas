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

func TestCronJob_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("ListCronJobs success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cronjob") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.CronJob{
				{ID: 1, User: "root", Command: "/usr/bin/echo hi", Enabled: true},
				{ID: 2, User: "admin", Command: "/bin/true"},
			})
		}))
		list, err := c.ListCronJobs(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d, want 2", len(list))
		}
	})

	t.Run("ListCronJobs server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.ListCronJobs(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ListCronJobs invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-an-array"))
		}))
		_, err := c.ListCronJobs(ctx)
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("GetCronJob success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cronjob/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.CronJob{
				ID: 7, User: "root", Command: "/bin/date",
				Enabled: true, Stdout: true, Stderr: false,
				Schedule: client.Schedule{Minute: "0", Hour: "*", Dom: "*", Month: "*", Dow: "*"},
			})
		}))
		got, err := c.GetCronJob(ctx, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 7 {
			t.Errorf("ID = %d", got.ID)
		}
		if got.Schedule.Minute != "0" {
			t.Errorf("Schedule.Minute = %q", got.Schedule.Minute)
		}
	})

	t.Run("GetCronJob 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetCronJob(ctx, 404)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true (err=%v)", err)
		}
	})

	t.Run("GetCronJob invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{bad"))
		}))
		_, err := c.GetCronJob(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("CreateCronJob sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.CronJobCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.User != "root" {
				t.Errorf("User = %q", req.User)
			}
			if req.Command != "/bin/backup" {
				t.Errorf("Command = %q", req.Command)
			}
			if req.Schedule.Hour != "3" {
				t.Errorf("Schedule.Hour = %q", req.Schedule.Hour)
			}
			writeJSON(w, http.StatusOK, client.CronJob{
				ID: 5, User: req.User, Command: req.Command,
				Enabled: req.Enabled, Schedule: req.Schedule,
			})
		}))
		resp, err := c.CreateCronJob(ctx, &client.CronJobCreateRequest{
			User: "root", Command: "/bin/backup", Enabled: true,
			Schedule: client.Schedule{Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 5 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("CreateCronJob 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad schedule"})
		}))
		_, err := c.CreateCronJob(ctx, &client.CronJobCreateRequest{User: "root", Command: "x"})
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

	t.Run("CreateCronJob invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.CreateCronJob(ctx, &client.CronJobCreateRequest{User: "root", Command: "x"})
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("UpdateCronJob sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cronjob/id/9") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.CronJobUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Command != "/bin/new" {
				t.Errorf("Command = %q", req.Command)
			}
			if req.Enabled == nil || *req.Enabled != false {
				t.Errorf("Enabled = %v, want false", req.Enabled)
			}
			writeJSON(w, http.StatusOK, client.CronJob{ID: 9, Command: "/bin/new"})
		}))
		enabled := false
		resp, err := c.UpdateCronJob(ctx, 9, &client.CronJobUpdateRequest{
			Command: "/bin/new", Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 9 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("UpdateCronJob server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateCronJob(ctx, 1, &client.CronJobUpdateRequest{Command: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("UpdateCronJob invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := c.UpdateCronJob(ctx, 1, &client.CronJobUpdateRequest{Command: "x"})
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("DeleteCronJob success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cronjob/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteCronJob(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteCronJob 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteCronJob(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("DeleteCronJob server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		err := c.DeleteCronJob(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
