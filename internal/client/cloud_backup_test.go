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

func TestCloudBackup_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloud_backup/id/1") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": 1,
				"description": "daily backup",
				"path": "/mnt/tank/data",
				"credentials": 5,
				"attributes": {"bucket": "my-bucket", "folder": "/"},
				"schedule": {"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*"},
				"pre_script": "",
				"post_script": "",
				"snapshot": true,
				"include": [],
				"exclude": [],
				"args": "",
				"enabled": true,
				"password": "restic-pass",
				"keep_last": 7,
				"transfer_setting": "DEFAULT"
			}`))
		}))

		got, err := c.GetCloudBackup(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "daily backup" {
			t.Errorf("Description = %q", got.Description)
		}
		if got.KeepLast != 7 {
			t.Errorf("KeepLast = %d", got.KeepLast)
		}
		if !got.Snapshot {
			t.Errorf("Snapshot = false")
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetCloudBackup(ctx, 99)
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
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.GetCloudBackup(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Create sends body with attributes RawMessage", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloud_backup") {
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
			if decoded["credentials"].(float64) != 5 {
				t.Errorf("credentials = %v", decoded["credentials"])
			}
			attrs, _ := decoded["attributes"].(map[string]interface{})
			if attrs["bucket"].(string) != "my-bucket" {
				t.Errorf("attributes.bucket = %v", attrs["bucket"])
			}
			if decoded["password"].(string) != "rpass" {
				t.Errorf("password = %v", decoded["password"])
			}
			if decoded["keep_last"].(float64) != 7 {
				t.Errorf("keep_last = %v", decoded["keep_last"])
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 10, "description": "", "path": "/mnt/tank/data", "enabled": true, "password": "rpass", "keep_last": 7, "schedule": {"minute":"0","hour":"2","dom":"*","month":"*","dow":"*"}, "pre_script":"", "post_script":"", "snapshot": false, "include": [], "exclude": [], "args": "", "transfer_setting": ""}`))
		}))

		snap := true
		enabled := true
		got, err := c.CreateCloudBackup(ctx, &client.CloudBackupCreateRequest{
			Path:        "/mnt/tank/data",
			Credentials: 5,
			Attributes:  json.RawMessage(`{"bucket":"my-bucket","folder":"/"}`),
			Schedule: &client.CloudBackupSchedule{
				Minute: "0", Hour: "2", Dom: "*", Month: "*", Dow: "*",
			},
			Snapshot: &snap,
			Enabled:  &enabled,
			Password: "rpass",
			KeepLast: 7,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 10 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad credentials"})
		}))
		_, err := c.CreateCloudBackup(ctx, &client.CloudBackupCreateRequest{
			Path: "/x", Credentials: 1, Attributes: json.RawMessage(`{}`), Password: "p", KeepLast: 1,
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
		_, err := c.CreateCloudBackup(ctx, &client.CloudBackupCreateRequest{
			Path: "/x", Credentials: 1, Attributes: json.RawMessage(`{}`), Password: "p", KeepLast: 1,
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
		_, err := c.CreateCloudBackup(ctx, &client.CloudBackupCreateRequest{
			Path: "/x", Credentials: 1, Attributes: json.RawMessage(`{}`), Password: "p", KeepLast: 1,
		})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloud_backup/id/4") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"description":"new desc"`) {
				t.Errorf("body missing description: %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 4, "description": "new desc", "path": "/x", "schedule": {}, "include": [], "exclude": [], "pre_script":"", "post_script":"", "snapshot": false, "args": "", "enabled": true, "password": "", "keep_last": 0, "transfer_setting": ""}`))
		}))

		desc := "new desc"
		got, err := c.UpdateCloudBackup(ctx, 4, &client.CloudBackupUpdateRequest{
			Description: &desc,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "new desc" {
			t.Errorf("Description = %q", got.Description)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		d := "x"
		_, err := c.UpdateCloudBackup(ctx, 4, &client.CloudBackupUpdateRequest{Description: &d})
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
		d := "x"
		_, err := c.UpdateCloudBackup(ctx, 4, &client.CloudBackupUpdateRequest{Description: &d})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloud_backup/id/1") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteCloudBackup(ctx, 1); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteCloudBackup(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
