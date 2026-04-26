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

func TestUser_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("ListUsers success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/user") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.User{
				{ID: 1, UID: 0, Username: "root", FullName: "root"},
				{ID: 2, UID: 1000, Username: "alice", FullName: "Alice"},
			})
		}))
		list, err := c.ListUsers(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("ListUsers invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-json"))
		}))
		_, err := c.ListUsers(ctx)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetUser success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/user/id/42") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.User{
				ID: 42, UID: 1001, Username: "bob", FullName: "Bob",
			})
		}))
		got, err := c.GetUser(ctx, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Username != "bob" {
			t.Errorf("Username = %q", got.Username)
		}
	})

	t.Run("GetUser 404 wrapped", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetUser(ctx, 999)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetUser invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("!!!"))
		}))
		_, err := c.GetUser(ctx, 1)
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("GetUserByUsername success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.User{
				{ID: 1, Username: "root"},
				{ID: 2, Username: "alice"},
			})
		}))
		u, err := c.GetUserByUsername(ctx, "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.ID != 2 {
			t.Errorf("ID = %d", u.ID)
		}
	})

	t.Run("GetUserByUsername not found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.User{{ID: 1, Username: "root"}})
		}))
		_, err := c.GetUserByUsername(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})

	t.Run("GetUserByUsername list error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "err"})
		}))
		_, err := c.GetUserByUsername(ctx, "alice")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("CreateUser sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.UserCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.Username != "newuser" {
				t.Errorf("Username = %q", req.Username)
			}
			if req.Password != "secret" {
				t.Errorf("Password = %q", req.Password)
			}
			if !req.GroupCreate {
				t.Errorf("GroupCreate = false, want true")
			}
			writeJSON(w, http.StatusOK, client.User{
				ID: 100, UID: 1100, Username: req.Username, FullName: req.FullName,
			})
		}))
		resp, err := c.CreateUser(ctx, &client.UserCreateRequest{
			Username: "newuser", FullName: "New User",
			Password: "secret", GroupCreate: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 100 {
			t.Errorf("ID = %d", resp.ID)
		}
	})

	t.Run("CreateUser 422 server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "user exists"})
		}))
		_, err := c.CreateUser(ctx, &client.UserCreateRequest{Username: "dup"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err not *APIError: %T", err)
		}
	})

	t.Run("CreateUser invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.CreateUser(ctx, &client.UserCreateRequest{Username: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("UpdateUser sends proper body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/user/id/42") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.UserUpdateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding: %v", err)
			}
			if req.FullName != "Updated" {
				t.Errorf("FullName = %q", req.FullName)
			}
			if req.Locked == nil || *req.Locked != true {
				t.Errorf("Locked = %v, want true", req.Locked)
			}
			writeJSON(w, http.StatusOK, client.User{ID: 42, FullName: "Updated"})
		}))
		locked := true
		resp, err := c.UpdateUser(ctx, 42, &client.UserUpdateRequest{
			FullName: "Updated", Locked: &locked,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.FullName != "Updated" {
			t.Errorf("FullName = %q", resp.FullName)
		}
	})

	t.Run("UpdateUser server error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		_, err := c.UpdateUser(ctx, 1, &client.UserUpdateRequest{FullName: "x"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("UpdateUser invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("oops"))
		}))
		_, err := c.UpdateUser(ctx, 1, &client.UserUpdateRequest{FullName: "x"})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("DeleteUser success with body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/user/id/42") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req map[string]bool
			if err := json.Unmarshal(body, &req); err != nil {
				t.Errorf("decoding body: %v", err)
			}
			if !req["delete_group"] {
				t.Errorf("delete_group = false, want true")
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteUser(ctx, 42); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeleteUser 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteUser(ctx, 1)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false, want true")
		}
	})
}
