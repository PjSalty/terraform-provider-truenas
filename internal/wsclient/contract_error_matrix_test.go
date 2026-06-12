package wsclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// TestContract_RPCErrorMatrix locks the JSON-RPC error spec to the
// wsclient call surface. For every JSON-RPC error code that TrueNAS
// can return, we assert the client classifies it correctly and the
// returned error preserves the code + errname for downstream IsNotFound
// (or future error classifiers).
//
// JSON-RPC 2.0 reserved codes (-32700..-32600 + -32603) per spec,
// plus TrueNAS' middleware-specific -32000 (concurrent call cap) and
// -32001 (method call error, which carries the errno/errname payload
// that IsNotFound digs into).
func TestContract_RPCErrorMatrix(t *testing.T) {
	cases := []struct {
		name         string
		errCode      int
		errMessage   string
		errData      string // raw JSON payload for RPCError.Data
		wantNotFound bool
		wantErrCode  int
	}{
		// Spec-reserved codes
		{"parse_error", CodeParseError, "Parse error", `null`, false, CodeParseError},
		{"invalid_request", CodeInvalidRequest, "Invalid Request", `null`, false, CodeInvalidRequest},
		{"method_not_found_is_NOT_resource_404", CodeMethodNotFound,
			"Method does not exist", `null`,
			// IsNotFound treats CodeMethodNotFound as NotFound — that's a
			// deliberate semantic choice in wsclient (commented in
			// errors.go line 28) because the server emits this when an
			// endpoint is removed in a TrueNAS version skew.
			true, CodeMethodNotFound},
		{"invalid_params", CodeInvalidParams, "Invalid params", `null`, false, CodeInvalidParams},
		{"internal_error", CodeInternalError, "Internal error", `null`, false, CodeInternalError},

		// TrueNAS-specific
		{"too_many_concurrent", CodeTooManyConcurrent, "Too many concurrent calls",
			`null`, false, CodeTooManyConcurrent},

		// MethodCallError variants — these are the rich ones IsNotFound
		// digs into via errname + reason.
		{"method_call_ENOENT", CodeMethodCallError, "Method call error",
			`{"error":2,"errname":"ENOENT","reason":"file not found"}`,
			true, CodeMethodCallError},
		{"method_call_EINVAL_does_not_exist", CodeMethodCallError, "Method call error",
			`{"error":22,"errname":"EINVAL","reason":"object does not exist"}`,
			true, CodeMethodCallError},
		{"method_call_EINVAL_real_validation", CodeMethodCallError, "Method call error",
			`{"error":22,"errname":"EINVAL","reason":"invalid characters in name"}`,
			false, CodeMethodCallError},
		{"method_call_ValidationErrors_no_such", CodeMethodCallError, "Method call error",
			`{"error":22,"errname":"ValidationErrors","reason":"no such object"}`,
			true, CodeMethodCallError},
		{"method_call_ValidationErrors_real_bad_input", CodeMethodCallError, "Method call error",
			`{"error":22,"errname":"ValidationErrors","reason":"username too short"}`,
			false, CodeMethodCallError},
		{"method_call_EEXIST", CodeMethodCallError, "Method call error",
			`{"error":17,"errname":"EEXIST","reason":"already exists"}`,
			false, CodeMethodCallError},
		{"method_call_EACCES", CodeMethodCallError, "Method call error",
			`{"error":13,"errname":"EACCES","reason":"permission denied"}`,
			false, CodeMethodCallError},
		{"method_call_EPERM", CodeMethodCallError, "Method call error",
			`{"error":1,"errname":"EPERM","reason":"operation not permitted"}`,
			false, CodeMethodCallError},
	}
	for _, tc := range cases {

		t.Run(tc.name, func(t *testing.T) {
			ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
				return nil, &RPCError{
					Code:    tc.errCode,
					Message: tc.errMessage,
					Data:    []byte(tc.errData),
				}
			})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			c, err := ts.NewClient(ctx)
			if err != nil {
				t.Fatalf("dial test server: %v", err)
			}
			// Hit any read-only method; the matrix is about the error
			// classification path, not the specific endpoint.
			_, err = c.GetSystemInfo(ctx)
			if err == nil {
				t.Fatal("expected error from server-side RPCError")
			}
			var rpcErr *RPCError
			if !errors.As(err, &rpcErr) {
				t.Fatalf("error %v does not wrap *RPCError — wsclient lost the error class", err)
			}
			if rpcErr.Code != tc.wantErrCode {
				t.Errorf("RPCError.Code = %d, want %d (wsclient mutated the wire code)",
					rpcErr.Code, tc.wantErrCode)
			}
			gotNotFound := IsNotFound(err)
			if gotNotFound != tc.wantNotFound {
				t.Errorf("IsNotFound = %v, want %v\n  err: %v", gotNotFound, tc.wantNotFound, err)
			}
		})
	}
}

// TestContract_RPCError_RoundTripsErrname asserts that for every
// MethodCallError, the errname / reason fields survive the wsclient's
// error wrapping all the way to the caller. A regression here breaks
// IsNotFound and any future error-classifier (Conflict, RateLimited,
// PermissionDenied) the provider may grow.
func TestContract_RPCError_RoundTripsErrname(t *testing.T) {
	want := struct {
		code    int
		errname string
		reason  string
	}{CodeMethodCallError, "EEXIST", "/mnt/test/foo already exists"}

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{
			Code:    want.code,
			Message: "Method call error",
			Data:    []byte(`{"error":17,"errname":"EEXIST","reason":"/mnt/test/foo already exists"}`),
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_, err = c.GetSystemInfo(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), want.errname) {
		t.Errorf("error string missing errname %q: %v", want.errname, err)
	}
	if !strings.Contains(err.Error(), want.reason) {
		t.Errorf("error string missing reason %q: %v", want.reason, err)
	}
}

// TestContract_RPCError_NoSecretLeak asserts that when TrueNAS' error
// data contains a request-echo with a secret, the wsclient does NOT
// surface that secret in the error string. The 422-equivalent over
// JSON-RPC is a MethodCallError with errname=ValidationErrors and a
// reason that often quotes the offending request body verbatim.
func TestContract_RPCError_NoSecretLeak(t *testing.T) {
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{
			Code:    CodeMethodCallError,
			Message: "Method call error",
			Data: []byte(`{"error":22,"errname":"ValidationErrors",` +
				`"reason":"bad value for password: 's3cret-pw'"}`),
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_, err = c.GetSystemInfo(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "s3cret-pw") {
		t.Errorf("secret leaked through wsclient RPCError: %v", err)
	}
}
