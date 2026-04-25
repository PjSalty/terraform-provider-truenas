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

func TestPrivilege_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("List success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/privilege") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, []client.Privilege{
				{ID: 1, Name: "ops", LocalGroups: []client.PrivilegeGroup{{ID: 1, GID: 100, Name: "ops"}}, Roles: []string{"READONLY_ADMIN"}},
				{ID: 2, Name: "dev"},
			})
		}))

		got, err := c.ListPrivileges(ctx)
		if err != nil {
			t.Fatalf("ListPrivileges: %v", err)
		}
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
		if got[0].Name != "ops" {
			t.Errorf("[0].Name = %q", got[0].Name)
		}
	})

	t.Run("List invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.ListPrivileges(ctx)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("List 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.ListPrivileges(ctx)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/privilege/id/5") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.Privilege{
				ID:    5,
				Name:  "priv5",
				Roles: []string{"READONLY_ADMIN", "SHARING_ADMIN"},
				LocalGroups: []client.PrivilegeGroup{
					{ID: 10, GID: 1000, Name: "group1"},
					{ID: 11, GID: 1001, Name: "group2"},
				},
			})
		}))

		got, err := c.GetPrivilege(ctx, 5)
		if err != nil {
			t.Fatalf("GetPrivilege: %v", err)
		}
		if got.ID != 5 || got.Name != "priv5" {
			t.Errorf("got = %+v", got)
		}
		gids := got.LocalGroupGIDs()
		if len(gids) != 2 || gids[0] != 1000 || gids[1] != 1001 {
			t.Errorf("LocalGroupGIDs = %v", gids)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetPrivilege(ctx, 999)
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
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetPrivilege(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.PrivilegeCreateRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Name != "ops" {
				t.Errorf("Name = %q", req.Name)
			}
			if len(req.LocalGroups) != 2 || req.LocalGroups[0] != 100 {
				t.Errorf("LocalGroups = %v", req.LocalGroups)
			}
			if !req.WebShell {
				t.Errorf("WebShell = false")
			}
			writeJSON(w, http.StatusOK, client.Privilege{ID: 7, Name: "ops", WebShell: true})
		}))

		got, err := c.CreatePrivilege(ctx, &client.PrivilegeCreateRequest{
			Name:        "ops",
			LocalGroups: []int{100, 101},
			DSGroups:    []interface{}{"S-1-5-21-xxx"},
			Roles:       []string{"READONLY_ADMIN"},
			WebShell:    true,
		})
		if err != nil {
			t.Fatalf("CreatePrivilege: %v", err)
		}
		if got.ID != 7 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid role"})
		}))
		_, err := c.CreatePrivilege(ctx, &client.PrivilegeCreateRequest{Name: "bad"})
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
		_, err := c.CreatePrivilege(ctx, &client.PrivilegeCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/privilege/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["name"] != "renamed" {
				t.Errorf("name = %v", raw["name"])
			}
			if _, has := raw["web_shell"]; has {
				t.Errorf("web_shell should be omitted when nil")
			}
			writeJSON(w, http.StatusOK, client.Privilege{ID: 7, Name: "renamed"})
		}))

		name := "renamed"
		got, err := c.UpdatePrivilege(ctx, 7, &client.PrivilegeUpdateRequest{Name: &name})
		if err != nil {
			t.Fatalf("UpdatePrivilege: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		name := "x"
		_, err := c.UpdatePrivilege(ctx, 7, &client.PrivilegeUpdateRequest{Name: &name})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update marshals web_shell ptr", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["web_shell"] != false {
				t.Errorf("web_shell = %v", raw["web_shell"])
			}
			writeJSON(w, http.StatusOK, client.Privilege{ID: 7})
		}))
		ws := false
		_, _ = c.UpdatePrivilege(ctx, 7, &client.PrivilegeUpdateRequest{WebShell: &ws})
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/privilege/id/7") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeletePrivilege(ctx, 7); err != nil {
			t.Fatalf("DeletePrivilege: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeletePrivilege(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("LocalGroupGIDs empty", func(t *testing.T) {
		p := &client.Privilege{}
		if gids := p.LocalGroupGIDs(); len(gids) != 0 {
			t.Errorf("want empty, got %v", gids)
		}
	})

	t.Run("DSGroupStrings mixed", func(t *testing.T) {
		p := &client.Privilege{DSGroups: []interface{}{"sid-1", float64(42), int(99)}}
		out := p.DSGroupStrings()
		if len(out) != 3 || out[0] != "sid-1" || out[1] != "42" || out[2] != "99" {
			t.Errorf("out = %v", out)
		}
	})
}
