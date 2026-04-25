package client_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

func TestGroup_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("List", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Group{
				{ID: 1, GID: 1000, Name: "wheel", Builtin: true},
				{ID: 2, GID: 3000, Name: "users", Builtin: false},
			})
		}))

		list, err := c.ListGroups(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("len = %d", len(list))
		}
	})

	t.Run("Get", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/group/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Group{
				ID:   7,
				GID:  3007,
				Name: "devs",
			})
		}))

		got, err := c.GetGroup(ctx, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Name != "devs" || got.GID != 3007 {
			t.Errorf("got = %+v", got)
		}
	})

	t.Run("GetByName found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Group{
				{ID: 1, Name: "admin"},
				{ID: 2, Name: "users"},
				{ID: 3, Name: "guests"},
			})
		}))

		got, err := c.GetGroupByName(ctx, "users")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != 2 {
			t.Errorf("ID = %d, want 2", got.ID)
		}
	})

	t.Run("GetByName not found", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []client.Group{
				{ID: 1, Name: "admin"},
			})
		}))

		_, err := c.GetGroupByName(ctx, "ghost")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false (err=%v)", err)
		}
	})

	t.Run("Create returns int then re-fetches", func(t *testing.T) {
		// Create endpoint returns just the integer ID; the client then
		// calls GetGroup(newID) to load the full record.
		var postCount int32
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				atomic.AddInt32(&postCount, 1)
				body, _ := io.ReadAll(r.Body)
				var req client.GroupCreateRequest
				if err := json.Unmarshal(body, &req); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if req.Name != "devs" {
					t.Errorf("Name = %q", req.Name)
				}
				_, _ = w.Write([]byte("42"))
			case http.MethodGet:
				if !strings.Contains(r.URL.Path, "/group/id/42") {
					t.Errorf("GET path = %s", r.URL.Path)
				}
				writeJSON(w, http.StatusOK, client.Group{
					ID:   42,
					GID:  3042,
					Name: "devs",
				})
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		}))

		resp, err := c.CreateGroup(ctx, &client.GroupCreateRequest{Name: "devs"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ID != 42 {
			t.Errorf("ID = %d", resp.ID)
		}
		if atomic.LoadInt32(&postCount) != 1 {
			t.Errorf("postCount = %d, want 1", postCount)
		}
	})

	t.Run("Update returns int then re-fetches", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				_, _ = w.Write([]byte("42"))
			case http.MethodGet:
				writeJSON(w, http.StatusOK, client.Group{
					ID:   42,
					Name: "renamed",
				})
			default:
				t.Errorf("unexpected method %s", r.Method)
			}
		}))

		resp, err := c.UpdateGroup(ctx, 42, &client.GroupUpdateRequest{Name: "renamed"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Name != "renamed" {
			t.Errorf("Name = %q", resp.Name)
		}
	})

	t.Run("Delete sends delete_users:false body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var got map[string]bool
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("decode body: %v (body=%q)", err, body)
			}
			if v, ok := got["delete_users"]; !ok || v {
				t.Errorf("body = %+v, want delete_users:false", got)
			}
			w.WriteHeader(http.StatusOK)
		}))

		if err := c.DeleteGroup(ctx, 42); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "gone"})
		}))

		err := c.DeleteGroup(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound false")
		}
	})
}
