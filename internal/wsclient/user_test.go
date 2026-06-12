package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListUsers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "user.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "username": "root"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	us, err := c.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(us) != 1 {
		t.Errorf("got %+v", us)
	}
}

func TestListUsers_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListUsers(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing users") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListUsers_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListUsers(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "user.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "username": "alice"}, nil
	})
	c, _ := ts.NewClient(ctx)
	u, err := c.GetUser(ctx, 5)
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.Username != "alice" {
		t.Errorf("got %+v", u)
	}
}

func TestGetUser_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetUser(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "getting user") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetUser_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetUser(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetUserByUsername_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{
			map[string]interface{}{"id": 1, "username": "root"},
			map[string]interface{}{"id": 2, "username": "alice"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	u, err := c.GetUserByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if u.ID != 2 {
		t.Errorf("got %+v", u)
	}
}

func TestGetUserByUsername_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetUserByUsername(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found err, got %v", err)
	}
}

func TestGetUserByUsername_listError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetUserByUsername(ctx, "x")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCreateUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "user.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 10, "username": "bob"}, nil
	})
	c, _ := ts.NewClient(ctx)
	u, err := c.CreateUser(ctx, &types.UserCreateRequest{Username: "bob", FullName: "Bob"})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if u.ID != 10 {
		t.Errorf("got %+v", u)
	}
}

func TestCreateUser_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateUser(ctx, &types.UserCreateRequest{Username: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating user") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateUser_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateUser(ctx, &types.UserCreateRequest{Username: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "user.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 10, "full_name": "Bob B"}, nil
	})
	c, _ := ts.NewClient(ctx)
	u, err := c.UpdateUser(ctx, 10, &types.UserUpdateRequest{FullName: "Bob B"})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if u.FullName != "Bob B" {
		t.Errorf("got %+v", u)
	}
}

func TestUpdateUser_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateUser(ctx, 10, &types.UserUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating user") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateUser_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateUser(ctx, 10, &types.UserUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "user.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteUser(ctx, 10); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if !saw {
		t.Error("server did not see user.delete")
	}
}

func TestDeleteUser_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteUser(ctx, 10)
	if err == nil || !strings.Contains(err.Error(), "deleting user") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
