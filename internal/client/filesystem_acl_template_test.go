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

func TestFilesystemACLTemplate_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/filesystem/acltemplate/id/3") {
				t.Errorf("path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"id":3,"name":"tmpl1","acltype":"POSIX1E","comment":"test","acl":[{"tag":"USER_OBJ"}],"builtin":false}`))
		}))

		got, err := c.GetFilesystemACLTemplate(ctx, 3)
		if err != nil {
			t.Fatalf("GetFilesystemACLTemplate: %v", err)
		}
		if got.ID != 3 || got.Name != "tmpl1" {
			t.Errorf("got = %+v", got)
		}
		if got.ACLType != "POSIX1E" {
			t.Errorf("ACLType = %q", got.ACLType)
		}
		if got.Builtin {
			t.Errorf("Builtin = true")
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		_, err := c.GetFilesystemACLTemplate(ctx, 99)
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
		_, err := c.GetFilesystemACLTemplate(ctx, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.GetFilesystemACLTemplate(ctx, 1)
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Create success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/filesystem/acltemplate") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["name"] != "mytmpl" {
				t.Errorf("name = %v", raw["name"])
			}
			if raw["acltype"] != "NFS4" {
				t.Errorf("acltype = %v", raw["acltype"])
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":      10,
				"name":    "mytmpl",
				"acltype": "NFS4",
			})
		}))

		got, err := c.CreateFilesystemACLTemplate(ctx, &client.FilesystemACLTemplateCreateRequest{
			Name:    "mytmpl",
			ACLType: "NFS4",
			Comment: "test tmpl",
			ACL:     json.RawMessage(`[{"tag":"owner@","perms":{"BASIC":"FULL_CONTROL"}}]`),
		})
		if err != nil {
			t.Fatalf("CreateFilesystemACLTemplate: %v", err)
		}
		if got.ID != 10 {
			t.Errorf("ID = %d", got.ID)
		}
	})

	t.Run("Create 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad acl"})
		}))
		_, err := c.CreateFilesystemACLTemplate(ctx, &client.FilesystemACLTemplateCreateRequest{Name: "x", ACLType: "NFS4"})
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
		_, err := c.CreateFilesystemACLTemplate(ctx, &client.FilesystemACLTemplateCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create invalid JSON response", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("bad"))
		}))
		_, err := c.CreateFilesystemACLTemplate(ctx, &client.FilesystemACLTemplateCreateRequest{Name: "x"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Create raw ACL round-trip", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			acl, ok := raw["acl"].([]interface{})
			if !ok {
				t.Fatalf("acl not array: %T", raw["acl"])
			}
			if len(acl) != 2 {
				t.Errorf("acl len = %d", len(acl))
			}
			writeJSON(w, http.StatusOK, client.FilesystemACLTemplate{ID: 11, Name: "x"})
		}))
		_, _ = c.CreateFilesystemACLTemplate(ctx, &client.FilesystemACLTemplateCreateRequest{
			Name:    "x",
			ACLType: "POSIX1E",
			ACL:     json.RawMessage(`[{"tag":"USER_OBJ"},{"tag":"GROUP_OBJ"}]`),
		})
	})

	t.Run("Update success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/filesystem/acltemplate/id/10") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if raw["name"] != "renamed" {
				t.Errorf("name = %v", raw["name"])
			}
			writeJSON(w, http.StatusOK, client.FilesystemACLTemplate{ID: 10, Name: "renamed"})
		}))
		name := "renamed"
		got, err := c.UpdateFilesystemACLTemplate(ctx, 10, &client.FilesystemACLTemplateUpdateRequest{Name: &name})
		if err != nil {
			t.Fatalf("UpdateFilesystemACLTemplate: %v", err)
		}
		if got.Name != "renamed" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("Update omits nil name", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if _, has := raw["name"]; has {
				t.Errorf("name should be omitted when nil")
			}
			if raw["comment"] != "new comment" {
				t.Errorf("comment = %v", raw["comment"])
			}
			writeJSON(w, http.StatusOK, client.FilesystemACLTemplate{ID: 10})
		}))
		comment := "new comment"
		_, err := c.UpdateFilesystemACLTemplate(ctx, 10, &client.FilesystemACLTemplateUpdateRequest{Comment: &comment})
		if err != nil {
			t.Fatalf("UpdateFilesystemACLTemplate: %v", err)
		}
	})

	t.Run("Update 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "bad"})
		}))
		n := "x"
		_, err := c.UpdateFilesystemACLTemplate(ctx, 10, &client.FilesystemACLTemplateUpdateRequest{Name: &n})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Update builtin fails", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "cannot modify builtin"})
		}))
		n := "x"
		_, err := c.UpdateFilesystemACLTemplate(ctx, 1, &client.FilesystemACLTemplateUpdateRequest{Name: &n})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Delete success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/filesystem/acltemplate/id/10") {
				t.Errorf("path = %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		if err := c.DeleteFilesystemACLTemplate(ctx, 10); err != nil {
			t.Fatalf("DeleteFilesystemACLTemplate: %v", err)
		}
	})

	t.Run("Delete 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
		}))
		err := c.DeleteFilesystemACLTemplate(ctx, 99)
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Delete 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		err := c.DeleteFilesystemACLTemplate(ctx, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Get includes raw acl", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":4,"name":"nfs4tmpl","acltype":"NFS4","acl":[{"tag":"owner@","type":"ALLOW"}]}`))
		}))
		got, err := c.GetFilesystemACLTemplate(ctx, 4)
		if err != nil {
			t.Fatalf("GetFilesystemACLTemplate: %v", err)
		}
		if len(got.ACL) == 0 || !strings.Contains(string(got.ACL), "owner@") {
			t.Errorf("ACL raw = %s", string(got.ACL))
		}
	})
}
