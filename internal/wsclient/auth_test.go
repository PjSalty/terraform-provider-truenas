package wsclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestAuthenticate_objectShape(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Some TrueNAS firmware revisions return an object instead of bool.
	// The handshake must accept that shape.
	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "auth.login_with_api_key" {
			return map[string]interface{}{"username": "root"}, nil
		}
		return nil, nil
	}, TestServerOptions{SkipAuth: true})

	c, err := New(ctx, ts.URL(), "k", true)
	if err != nil {
		t.Errorf("New: %v", err)
	}
	if c != nil {
		_ = c.Close()
	}
}

func TestAuthenticate_unexpectedShape(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "auth.login_with_api_key" {
			// Server returns a string, which is neither bool nor object.
			return "what", nil
		}
		return nil, nil
	}, TestServerOptions{SkipAuth: true})

	_, err := New(ctx, ts.URL(), "k", true)
	if err == nil {
		t.Fatal("expected unexpected-shape error")
	}
	if !strings.Contains(err.Error(), "unexpected result") {
		t.Errorf("error = %v, want substring 'unexpected result'", err)
	}
}

func TestAuthenticate_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServerWithOptions(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "server boom"}
	}, TestServerOptions{SkipAuth: true})

	_, err := New(ctx, ts.URL(), "k", true)
	if err == nil {
		t.Fatal("expected server error")
	}
	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Errorf("expected wrapped *RPCError, got %v", err)
	}
}
