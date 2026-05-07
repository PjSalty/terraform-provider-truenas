package wsclient

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetFilesystemACLTemplate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.acltemplate.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1, "name": "RESTRICTED", "acltype": "POSIX1E",
			"comment": "restrictive", "builtin": true,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	tmpl, err := c.GetFilesystemACLTemplate(ctx, 1)
	if err != nil {
		t.Fatalf("GetFilesystemACLTemplate: %v", err)
	}
	if tmpl.Name != "RESTRICTED" {
		t.Errorf("got %+v", tmpl)
	}
}

func TestGetFilesystemACLTemplate_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFilesystemACLTemplate(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting filesystem ACL template") {
		t.Errorf("got %v", err)
	}
}

func TestGetFilesystemACLTemplate_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetFilesystemACLTemplate(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestCreateFilesystemACLTemplate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.acltemplate.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "name": "MY_TMPL", "acltype": "POSIX1E"}, nil
	})
	c, _ := ts.NewClient(ctx)
	tmpl, err := c.CreateFilesystemACLTemplate(ctx, &types.FilesystemACLTemplateCreateRequest{
		Name: "MY_TMPL", ACLType: "POSIX1E", ACL: json.RawMessage(`[]`),
	})
	if err != nil {
		t.Fatalf("CreateFilesystemACLTemplate: %v", err)
	}
	if tmpl.ID != 9 {
		t.Errorf("got %+v", tmpl)
	}
}

func TestCreateFilesystemACLTemplate_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateFilesystemACLTemplate(ctx, &types.FilesystemACLTemplateCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating filesystem ACL template") {
		t.Errorf("got %v", err)
	}
}

func TestCreateFilesystemACLTemplate_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateFilesystemACLTemplate(ctx, &types.FilesystemACLTemplateCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateFilesystemACLTemplate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.acltemplate.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 9, "name": "RENAMED"}, nil
	})
	c, _ := ts.NewClient(ctx)
	name := "RENAMED"
	tmpl, err := c.UpdateFilesystemACLTemplate(ctx, 9, &types.FilesystemACLTemplateUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdateFilesystemACLTemplate: %v", err)
	}
	if tmpl.Name != "RENAMED" {
		t.Errorf("got %+v", tmpl)
	}
}

func TestUpdateFilesystemACLTemplate_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateFilesystemACLTemplate(ctx, 9, &types.FilesystemACLTemplateUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating filesystem ACL template") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateFilesystemACLTemplate_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateFilesystemACLTemplate(ctx, 9, &types.FilesystemACLTemplateUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteFilesystemACLTemplate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "filesystem.acltemplate.delete" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteFilesystemACLTemplate(ctx, 9); err != nil {
		t.Errorf("DeleteFilesystemACLTemplate: %v", err)
	}
}

func TestDeleteFilesystemACLTemplate_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteFilesystemACLTemplate(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "deleting filesystem ACL template") {
		t.Errorf("got %v", err)
	}
}
