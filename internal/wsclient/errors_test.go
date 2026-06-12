package wsclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestRPCError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *RPCError
		want string
	}{
		{
			name: "code+message only",
			err:  &RPCError{Code: -32601, Message: "Method not found"},
			want: "truenas rpc error -32601: Method not found",
		},
		{
			name: "with errname only",
			err:  &RPCError{Code: -32001, Message: "Method call error", Data: json.RawMessage(`{"errname":"EINVAL"}`)},
			want: "truenas rpc error -32001: Method call error (EINVAL)",
		},
		{
			name: "with reason only",
			err:  &RPCError{Code: -32001, Message: "Method call error", Data: json.RawMessage(`{"reason":"bad input"}`)},
			want: "truenas rpc error -32001: Method call error: bad input",
		},
		{
			name: "with both errname and reason",
			err:  &RPCError{Code: -32001, Message: "Method call error", Data: json.RawMessage(`{"errname":"EINVAL","reason":"bad input"}`)},
			want: "truenas rpc error -32001: Method call error: bad input (EINVAL)",
		},
		{
			name: "malformed data",
			err:  &RPCError{Code: -32001, Message: "X", Data: json.RawMessage(`not json`)},
			want: "truenas rpc error -32001: X",
		},
		{
			name: "nil error",
			err:  nil,
			want: "<nil RPCError>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error: got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRPCError_errnameAndReason_emptyData(t *testing.T) {
	e := &RPCError{Code: -32601, Message: "x"}
	errname, reason := e.errnameAndReason()
	if errname != "" || reason != "" {
		t.Errorf("expected both empty, got (%q, %q)", errname, reason)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"non-RPC error", errors.New("plain"), false},
		{"wrapped non-RPC", fmt.Errorf("wrapped: %w", errors.New("inner")), false},
		{"CodeMethodNotFound", &RPCError{Code: CodeMethodNotFound}, true},
		{"CodeMethodCallError ENOENT", &RPCError{Code: CodeMethodCallError, Data: json.RawMessage(`{"errname":"ENOENT"}`)}, true},
		{"CodeMethodCallError ValidationErrors does not exist",
			&RPCError{Code: CodeMethodCallError, Data: json.RawMessage(`{"errname":"ValidationErrors","reason":"object 'foo' does not exist"}`)}, true},
		{"CodeMethodCallError EINVAL not found",
			&RPCError{Code: CodeMethodCallError, Data: json.RawMessage(`{"errname":"EINVAL","reason":"target not found"}`)}, true},
		{"CodeMethodCallError EINVAL no such",
			&RPCError{Code: CodeMethodCallError, Data: json.RawMessage(`{"errname":"EINVAL","reason":"no such instance"}`)}, true},
		{"CodeMethodCallError ValidationErrors something else",
			&RPCError{Code: CodeMethodCallError, Data: json.RawMessage(`{"errname":"ValidationErrors","reason":"name too long"}`)}, false},
		{"CodeMethodCallError unknown errname",
			&RPCError{Code: CodeMethodCallError, Data: json.RawMessage(`{"errname":"OTHER","reason":"does not exist"}`)}, false},
		{"CodeInternalError", &RPCError{Code: CodeInternalError}, false},
		{"CodeInvalidParams", &RPCError{Code: CodeInvalidParams}, false},
		{"wrapped RPCError", fmt.Errorf("wrap: %w", &RPCError{Code: CodeMethodNotFound}), true},
		{"empty data on call error", &RPCError{Code: CodeMethodCallError}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	if ErrConnectionLost.Error() == "" {
		t.Error("ErrConnectionLost has empty message")
	}
	if ErrShuttingDown.Error() == "" {
		t.Error("ErrShuttingDown has empty message")
	}
	if errors.Is(ErrConnectionLost, ErrShuttingDown) {
		t.Error("ErrConnectionLost should not match ErrShuttingDown")
	}
}
