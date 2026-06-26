package resources

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// mkdirParents must walk the ancestor chain below the leaf, skip /mnt and
// any ancestor that already exists, and mkdir only the missing ones in
// shallow->deep order. The leaf itself is created by Create, not here.
func TestDirectoryResource_mkdirParents(t *testing.T) {
	ctx := context.Background()
	existing := map[string]bool{"/mnt/tank": true} // pool root already exists
	var mkdirs []string

	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "filesystem.stat":
			p, _ := params[0].(string)
			if existing[p] {
				return map[string]interface{}{"realpath": p, "type": "DIRECTORY", "mode": 0o40755}, nil
			}
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "ENOENT", Data: json.RawMessage(`{"errname":"ENOENT"}`)}
		case "filesystem.mkdir":
			obj, _ := params[0].(map[string]interface{})
			p, _ := obj["path"].(string)
			mkdirs = append(mkdirs, p)
			existing[p] = true
			return map[string]interface{}{"realpath": p, "type": "DIRECTORY", "mode": 0o40755}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	r := &DirectoryResource{client: c}

	if err := r.mkdirParents(ctx, "/mnt/tank/a/b/c", "755"); err != nil {
		t.Fatalf("mkdirParents: %v", err)
	}

	// /mnt skipped (never statted/created), /mnt/tank already exists, so
	// only a and a/b are created, in order. The leaf c is not created here.
	want := []string{"/mnt/tank/a", "/mnt/tank/a/b"}
	if !reflect.DeepEqual(mkdirs, want) {
		t.Errorf("created %v, want %v", mkdirs, want)
	}
}

// a top-level path has no parent to create; mkdirParents returns nil
// without calling the server.
func TestDirectoryResource_mkdirParents_noParent(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		t.Errorf("server should not be called for a top-level path, got %s", method)
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	if err := r.mkdirParents(ctx, "/d", "755"); err != nil {
		t.Fatalf("mkdirParents(top-level): %v", err)
	}
}

// a failing mkdir on a missing ancestor surfaces as an error.
func TestDirectoryResource_mkdirParents_mkdirError(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "filesystem.stat":
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Data: json.RawMessage(`{"errname":"ENOENT"}`)}
		case "filesystem.mkdir":
			return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "boom"}
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	if err := r.mkdirParents(ctx, "/mnt/tank/a/b", "755"); err == nil {
		t.Fatal("mkdirParents should error when an ancestor mkdir fails")
	}
}
