package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetFilesystemACL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.getacl" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"path": "/mnt/tank/data", "uid": 0, "gid": 0,
			"acltype": "POSIX1E", "trivial": false,
			"acl": []interface{}{
				map[string]interface{}{
					"tag": "USER", "id": 1000, "default": false,
					"perms": map[string]interface{}{"READ": true, "WRITE": true, "EXECUTE": false},
				},
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	acl, err := c.GetFilesystemACL(ctx, "/mnt/tank/data")
	if err != nil {
		t.Fatalf("GetFilesystemACL: %v", err)
	}
	if acl.Path != "/mnt/tank/data" || len(acl.ACL) != 1 || !acl.ACL[0].Perms.Write {
		t.Errorf("got %+v", acl)
	}
}

func TestGetFilesystemACL_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFilesystemACL(ctx, "/x")
	if err == nil || !strings.Contains(err.Error(), "getting filesystem ACL") {
		t.Errorf("got %v", err)
	}
}

func TestGetFilesystemACL_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFilesystemACL(ctx, "/x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestSetFilesystemACL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.setacl" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	err := c.SetFilesystemACL(ctx, &types.SetACLRequest{
		Path: "/mnt/tank/data",
		DACL: []types.SetACLEntry{{Tag: "USER", ID: 1000}},
	})
	if err != nil {
		t.Errorf("SetFilesystemACL: %v", err)
	}
}

func TestSetFilesystemACL_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.SetFilesystemACL(ctx, &types.SetACLRequest{Path: "/x"})
	if err == nil || !strings.Contains(err.Error(), "setting filesystem ACL") {
		t.Errorf("got %v", err)
	}
}
