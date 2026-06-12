package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetACMEDNSAuthenticator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "acme.dns.authenticator.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 1, "name": "cf",
			"attributes": map[string]interface{}{"authenticator": "cloudflare", "api_key": "***"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	auth, err := c.GetACMEDNSAuthenticator(ctx, 1)
	if err != nil {
		t.Fatalf("GetACMEDNSAuthenticator: %v", err)
	}
	if auth.Name != "cf" {
		t.Errorf("got %+v", auth)
	}
}

func TestGetACMEDNSAuthenticator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetACMEDNSAuthenticator(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting ACME DNS authenticator") {
		t.Errorf("got %v", err)
	}
}

func TestGetACMEDNSAuthenticator_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetACMEDNSAuthenticator(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestCreateACMEDNSAuthenticator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "acme.dns.authenticator.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 7, "name": "cf",
			"attributes": map[string]interface{}{"authenticator": "cloudflare"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	auth, err := c.CreateACMEDNSAuthenticator(ctx, &types.ACMEDNSAuthenticatorCreateRequest{
		Name: "cf", Attributes: map[string]interface{}{"authenticator": "cloudflare"},
	})
	if err != nil {
		t.Fatalf("CreateACMEDNSAuthenticator: %v", err)
	}
	if auth.ID != 7 {
		t.Errorf("got %+v", auth)
	}
}

func TestCreateACMEDNSAuthenticator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "missing field"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateACMEDNSAuthenticator(ctx, &types.ACMEDNSAuthenticatorCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating ACME DNS authenticator") {
		t.Errorf("got %v", err)
	}
}

func TestCreateACMEDNSAuthenticator_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateACMEDNSAuthenticator(ctx, &types.ACMEDNSAuthenticatorCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateACMEDNSAuthenticator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "acme.dns.authenticator.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 7, "name": "cf-renamed"}, nil
	})
	c, _ := ts.NewClient(ctx)
	auth, err := c.UpdateACMEDNSAuthenticator(ctx, 7, &types.ACMEDNSAuthenticatorUpdateRequest{
		Name: "cf-renamed",
	})
	if err != nil {
		t.Fatalf("UpdateACMEDNSAuthenticator: %v", err)
	}
	if auth.Name != "cf-renamed" {
		t.Errorf("got %+v", auth)
	}
}

func TestUpdateACMEDNSAuthenticator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateACMEDNSAuthenticator(ctx, 7, &types.ACMEDNSAuthenticatorUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating ACME DNS authenticator") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateACMEDNSAuthenticator_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateACMEDNSAuthenticator(ctx, 7, &types.ACMEDNSAuthenticatorUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteACMEDNSAuthenticator(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "acme.dns.authenticator.delete" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteACMEDNSAuthenticator(ctx, 7); err != nil {
		t.Errorf("DeleteACMEDNSAuthenticator: %v", err)
	}
}

func TestDeleteACMEDNSAuthenticator_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteACMEDNSAuthenticator(ctx, 7)
	if err == nil || !strings.Contains(err.Error(), "deleting ACME DNS authenticator") {
		t.Errorf("got %v", err)
	}
}
