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

func TestKeychainCredential_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/keychaincredential/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.KeychainCredential{
				ID:   3,
				Name: "backup-key",
				Type: "SSH_KEY_PAIR",
				Attributes: map[string]interface{}{
					"private_key": "PRIVATE",
					"public_key":  "PUBLIC",
				},
			})
		}))

		got, err := c.GetKeychainCredential(ctx, 3)
		if err != nil {
			t.Fatalf("GetKeychainCredential: %v", err)
		}
		if got.ID != 3 || got.Name != "backup-key" {
			t.Errorf("got = %+v", got)
		}
		if got.Type != "SSH_KEY_PAIR" {
			t.Errorf("Type = %q", got.Type)
		}
		if got.Attributes["private_key"] != "PRIVATE" {
			t.Errorf("private_key = %v", got.Attributes["private_key"])
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetKeychainCredential(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetKeychainCredential(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetKeychainCredential(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create SSH_KEY_PAIR", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/keychaincredential") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.KeychainCredentialCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "mykey" {
				t.Errorf("Name = %q", req.Name)
			}
			if req.Type != "SSH_KEY_PAIR" {
				t.Errorf("Type = %q", req.Type)
			}
			if req.Attributes["public_key"] != "ssh-rsa AAAA" {
				t.Errorf("public_key = %v", req.Attributes["public_key"])
			}
			writeJSON(w, http.StatusOK, client.KeychainCredential{ID: 5, Name: "mykey", Type: "SSH_KEY_PAIR"})
		}))

		got, err := c.CreateKeychainCredential(ctx, &client.KeychainCredentialCreateRequest{
			Name: "mykey",
			Type: "SSH_KEY_PAIR",
			Attributes: map[string]interface{}{
				"private_key": "PRIV",
				"public_key":  "ssh-rsa AAAA",
			},
		})
		if err != nil {
			t.Fatalf("CreateKeychainCredential: %v", err)
		}
		if got.ID != 5 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create SSH_CREDENTIALS", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.KeychainCredentialCreateRequest
			_ = json.Unmarshal(body, &req)
			if req.Type != "SSH_CREDENTIALS" {
				t.Errorf("Type = %q", req.Type)
			}
			if req.Attributes["host"] != "backup.example.com" {
				t.Errorf("host = %v", req.Attributes["host"])
			}
			writeJSON(w, http.StatusOK, client.KeychainCredential{ID: 6, Name: "remote", Type: "SSH_CREDENTIALS"})
		}))

		got, err := c.CreateKeychainCredential(ctx, &client.KeychainCredentialCreateRequest{
			Name: "remote",
			Type: "SSH_CREDENTIALS",
			Attributes: map[string]interface{}{
				"host":            "backup.example.com",
				"port":            22,
				"username":        "backup",
				"private_key":     5,
				"remote_host_key": "ssh-rsa xxx",
				"connect_timeout": 10,
			},
		})
		if err != nil {
			t.Fatalf("CreateKeychainCredential: %v", err)
		}
		if got.ID != 6 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid key"})
		}))
		_, err := c.CreateKeychainCredential(ctx, &client.KeychainCredentialCreateRequest{Name: "bad", Type: "SSH_KEY_PAIR"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Create 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.CreateKeychainCredential(ctx, &client.KeychainCredentialCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/keychaincredential/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.KeychainCredentialUpdateRequest
			_ = json.Unmarshal(body, &req)
			if req.Name != "renamed" {
				t.Errorf("Name = %q", req.Name)
			}
			writeJSON(w, http.StatusOK, client.KeychainCredential{ID: 5, Name: "renamed"})
		}))

		got, err := c.UpdateKeychainCredential(ctx, 5, &client.KeychainCredentialUpdateRequest{
			Name: "renamed",
		})
		if err != nil {
			t.Fatalf("UpdateKeychainCredential: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateKeychainCredential(ctx, 5, &client.KeychainCredentialUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.UpdateKeychainCredential(ctx, 5, &client.KeychainCredentialUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/keychaincredential/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteKeychainCredential(ctx, 5); err != nil {
			t.Fatalf("DeleteKeychainCredential: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteKeychainCredential(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Delete in-use 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "in use by replication task"})
		}))
		err := c.DeleteKeychainCredential(ctx, 5)
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Create invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.CreateKeychainCredential(ctx, &client.KeychainCredentialCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.UpdateKeychainCredential(ctx, 5, &client.KeychainCredentialUpdateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
