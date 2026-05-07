package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetKeychainCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "keychaincredential.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "name": "k", "type": "SSH_KEY_PAIR"}, nil
	})
	c, _ := ts.NewClient(ctx)
	cred, err := c.GetKeychainCredential(ctx, 1)
	if err != nil {
		t.Fatalf("GetKeychainCredential: %v", err)
	}
	if cred.Type != "SSH_KEY_PAIR" {
		t.Errorf("got %+v", cred)
	}
}

func TestGetKeychainCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKeychainCredential(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting keychain credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetKeychainCredential_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetKeychainCredential(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateKeychainCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "keychaincredential.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "kk"}, nil
	})
	c, _ := ts.NewClient(ctx)
	cred, err := c.CreateKeychainCredential(ctx, &types.KeychainCredentialCreateRequest{Name: "kk", Type: "SSH_KEY_PAIR"})
	if err != nil {
		t.Fatalf("CreateKeychainCredential: %v", err)
	}
	if cred.ID != 2 {
		t.Errorf("got %+v", cred)
	}
}

func TestCreateKeychainCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateKeychainCredential(ctx, &types.KeychainCredentialCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating keychain credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateKeychainCredential_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateKeychainCredential(ctx, &types.KeychainCredentialCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateKeychainCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "keychaincredential.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "name": "renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	cred, err := c.UpdateKeychainCredential(ctx, 2, &types.KeychainCredentialUpdateRequest{Name: "renamed"})
	if err != nil {
		t.Fatalf("UpdateKeychainCredential: %v", err)
	}
	if cred.Name != "renamed" {
		t.Errorf("got %+v", cred)
	}
}

func TestUpdateKeychainCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKeychainCredential(ctx, 2, &types.KeychainCredentialUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating keychain credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateKeychainCredential_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateKeychainCredential(ctx, 2, &types.KeychainCredentialUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteKeychainCredential(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "keychaincredential.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteKeychainCredential(ctx, 2); err != nil {
		t.Fatalf("DeleteKeychainCredential: %v", err)
	}
	if !saw {
		t.Error("server did not see keychaincredential.delete")
	}
}

func TestDeleteKeychainCredential_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteKeychainCredential(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting keychain credential") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
