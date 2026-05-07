package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetISCSIInitiator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.initiator.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "comment": "g"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetISCSIInitiator(ctx, 1)
	if err != nil {
		t.Fatalf("GetISCSIInitiator: %v", err)
	}
	if r.Comment != "g" {
		t.Errorf("got %+v", r)
	}
}

func TestGetISCSIInitiator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIInitiator(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting iSCSI initiator") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetISCSIInitiator_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetISCSIInitiator(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateISCSIInitiator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.initiator.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "comment": "new"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateISCSIInitiator(ctx, &types.ISCSIInitiatorCreateRequest{Comment: "new"})
	if err != nil {
		t.Fatalf("CreateISCSIInitiator: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateISCSIInitiator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIInitiator(ctx, &types.ISCSIInitiatorCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating iSCSI initiator") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateISCSIInitiator_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateISCSIInitiator(ctx, &types.ISCSIInitiatorCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateISCSIInitiator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "iscsi.initiator.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "comment": "updated"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateISCSIInitiator(ctx, 2, &types.ISCSIInitiatorUpdateRequest{Comment: "updated"})
	if err != nil {
		t.Fatalf("UpdateISCSIInitiator: %v", err)
	}
	if r.Comment != "updated" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateISCSIInitiator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIInitiator(ctx, 2, &types.ISCSIInitiatorUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating iSCSI initiator") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateISCSIInitiator_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateISCSIInitiator(ctx, 2, &types.ISCSIInitiatorUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteISCSIInitiator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "iscsi.initiator.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteISCSIInitiator(ctx, 2); err != nil {
		t.Fatalf("DeleteISCSIInitiator: %v", err)
	}
	if !saw {
		t.Error("server did not see iscsi.initiator.delete")
	}
}

func TestDeleteISCSIInitiator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteISCSIInitiator(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting iSCSI initiator") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
