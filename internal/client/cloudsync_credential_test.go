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

func TestCloudSyncCredential_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloudsync/credentials/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.CloudSyncCredential{
				ID:   3,
				Name: "my-s3",
				Provider: map[string]interface{}{
					"type":              "S3",
					"access_key_id":     "AKIA...",
					"secret_access_key": "sekret",
				},
			})
		}))

		got, err := c.GetCloudSyncCredential(ctx, 3)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "my-s3" {
			t.Errorf("Name = %q", got.Name)
		}
		if got.Provider["type"].(string) != "S3" {
			t.Errorf("Provider type = %v", got.Provider["type"])
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		_, err := c.GetCloudSyncCredential(ctx, 99)
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
		_, err := c.GetCloudSyncCredential(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("List success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/cloudsync/credentials") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.CloudSyncCredential{
				{ID: 1, Name: "s3-prod", Provider: map[string]interface{}{"type": "S3"}},
				{ID: 2, Name: "b2-backup", Provider: map[string]interface{}{"type": "B2"}},
			})
		}))

		list, err := c.ListCloudSyncCredentials(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("GetByName found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.CloudSyncCredential{
				{ID: 1, Name: "s3-prod"},
				{ID: 2, Name: "b2-backup"},
			})
		}))

		got, err := c.GetCloudSyncCredentialByName(ctx, "b2-backup")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 2 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("GetByName not found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.CloudSyncCredential{
				{ID: 1, Name: "other"},
			})
		}))

		_, err := c.GetCloudSyncCredentialByName(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Create sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloudsync/credentials") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded["name"].(string) != "new-s3" {
				t.Errorf("name = %v", decoded["name"])
			}
			prov, _ := decoded["provider"].(map[string]interface{})
			if prov["type"].(string) != "S3" {
				t.Errorf("provider.type = %v", prov["type"])
			}
			writeJSON(w, http.StatusOK, client.CloudSyncCredential{
				ID:       11,
				Name:     "new-s3",
				Provider: map[string]interface{}{"type": "S3"},
			})
		}))

		got, err := c.CreateCloudSyncCredential(ctx, &client.CloudSyncCredentialCreateRequest{
			Name: "new-s3",
			Provider: map[string]interface{}{
				"type":              "S3",
				"access_key_id":     "AKIA...",
				"secret_access_key": "sekret",
			},
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
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad provider"})
		}))
		_, err := c.CreateCloudSyncCredential(ctx, &client.CloudSyncCredentialCreateRequest{
			Name:     "x",
			Provider: map[string]interface{}{"type": "S3"},
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
		_, err := c.CreateCloudSyncCredential(ctx, &client.CloudSyncCredentialCreateRequest{
			Name:     "x",
			Provider: map[string]interface{}{"type": "S3"},
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
		_, err := c.CreateCloudSyncCredential(ctx, &client.CloudSyncCredentialCreateRequest{
			Name:     "x",
			Provider: map[string]interface{}{"type": "S3"},
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
			if !strings.HasSuffix(r.URL.Path, "/cloudsync/credentials/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"name":"renamed"`) {
				t.Errorf("body missing name: %s", body)
			}
			writeJSON(w, http.StatusOK, client.CloudSyncCredential{ID: 7, Name: "renamed"})
		}))

		got, err := c.UpdateCloudSyncCredential(ctx, 7, &client.CloudSyncCredentialUpdateRequest{
			Name: "renamed",
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
		_, err := c.UpdateCloudSyncCredential(ctx, 7, &client.CloudSyncCredentialUpdateRequest{Name: "x"})
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
		_, err := c.UpdateCloudSyncCredential(ctx, 7, &client.CloudSyncCredentialUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/cloudsync/credentials/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteCloudSyncCredential(ctx, 3); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))
		err := c.DeleteCloudSyncCredential(ctx, 3)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})
}
