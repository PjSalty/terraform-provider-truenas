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

func TestAPIKey_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("GetAPIKey success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/api_key/id/42") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.APIKey{
				ID: 42, Name: "backup", Username: "admin", Local: true,
			})
		}))

		got, err := c.GetAPIKey(ctx, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 42 {
			t.Errorf("ID = %d", got.ID)
		}
		if got.Name != "backup" {
			t.Errorf("Name = %q", got.Name)
		}
		if !got.Local {
			t.Errorf("Local = false, want true")
		}
	})

	t.Run("GetAPIKey 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetAPIKey(ctx, 99)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true (err=%v)", err)
		}
	})

	t.Run("GetAPIKey invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("<html>not json</html>"))
		}))
		_, err := c.GetAPIKey(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("GetAPIKey server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetAPIKey(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
		if apiErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("CreateAPIKey sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/api_key") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.APIKeyCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
			if req.Name != "ci-key" {
				t.Errorf("Name = %q, want ci-key", req.Name)
			}
			if req.Username != "admin" {
				t.Errorf("Username = %q, want admin", req.Username)
			}
			writeJSON(w, http.StatusOK, client.APIKey{
				ID: 7, Name: req.Name, Username: req.Username,
				Key: "1-raw-secret-string", Local: true,
			})
		}))

		resp, err := c.CreateAPIKey(ctx, &client.APIKeyCreateRequest{
			Name: "ci-key", Username: "admin",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 7 {
			t.Errorf("ID = %d", resp.ID)
		}
		if resp.Key != "1-raw-secret-string" {
			t.Errorf("Key = %q", resp.Key)
		}
	})

	t.Run("CreateAPIKey 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad name"})
		}))
		_, err := c.CreateAPIKey(ctx, &client.APIKeyCreateRequest{Name: ""})
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

	t.Run("CreateAPIKey invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("garbage"))
		}))
		_, err := c.CreateAPIKey(ctx, &client.APIKeyCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("UpdateAPIKey sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/api_key/id/42") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.APIKeyUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
			if req.Name != "renamed" {
				t.Errorf("Name = %q", req.Name)
			}
			if !req.Reset {
				t.Errorf("Reset = false, want true")
			}
			writeJSON(w, http.StatusOK, client.APIKey{
				ID: 42, Name: "renamed", Username: "admin", Local: true,
				Key: "2-new-reset",
			})
		}))

		resp, err := c.UpdateAPIKey(ctx, 42, &client.APIKeyUpdateRequest{
			Name: "renamed", Reset: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "renamed" {
			t.Errorf("Name = %q", resp.Name)
		}
	})

	t.Run("UpdateAPIKey server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "nope"})
		}))
		_, err := c.UpdateAPIKey(ctx, 42, &client.APIKeyUpdateRequest{Name: "x"})
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

	t.Run("UpdateAPIKey invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{not valid"))
		}))
		_, err := c.UpdateAPIKey(ctx, 1, &client.APIKeyUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected parse error, got nil")
		}
	})

	t.Run("DeleteAPIKey success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s, want DELETE", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/api_key/id/42") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteAPIKey(ctx, 42); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteAPIKey 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteAPIKey(ctx, 404)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("DeleteAPIKey server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "oops"})
		}))
		err := c.DeleteAPIKey(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("GetAPIKey zero ID path formatting", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/api_key/id/0") {
				t.Errorf("path = %s, want .../api_key/id/0", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.APIKey{ID: 0, Name: "x"})
		}))
		_, err := c.GetAPIKey(ctx, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("CreateAPIKey empty username omitted", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			// username is omitempty, so it should not be present in the JSON
			if strings.Contains(string(body), `"username"`) {
				t.Errorf("username should be omitted when empty, body = %s", string(body))
			}
			writeJSON(w, http.StatusOK, client.APIKey{ID: 1, Name: "x"})
		}))
		_, err := c.CreateAPIKey(ctx, &client.APIKeyCreateRequest{Name: "x"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
