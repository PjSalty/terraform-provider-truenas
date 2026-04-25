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

func TestRsyncTask_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/rsynctask/id/4") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.RsyncTask{
				ID:         4,
				Path:       "/mnt/tank/data",
				Remotehost: "backup.example.com",
				Remoteport: 22,
				Mode:       "SSH",
				Direction:  "PUSH",
				User:       "rsync",
				Enabled:    true,
				Schedule: client.Schedule{
					Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*",
				},
			})
		}))

		got, err := c.GetRsyncTask(ctx, 4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Path != "/mnt/tank/data" {
			t.Errorf("Path = %q", got.Path)
		}
		if got.Remotehost != "backup.example.com" {
			t.Errorf("Remotehost = %q", got.Remotehost)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetRsyncTask(ctx, 99)
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
		_, err := c.GetRsyncTask(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Create sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/rsynctask") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded["path"].(string) != "/mnt/tank/data" {
				t.Errorf("path = %v", decoded["path"])
			}
			if decoded["user"].(string) != "rsync" {
				t.Errorf("user = %v", decoded["user"])
			}
			if decoded["direction"].(string) != "PUSH" {
				t.Errorf("direction = %v", decoded["direction"])
			}
			writeJSON(w, http.StatusOK, client.RsyncTask{
				ID:        7,
				Path:      "/mnt/tank/data",
				User:      "rsync",
				Direction: "PUSH",
				Enabled:   true,
			})
		}))

		got, err := c.CreateRsyncTask(ctx, &client.RsyncTaskCreateRequest{
			Path:       "/mnt/tank/data",
			Remotehost: "backup.example.com",
			Mode:       "SSH",
			Direction:  "PUSH",
			User:       "rsync",
			Enabled:    true,
			Schedule: client.Schedule{
				Minute: "0", Hour: "3", Dom: "*", Month: "*", Dow: "*",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 7 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad path"})
		}))
		_, err := c.CreateRsyncTask(ctx, &client.RsyncTaskCreateRequest{Path: "/bad", User: "root"})
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
		_, err := c.CreateRsyncTask(ctx, &client.RsyncTaskCreateRequest{Path: "/x", User: "root"})
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
		_, err := c.CreateRsyncTask(ctx, &client.RsyncTaskCreateRequest{Path: "/x", User: "root"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/rsynctask/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"desc":"updated desc"`) {
				t.Errorf("body missing desc: %s", body)
			}
			writeJSON(w, http.StatusOK, client.RsyncTask{ID: 5, Desc: "updated desc"})
		}))

		enabled := true
		got, err := c.UpdateRsyncTask(ctx, 5, &client.RsyncTaskUpdateRequest{
			Desc:    "updated desc",
			Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Desc != "updated desc" {
			t.Errorf("Desc = %q", got.Desc)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "nope"})
		}))
		_, err := c.UpdateRsyncTask(ctx, 5, &client.RsyncTaskUpdateRequest{Desc: "x"})
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
		_, err := c.UpdateRsyncTask(ctx, 5, &client.RsyncTaskUpdateRequest{Desc: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/rsynctask/id/4") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteRsyncTask(ctx, 4); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteRsyncTask(ctx, 4)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
