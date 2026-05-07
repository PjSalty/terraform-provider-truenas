package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListPrivileges(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "privilege.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "name": "p1"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	ps, err := c.ListPrivileges(ctx)
	if err != nil {
		t.Fatalf("ListPrivileges: %v", err)
	}
	if len(ps) != 1 {
		t.Errorf("got %+v", ps)
	}
}

func TestListPrivileges_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListPrivileges(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing privileges") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListPrivileges_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListPrivileges(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetPrivilege(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "privilege.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "name": "p5"}, nil
	})
	c, _ := ts.NewClient(ctx)
	p, err := c.GetPrivilege(ctx, 5)
	if err != nil {
		t.Fatalf("GetPrivilege: %v", err)
	}
	if p.Name != "p5" {
		t.Errorf("got %+v", p)
	}
}

func TestGetPrivilege_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPrivilege(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "getting privilege") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetPrivilege_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetPrivilege(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreatePrivilege(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "privilege.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 6, "name": "new"}, nil
	})
	c, _ := ts.NewClient(ctx)
	p, err := c.CreatePrivilege(ctx, &types.PrivilegeCreateRequest{Name: "new"})
	if err != nil {
		t.Fatalf("CreatePrivilege: %v", err)
	}
	if p.ID != 6 {
		t.Errorf("got %+v", p)
	}
}

func TestCreatePrivilege_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreatePrivilege(ctx, &types.PrivilegeCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating privilege") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreatePrivilege_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreatePrivilege(ctx, &types.PrivilegeCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdatePrivilege(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "privilege.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 6, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	name := "renamed"
	p, err := c.UpdatePrivilege(ctx, 6, &types.PrivilegeUpdateRequest{Name: &name})
	if err != nil {
		t.Fatalf("UpdatePrivilege: %v", err)
	}
	if p.Name != "renamed" {
		t.Errorf("got %+v", p)
	}
}

func TestUpdatePrivilege_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdatePrivilege(ctx, 6, &types.PrivilegeUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating privilege") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdatePrivilege_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdatePrivilege(ctx, 6, &types.PrivilegeUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeletePrivilege(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "privilege.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeletePrivilege(ctx, 6); err != nil {
		t.Fatalf("DeletePrivilege: %v", err)
	}
	if !saw {
		t.Error("server did not see privilege.delete")
	}
}

func TestDeletePrivilege_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeletePrivilege(ctx, 6)
	if err == nil || !strings.Contains(err.Error(), "deleting privilege") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
