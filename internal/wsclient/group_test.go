package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListGroups(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "group.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "name": "wheel"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	gs, err := c.ListGroups(ctx)
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}
	if len(gs) != 1 {
		t.Errorf("got %+v", gs)
	}
}

func TestListGroups_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListGroups(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing groups") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListGroups_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListGroups(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "group.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "name": "ops"}, nil
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.GetGroup(ctx, 5)
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if g.Name != "ops" {
		t.Errorf("got %+v", g)
	}
}

func TestGetGroup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetGroup(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "getting group") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetGroup_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetGroup(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetGroupByName_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{map[string]interface{}{"id": 3, "name": "found"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.GetGroupByName(ctx, "found")
	if err != nil {
		t.Fatalf("GetGroupByName: %v", err)
	}
	if g.ID != 3 {
		t.Errorf("got %+v", g)
	}
}

func TestGetGroupByName_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetGroupByName(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found err, got %v", err)
	}
}

func TestGetGroupByName_listError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetGroupByName(ctx, "x")
	if err == nil {
		t.Error("expected error")
	}
}

// Group create/update may return either the full object (ID present) or
// just the ID number (TrueNAS pre-25 behavior). We test both branches.

func TestCreateGroup_object(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "group.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 7, "name": "ops"}, nil
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.CreateGroup(ctx, &types.GroupCreateRequest{Name: "ops"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.ID != 7 {
		t.Errorf("got %+v", g)
	}
}

func TestCreateGroup_idOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "group.create":
			return 8, nil
		case "group.get_instance":
			return map[string]interface{}{"id": 8, "name": "fresh"}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.CreateGroup(ctx, &types.GroupCreateRequest{Name: "fresh"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if g.Name != "fresh" {
		t.Errorf("got %+v", g)
	}
}

func TestCreateGroup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateGroup(ctx, &types.GroupCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating group") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateGroup_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateGroup(ctx, &types.GroupCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateGroup_object(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "group.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 7, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.UpdateGroup(ctx, 7, &types.GroupUpdateRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if g.Name != "renamed" {
		t.Errorf("got %+v", g)
	}
}

func TestUpdateGroup_idOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "group.update":
			return 7, nil
		case "group.get_instance":
			return map[string]interface{}{"id": 7, "name": "after"}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	g, err := c.UpdateGroup(ctx, 7, &types.GroupUpdateRequest{})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}
	if g.Name != "after" {
		t.Errorf("got %+v", g)
	}
}

func TestUpdateGroup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateGroup(ctx, 7, &types.GroupUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating group") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateGroup_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateGroup(ctx, 7, &types.GroupUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteGroup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "group.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteGroup(ctx, 7); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if !saw {
		t.Error("server did not see group.delete")
	}
}

func TestDeleteGroup_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteGroup(ctx, 7)
	if err == nil || !strings.Contains(err.Error(), "deleting group") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
