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

func TestISCSIAuth_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s, want GET", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/auth/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.ISCSIAuth{
				ID:            7,
				Tag:           2,
				User:          "initiator",
				Secret:        "s3cret12345",
				Peeruser:      "peer",
				Peersecret:    "p33rSecret",
				DiscoveryAuth: "CHAP",
			})
		}))

		got, err := c.GetISCSIAuth(ctx, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 7 || got.Tag != 2 || got.User != "initiator" {
			t.Errorf("unexpected: %+v", got)
		}
		if got.DiscoveryAuth != "CHAP" {
			t.Errorf("DiscoveryAuth = %q", got.DiscoveryAuth)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))

		_, err := c.GetISCSIAuth(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))

		_, err := c.GetISCSIAuth(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create success sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/auth") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var decoded map[string]interface{}
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatalf("decoding body: %v", err)
			}
			if decoded["tag"].(float64) != 3 {
				t.Errorf("tag = %v", decoded["tag"])
			}
			if decoded["user"].(string) != "u1" {
				t.Errorf("user = %v", decoded["user"])
			}
			if decoded["secret"].(string) != "sekret12" {
				t.Errorf("secret = %v", decoded["secret"])
			}

			writeJSON(w, http.StatusOK, client.ISCSIAuth{
				ID:     11,
				Tag:    3,
				User:   "u1",
				Secret: "sekret12",
			})
		}))

		got, err := c.CreateISCSIAuth(ctx, &client.ISCSIAuthCreateRequest{
			Tag:    3,
			User:   "u1",
			Secret: "sekret12",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 11 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422 returns APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "secret too short"})
		}))

		_, err := c.CreateISCSIAuth(ctx, &client.ISCSIAuthCreateRequest{Tag: 1, User: "x", Secret: "short"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("Create 500 returns APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))

		_, err := c.CreateISCSIAuth(ctx, &client.ISCSIAuthCreateRequest{Tag: 1, User: "x", Secret: "pw12345678"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
	})

	t.Run("Create invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{broken"))
		}))

		_, err := c.CreateISCSIAuth(ctx, &client.ISCSIAuthCreateRequest{Tag: 1, User: "x", Secret: "pw12345678"})
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Update success sends body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s, want PUT", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/auth/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"user":"newuser"`) {
				t.Errorf("body missing user: %s", body)
			}
			writeJSON(w, http.StatusOK, client.ISCSIAuth{ID: 5, Tag: 1, User: "newuser"})
		}))

		newUser := "newuser"
		got, err := c.UpdateISCSIAuth(ctx, 5, &client.ISCSIAuthUpdateRequest{User: &newUser})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.User != "newuser" {
			t.Errorf("User = %q", got.User)
		}
	})

	t.Run("Update 422 returns APIError", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid"})
		}))

		newUser := "x"
		_, err := c.UpdateISCSIAuth(ctx, 5, &client.ISCSIAuthUpdateRequest{User: &newUser})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != http.StatusUnprocessableEntity {
			t.Errorf("StatusCode = %d", apiErr.StatusCode)
		}
	})

	t.Run("Update invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bogus"))
		}))

		newUser := "x"
		_, err := c.UpdateISCSIAuth(ctx, 5, &client.ISCSIAuthUpdateRequest{User: &newUser})
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s, want DELETE", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/iscsi/auth/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteISCSIAuth(ctx, 7); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404 is IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))

		err := c.DeleteISCSIAuth(ctx, 7)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("Delete 422 does not exist is IsNotFound", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "record does not exist"})
		}))

		err := c.DeleteISCSIAuth(ctx, 7)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})
}
