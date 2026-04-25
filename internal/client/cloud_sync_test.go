package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestCloudSync_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get preserves attributes", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{
				"id": 1,
				"description": "backup to s3",
				"path": "/mnt/tank/backups",
				"credentials": 2,
				"direction": "PUSH",
				"transfer_mode": "COPY",
				"schedule": {"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*"},
				"enabled": true,
				"attributes": {
					"bucket": "my-backup",
					"folder": "daily",
					"storage_class": "STANDARD"
				}
			}`))
		}))

		got, err := c.GetCloudSync(ctx, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Description != "backup to s3" {
			t.Errorf("Description = %q", got.Description)
		}
		if got.Schedule.Hour != "2" {
			t.Errorf("Schedule.Hour = %q", got.Schedule.Hour)
		}
		if got.Attributes["bucket"] != "my-backup" {
			t.Errorf("attributes.bucket = %v", got.Attributes["bucket"])
		}
		if got.Attributes["storage_class"] != "STANDARD" {
			t.Errorf("attributes.storage_class = %v", got.Attributes["storage_class"])
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "nope"})
		}))
		_, err := c.GetCloudSync(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})

	t.Run("Create sends attributes verbatim", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			attrs, ok := req["attributes"].(map[string]interface{})
			if !ok {
				t.Fatalf("attributes not present or wrong type")
			}
			if attrs["bucket"] != "mybucket" {
				t.Errorf("bucket = %v", attrs["bucket"])
			}
			if attrs["chunk_size"].(float64) != 96 {
				t.Errorf("chunk_size = %v", attrs["chunk_size"])
			}
			_, _ = w.Write([]byte(`{
				"id": 10,
				"path": "/mnt/tank/data",
				"credentials": 2,
				"direction": "PUSH",
				"transfer_mode": "COPY",
				"schedule": {"minute": "0", "hour": "*", "dom": "*", "month": "*", "dow": "*"},
				"enabled": true,
				"attributes": {"bucket": "mybucket", "chunk_size": 96}
			}`))
		}))

		resp, err := c.CreateCloudSync(ctx, &client.CloudSyncCreateRequest{
			Path:         "/mnt/tank/data",
			Credentials:  2,
			Direction:    "PUSH",
			TransferMode: "COPY",
			Schedule: client.Schedule{
				Minute: "0",
				Hour:   "*",
				Dom:    "*",
				Month:  "*",
				Dow:    "*",
			},
			Enabled: true,
			Attributes: map[string]interface{}{
				"bucket":     "mybucket",
				"chunk_size": 96,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 10 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("Update", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			_, _ = w.Write([]byte(`{
				"id": 10,
				"path": "/mnt/tank/data",
				"credentials": 2,
				"direction": "PUSH",
				"transfer_mode": "COPY",
				"schedule": {"minute": "0", "hour": "*", "dom": "*", "month": "*", "dow": "*"},
				"enabled": false,
				"attributes": {}
			}`))
		}))

		enabled := false
		resp, err := c.UpdateCloudSync(ctx, 10, &client.CloudSyncUpdateRequest{
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

		if err := c.DeleteCloudSync(ctx, 10); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 -> IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))

		err := c.DeleteCloudSync(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})
}
