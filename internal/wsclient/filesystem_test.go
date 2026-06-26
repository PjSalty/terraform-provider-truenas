package wsclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// dirStat is the canonical filesystem.stat result for a 0755 directory
// owned by uid/gid 1000. mode carries the full st_mode (dir type bits +
// perms), matching what the middleware returns.
func dirStat() map[string]interface{} {
	return map[string]interface{}{
		"realpath": "/mnt/tank/d", "type": "DIRECTORY", "size": 4096,
		"mode": 0o40755, "uid": 1000, "gid": 1000, "acl": false,
		"is_mountpoint": false,
	}
}

func TestMkdir(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.mkdir" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		gotParams = params
		return dirStat(), nil
	})
	c, _ := ts.NewClient(ctx)

	raise := false
	stat, err := c.Mkdir(ctx, &types.MkdirRequest{
		Path:    "/mnt/tank/d",
		Options: &types.MkdirOptions{Mode: "755", RaiseChmodError: &raise},
	})
	if err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	if stat.Mode != 0o40755 || stat.UID != 1000 || stat.GID != 1000 || stat.Type != "DIRECTORY" {
		t.Errorf("decoded stat wrong: %+v", stat)
	}

	// wire shape: one positional object arg {path, options{mode, raise_chmod_error}}
	if len(gotParams) != 1 {
		t.Fatalf("want 1 positional arg, got %d: %v", len(gotParams), gotParams)
	}
	obj, ok := gotParams[0].(map[string]interface{})
	if !ok {
		t.Fatalf("arg 0 is not an object: %T", gotParams[0])
	}
	if obj["path"] != "/mnt/tank/d" {
		t.Errorf("path: got %v", obj["path"])
	}
	opts, ok := obj["options"].(map[string]interface{})
	if !ok {
		t.Fatalf("options missing/not object: %v", obj["options"])
	}
	if opts["mode"] != "755" {
		t.Errorf("options.mode: got %v", opts["mode"])
	}
	if _, present := opts["raise_chmod_error"]; !present {
		t.Errorf("options.raise_chmod_error missing: %v", opts)
	}
}

func TestMkdir_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.Mkdir(ctx, &types.MkdirRequest{Path: "/x"})
	if err == nil || !strings.Contains(err.Error(), "creating directory") {
		t.Errorf("got %v", err)
	}
}

func TestMkdir_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.Mkdir(ctx, &types.MkdirRequest{Path: "/x"})
	if err == nil || !strings.Contains(err.Error(), "parsing mkdir response") {
		t.Errorf("got %v", err)
	}
}

func TestStatFilesystem(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.stat" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		gotParams = params
		return dirStat(), nil
	})
	c, _ := ts.NewClient(ctx)

	stat, err := c.StatFilesystem(ctx, "/mnt/tank/d")
	if err != nil {
		t.Fatalf("StatFilesystem: %v", err)
	}
	if stat.Mode != 0o40755 || stat.UID != 1000 {
		t.Errorf("decoded stat wrong: %+v", stat)
	}
	// wire shape: positional [path]
	if len(gotParams) != 1 || gotParams[0] != "/mnt/tank/d" {
		t.Errorf("want positional [path], got %v", gotParams)
	}
}

func TestStatFilesystem_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		// middleware raises CallError with errname ENOENT on a missing path
		return nil, &RPCError{Code: CodeMethodCallError, Message: "ENOENT", Data: json.RawMessage(`{"errname":"ENOENT"}`)}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.StatFilesystem(ctx, "/mnt/tank/missing")
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if !IsNotFound(err) {
		t.Errorf("IsNotFound should be true for ENOENT, got err %v", err)
	}
}

func TestStatFilesystem_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.StatFilesystem(ctx, "/x")
	if err == nil || !strings.Contains(err.Error(), "parsing stat response") {
		t.Errorf("got %v", err)
	}
}

func TestSetFilesystemPerm(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "filesystem.setperm":
			gotParams = params
			return int64(91), nil // job id
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": int64(91), "state": "SUCCESS", "result": nil, "error": "",
			}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)

	mode := "750"
	uid := 1000
	if err := c.SetFilesystemPerm(ctx, &types.SetPermRequest{Path: "/mnt/tank/d", Mode: &mode, UID: &uid}); err != nil {
		t.Fatalf("SetFilesystemPerm: %v", err)
	}
	// wire shape: one positional object arg {path, mode, uid}
	if len(gotParams) != 1 {
		t.Fatalf("want 1 positional arg, got %d", len(gotParams))
	}
	obj, ok := gotParams[0].(map[string]interface{})
	if !ok {
		t.Fatalf("arg 0 not an object: %T", gotParams[0])
	}
	if obj["path"] != "/mnt/tank/d" || obj["mode"] != "750" {
		t.Errorf("setperm args wrong: %v", obj)
	}
}

func TestSetFilesystemPerm_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.SetFilesystemPerm(ctx, &types.SetPermRequest{Path: "/x"})
	if err == nil || !strings.Contains(err.Error(), "setting perms on") {
		t.Errorf("got %v", err)
	}
}
