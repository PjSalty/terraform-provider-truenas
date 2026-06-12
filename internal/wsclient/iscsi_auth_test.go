package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetISCSIAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.auth.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "tag": 1, "user": "u"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetISCSIAuth(ctx, 1)
	if err != nil {
		t.Fatalf("GetISCSIAuth: %v", err)
	}
	if r.User != "u" {
		t.Errorf("got %+v", r)
	}
}

func TestGetISCSIAuth_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIAuth(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting iSCSI auth") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetISCSIAuth_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIAuth(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateISCSIAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.auth.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "tag": 2}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateISCSIAuth(ctx, &types.ISCSIAuthCreateRequest{Tag: 2, User: "u", Secret: "passwordlongenough"})
	if err != nil {
		t.Fatalf("CreateISCSIAuth: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateISCSIAuth_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIAuth(ctx, &types.ISCSIAuthCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating iSCSI auth") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateISCSIAuth_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIAuth(ctx, &types.ISCSIAuthCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateISCSIAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.auth.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "user": "u2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	user := "u2"
	r, err := c.UpdateISCSIAuth(ctx, 2, &types.ISCSIAuthUpdateRequest{User: &user})
	if err != nil {
		t.Fatalf("UpdateISCSIAuth: %v", err)
	}
	if r.User != "u2" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateISCSIAuth_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIAuth(ctx, 2, &types.ISCSIAuthUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI auth") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateISCSIAuth_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIAuth(ctx, 2, &types.ISCSIAuthUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteISCSIAuth(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "iscsi.auth.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteISCSIAuth(ctx, 2); err != nil {
		t.Fatalf("DeleteISCSIAuth: %v", err)
	}
	if !saw {
		t.Error("server did not see iscsi.auth.delete")
	}
}

func TestDeleteISCSIAuth_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteISCSIAuth(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting iSCSI auth") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
