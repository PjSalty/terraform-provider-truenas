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

func TestFilesystemACL_CRUD(t *testing.T) {
	ctx := context.Background()

	t.Run("Get success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/filesystem/getacl") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.GetACLRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Path != "/mnt/tank/data" {
				t.Errorf("Path = %q", req.Path)
			}
			writeJSON(w, http.StatusOK, client.FilesystemACL{
				Path:    "/mnt/tank/data",
				UID:     1000,
				GID:     1000,
				ACLType: "POSIX1E",
				Trivial: false,
				ACL: []client.ACLEntry{
					{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: true, Execute: true}},
				},
			})
		}))

		got, err := c.GetFilesystemACL(ctx, "/mnt/tank/data")
		if err != nil {
			t.Fatalf("GetFilesystemACL: %v", err)
		}
		if got.Path != "/mnt/tank/data" {
			t.Errorf("Path = %q", got.Path)
		}
		if got.ACLType != "POSIX1E" {
			t.Errorf("ACLType = %q", got.ACLType)
		}
		if len(got.ACL) != 1 || !got.ACL[0].Perms.Read {
			t.Errorf("ACL = %+v", got.ACL)
		}
	})

	t.Run("Get 404", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "no such path"})
		}))
		_, err := c.GetFilesystemACL(ctx, "/mnt/missing")
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Get 422", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid path"})
		}))
		_, err := c.GetFilesystemACL(ctx, "bad")
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Get 500", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		_, err := c.GetFilesystemACL(ctx, "/mnt/x")
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Get invalid JSON", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not json"))
		}))
		_, err := c.GetFilesystemACL(ctx, "/mnt/x")
		if err == nil {
			t.Fatal("expected JSON parse error")
		}
	})

	t.Run("Get decodes user/group pointers", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"path":"/p","uid":0,"gid":0,"acltype":"NFS4","trivial":true,"user":"root","group":"wheel","acl":[]}`))
		}))
		got, err := c.GetFilesystemACL(ctx, "/p")
		if err != nil {
			t.Fatalf("GetFilesystemACL: %v", err)
		}
		if got.User == nil || *got.User != "root" {
			t.Errorf("User = %+v", got.User)
		}
		if got.Group == nil || *got.Group != "wheel" {
			t.Errorf("Group = %+v", got.Group)
		}
	})

	t.Run("Get decodes NFS4 with multiple entries", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, client.FilesystemACL{
				Path:    "/mnt/tank",
				ACLType: "NFS4",
				ACL: []client.ACLEntry{
					{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true}},
					{Tag: "GROUP_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Execute: true}},
					{Tag: "OTHER", ID: -1, Perms: client.ACLPerms{}},
				},
			})
		}))
		got, err := c.GetFilesystemACL(ctx, "/mnt/tank")
		if err != nil {
			t.Fatalf("GetFilesystemACL: %v", err)
		}
		if len(got.ACL) != 3 {
			t.Errorf("len = %d", len(got.ACL))
		}
	})

	t.Run("Set success", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("method = %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/filesystem/setacl") {
				t.Errorf("path = %s", r.URL.Path)
			}
			body, _ := io.ReadAll(r.Body)
			var req client.SetACLRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if req.Path != "/mnt/tank/data" {
				t.Errorf("Path = %q", req.Path)
			}
			if len(req.DACL) != 1 {
				t.Errorf("DACL len = %d", len(req.DACL))
			}
			if req.UID == nil || *req.UID != 1000 {
				t.Errorf("UID = %+v", req.UID)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("null"))
		}))
		uid := 1000
		gid := 1000
		err := c.SetFilesystemACL(ctx, &client.SetACLRequest{
			Path:    "/mnt/tank/data",
			ACLType: "POSIX1E",
			UID:     &uid,
			GID:     &gid,
			DACL: []client.SetACLEntry{
				{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: true, Execute: true}},
			},
		})
		if err != nil {
			t.Fatalf("SetFilesystemACL: %v", err)
		}
	})

	t.Run("Set 422 bad acl", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "invalid acl"})
		}))
		err := c.SetFilesystemACL(ctx, &client.SetACLRequest{Path: "/p"})
		if err == nil {
			t.Fatal("expected error")
		}
		var apiErr *client.APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 422 {
			t.Errorf("err = %v", err)
		}
	})

	t.Run("Set 500 error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
		}))
		err := c.SetFilesystemACL(ctx, &client.SetACLRequest{Path: "/p"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("Set 404 error", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "path missing"})
		}))
		err := c.SetFilesystemACL(ctx, &client.SetACLRequest{Path: "/p"})
		if err == nil {
			t.Fatal("expected error")
		}
		if !client.IsNotFound(err) {
			t.Errorf("IsNotFound = false")
		}
	})

	t.Run("Set omits nil uid/gid", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var raw map[string]interface{}
			_ = json.Unmarshal(body, &raw)
			if _, has := raw["uid"]; has {
				t.Errorf("uid should be omitted")
			}
			if _, has := raw["gid"]; has {
				t.Errorf("gid should be omitted")
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("null"))
		}))
		err := c.SetFilesystemACL(ctx, &client.SetACLRequest{
			Path: "/p",
			DACL: []client.SetACLEntry{},
		})
		if err != nil {
			t.Fatalf("SetFilesystemACL: %v", err)
		}
	})

	t.Run("Set marshals multiple dacl entries", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.SetACLRequest
			_ = json.Unmarshal(body, &req)
			if len(req.DACL) != 3 {
				t.Errorf("DACL len = %d", len(req.DACL))
			}
			if req.DACL[1].Tag != "GROUP_OBJ" {
				t.Errorf("DACL[1].Tag = %q", req.DACL[1].Tag)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("null"))
		}))
		err := c.SetFilesystemACL(ctx, &client.SetACLRequest{
			Path:    "/p",
			ACLType: "POSIX1E",
			DACL: []client.SetACLEntry{
				{Tag: "USER_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Write: true, Execute: true}},
				{Tag: "GROUP_OBJ", ID: -1, Perms: client.ACLPerms{Read: true, Execute: true}},
				{Tag: "OTHER", ID: -1, Perms: client.ACLPerms{Read: true}},
			},
		})
		if err != nil {
			t.Fatalf("SetFilesystemACL: %v", err)
		}
	})

	t.Run("Get path in request body", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req client.GetACLRequest
			_ = json.Unmarshal(body, &req)
			if req.Path != "/weird path/with spaces" {
				t.Errorf("Path = %q", req.Path)
			}
			writeJSON(w, http.StatusOK, client.FilesystemACL{Path: req.Path})
		}))
		_, err := c.GetFilesystemACL(ctx, "/weird path/with spaces")
		if err != nil {
			t.Fatalf("GetFilesystemACL: %v", err)
		}
	})

	t.Run("Get path escaping not applied", func(t *testing.T) {
		_, c := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasSuffix(r.URL.Path, "/filesystem/getacl") {
				t.Errorf("path = %s", r.URL.Path)
			}
			writeJSON(w, http.StatusOK, client.FilesystemACL{})
		}))
		_, _ = c.GetFilesystemACL(ctx, "/mnt/tank")
	})
}
