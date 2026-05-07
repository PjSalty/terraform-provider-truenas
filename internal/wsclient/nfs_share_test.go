package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetNFSShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.nfs.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 3, "path": "/mnt/tank/data", "enabled": true}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.GetNFSShare(ctx, 3)
	if err != nil {
		t.Fatalf("GetNFSShare: %v", err)
	}
	if s.Path != "/mnt/tank/data" {
		t.Errorf("got %+v", s)
	}
}

func TestGetNFSShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNFSShare(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting NFS share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetNFSShare_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetNFSShare(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestListNFSShares(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.nfs.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "path": "/mnt/a"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	ss, err := c.ListNFSShares(ctx)
	if err != nil {
		t.Fatalf("ListNFSShares: %v", err)
	}
	if len(ss) != 1 {
		t.Errorf("got %+v", ss)
	}
}

func TestListNFSShares_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListNFSShares(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing NFS shares") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListNFSShares_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListNFSShares(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateNFSShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.nfs.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 4, "path": "/mnt/x"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.CreateNFSShare(ctx, &types.NFSShareCreateRequest{Path: "/mnt/x"})
	if err != nil {
		t.Fatalf("CreateNFSShare: %v", err)
	}
	if s.ID != 4 {
		t.Errorf("got %+v", s)
	}
}

func TestCreateNFSShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNFSShare(ctx, &types.NFSShareCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating NFS share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateNFSShare_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateNFSShare(ctx, &types.NFSShareCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateNFSShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "sharing.nfs.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 4, "path": "/mnt/y"}, nil
	})
	c, _ := ts.NewClient(ctx)
	s, err := c.UpdateNFSShare(ctx, 4, &types.NFSShareUpdateRequest{Path: "/mnt/y"})
	if err != nil {
		t.Fatalf("UpdateNFSShare: %v", err)
	}
	if s.Path != "/mnt/y" {
		t.Errorf("got %+v", s)
	}
}

func TestUpdateNFSShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNFSShare(ctx, 4, &types.NFSShareUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating NFS share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateNFSShare_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateNFSShare(ctx, 4, &types.NFSShareUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteNFSShare(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "sharing.nfs.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteNFSShare(ctx, 4); err != nil {
		t.Fatalf("DeleteNFSShare: %v", err)
	}
	if !saw {
		t.Error("server did not see sharing.nfs.delete")
	}
}

func TestDeleteNFSShare_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteNFSShare(ctx, 4)
	if err == nil || !strings.Contains(err.Error(), "deleting NFS share") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
