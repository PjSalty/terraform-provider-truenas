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

func TestReplication_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/replication/id/2") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Replication{
				ID:              2,
				Name:            "daily-backup",
				Direction:       "PUSH",
				Transport:       "SSH",
				SourceDatasets:  []string{"tank/data"},
				TargetDataset:   "backup/tank/data",
				Recursive:       true,
				AutoBool:        true,
				Enabled:         true,
				RetentionPolicy: "SOURCE",
				NamingSchema:    []string{"auto-%Y-%m-%d"},
			})
		}))

		got, err := c.GetReplication(ctx, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "daily-backup" {
			t.Errorf("Name = %q", got.Name)
		}
		if len(got.SourceDatasets) != 1 || got.SourceDatasets[0] != "tank/data" {
			t.Errorf("SourceDatasets = %+v", got.SourceDatasets)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetReplication(ctx, 99)
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
			_, _ = w.Write([]byte("xx"))
		}))
		_, err := c.GetReplication(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Create sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/replication") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded["name"].(string) != "rep1" {
				t.Errorf("name = %v", decoded["name"])
			}
			if decoded["direction"].(string) != "PUSH" {
				t.Errorf("direction = %v", decoded["direction"])
			}
			src, _ := decoded["source_datasets"].([]interface{})
			if len(src) != 1 {
				t.Errorf("source_datasets len = %d", len(src))
			}
			writeJSON(w, http.StatusOK, client.Replication{
				ID:   11,
				Name: "rep1",
			})
		}))

		got, err := c.CreateReplication(ctx, &client.ReplicationCreateRequest{
			Name:            "rep1",
			Direction:       "PUSH",
			Transport:       "SSH",
			SourceDatasets:  []string{"tank/a"},
			TargetDataset:   "backup/tank/a",
			Recursive:       true,
			Enabled:         true,
			RetentionPolicy: "SOURCE",
			NamingSchema:    []string{"auto-%Y-%m-%d"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 11 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid"})
		}))
		_, err := c.CreateReplication(ctx, &client.ReplicationCreateRequest{Name: "x"})
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
		_, err := c.CreateReplication(ctx, &client.ReplicationCreateRequest{Name: "x"})
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
		_, err := c.CreateReplication(ctx, &client.ReplicationCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Update sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/replication/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"name":"renamed"`) {
				t.Errorf("body missing name: %s", body)
			}
			writeJSON(w, http.StatusOK, client.Replication{ID: 7, Name: "renamed"})
		}))

		enabled := false
		got, err := c.UpdateReplication(ctx, 7, &client.ReplicationUpdateRequest{
			Name:    "renamed",
			Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update 422 APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "nope"})
		}))
		_, err := c.UpdateReplication(ctx, 7, &client.ReplicationUpdateRequest{Name: "x"})
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
		_, err := c.UpdateReplication(ctx, 7, &client.ReplicationUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/replication/id/2") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteReplication(ctx, 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteReplication(ctx, 2)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
