package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetSMBShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.smb.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "path": "/mnt/x", "name": "myshare"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.GetSMBShare(ctx, 5)
	if err != nil {
		t.Fatalf("GetSMBShare: %v", err)
	}
	if s.Name != "myshare" {
		t.Errorf("got %+v", s)
	}
}

func TestGetSMBShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSMBShare(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting SMB share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetSMBShare_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetSMBShare(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestListSMBShares(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.smb.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "name": "a"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	ss, err := c.ListSMBShares(ctx)
	if err != nil {
		t.Fatalf("ListSMBShares: %v", err)
	}
	if len(ss) != 1 {
		t.Errorf("got %+v", ss)
	}
}

func TestListSMBShares_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListSMBShares(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing SMB shares") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListSMBShares_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListSMBShares(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateSMBShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.smb.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 6, "name": "n", "path": "/mnt/p"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.CreateSMBShare(ctx, &types.SMBShareCreateRequest{Path: "/mnt/p", Name: "n"})
	if err != nil {
		t.Fatalf("CreateSMBShare: %v", err)
	}
	if s.ID != 6 {
		t.Errorf("got %+v", s)
	}
}

func TestCreateSMBShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateSMBShare(ctx, &types.SMBShareCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating SMB share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateSMBShare_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateSMBShare(ctx, &types.SMBShareCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateSMBShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.smb.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 6, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.UpdateSMBShare(ctx, 6, &types.SMBShareUpdateRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateSMBShare: %v", err)
	}
	if s.Name != "renamed" {
		t.Errorf("got %+v", s)
	}
}

func TestUpdateSMBShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBShare(ctx, 6, &types.SMBShareUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating SMB share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateSMBShare_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBShare(ctx, 6, &types.SMBShareUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteSMBShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "sharing.smb.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteSMBShare(ctx, 6); err != nil {
		t.Fatalf("DeleteSMBShare: %v", err)
	}
	if !saw {
		t.Error("server did not see sharing.smb.delete")
	}
}

func TestDeleteSMBShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteSMBShare(ctx, 6)
	if err == nil || !strings.Contains(err.Error(), "deleting SMB share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
