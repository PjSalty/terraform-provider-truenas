package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetCloudSyncCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.credentials.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "c1"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetCloudSyncCredential(ctx, 1)
	if err != nil {
		t.Fatalf("GetCloudSyncCredential: %v", err)
	}
	if r.Name != "c1" {
		t.Errorf("got %+v", r)
	}
}

func TestGetCloudSyncCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudSyncCredential(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting cloud sync credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetCloudSyncCredential_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudSyncCredential(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestListCloudSyncCredentials(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.credentials.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{map[string]interface{}{"id": 1, "name": "c"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	cs, err := c.ListCloudSyncCredentials(ctx)
	if err != nil {
		t.Fatalf("ListCloudSyncCredentials: %v", err)
	}
	if len(cs) != 1 {
		t.Errorf("got %+v", cs)
	}
}

func TestListCloudSyncCredentials_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListCloudSyncCredentials(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing cloud sync credentials") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListCloudSyncCredentials_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListCloudSyncCredentials(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetCloudSyncCredentialByName_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{map[string]interface{}{"id": 3, "name": "found"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetCloudSyncCredentialByName(ctx, "found")
	if err != nil {
		t.Fatalf("GetCloudSyncCredentialByName: %v", err)
	}
	if r.ID != 3 {
		t.Errorf("got %+v", r)
	}
}

func TestGetCloudSyncCredentialByName_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudSyncCredentialByName(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not-found err, got %v", err)
	}
}

func TestGetCloudSyncCredentialByName_listError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCloudSyncCredentialByName(ctx, "x")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCreateCloudSyncCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.credentials.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 4, "name": "n4"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateCloudSyncCredential(ctx, &types.CloudSyncCredentialCreateRequest{Name: "n4"})
	if err != nil {
		t.Fatalf("CreateCloudSyncCredential: %v", err)
	}
	if r.ID != 4 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateCloudSyncCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCloudSyncCredential(ctx, &types.CloudSyncCredentialCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating cloud sync credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateCloudSyncCredential_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCloudSyncCredential(ctx, &types.CloudSyncCredentialCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateCloudSyncCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cloudsync.credentials.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 4, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateCloudSyncCredential(ctx, 4, &types.CloudSyncCredentialUpdateRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateCloudSyncCredential: %v", err)
	}
	if r.Name != "renamed" {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateCloudSyncCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCloudSyncCredential(ctx, 4, &types.CloudSyncCredentialUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating cloud sync credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateCloudSyncCredential_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCloudSyncCredential(ctx, 4, &types.CloudSyncCredentialUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteCloudSyncCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "cloudsync.credentials.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteCloudSyncCredential(ctx, 4); err != nil {
		t.Fatalf("DeleteCloudSyncCredential: %v", err)
	}
	if !saw {
		t.Error("server did not see cloudsync.credentials.delete")
	}
}

func TestDeleteCloudSyncCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteCloudSyncCredential(ctx, 4)
	if err == nil || !strings.Contains(err.Error(), "deleting cloud sync credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
