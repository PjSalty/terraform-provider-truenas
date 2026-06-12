package wsclient

import (
	"encoding/json"
	"errors"
	"testing"
)

// FuzzIsNotFound checks that IsNotFound never panics no matter what
// the JSON-RPC error envelope contains. Surface is small but error
// envelopes from a malicious or buggy upstream could carry surprising
// shapes (NaN-like numeric codes, unicode bombs in messages, deeply
// nested data blobs).
//
// The IsNotFound classifier is on the hot path for Delete idempotency
// — every resource Delete handler calls it. A panic here would turn
// the destroy step into a stack trace, so we drive it through random
// inputs to surface anything go-fuzz can find.
func FuzzIsNotFound(f *testing.F) {
	// Seed corpus: every error code IsNotFound branches on, plus a
	// few representative data shapes.
	seeds := []struct {
		code    int
		message string
		data    string
	}{
		{-32601, "Method not found", ""},
		{-32001, "Method call error", `{"errname":"ENOENT","reason":"x does not exist"}`},
		{-32001, "Method call error", `{"errname":"EBUSY","reason":"Rate Limit"}`},
		{-32001, "Method call error", `{"errname":"ValidationErrors","reason":"no such object"}`},
		{-32602, "Invalid params: [ENOENT] None: X does not exist", ""},
		{-32602, "Invalid params: count must be positive", ""},
		{-32700, "Parse error", ""},
		{0, "", ""},
	}
	for _, s := range seeds {
		f.Add(s.code, s.message, s.data)
	}

	f.Fuzz(func(t *testing.T, code int, message string, data string) {
		err := &RPCError{
			Code:    code,
			Message: message,
		}
		if data != "" {
			err.Data = json.RawMessage(data)
		}
		// Wrap in a generic error to exercise the errors.As() path.
		wrapped := errors.New("wrap: " + err.Error())
		_ = IsNotFound(wrapped) // exercises the not-RPCError fast path
		_ = IsNotFound(err)     // exercises the typed path
		_ = IsNotFound(nil)     // sanity: must not panic on nil
	})
}

// FuzzIsAuthRateLimited mirrors the same structure for the auth
// rate-limit classifier. Same hot-path concern: auth retries depend
// on this not panicking.
func FuzzIsAuthRateLimited(f *testing.F) {
	seeds := []struct {
		code    int
		message string
		data    string
	}{
		{-32001, "Method call error", `{"errname":"EBUSY","reason":"Rate Limit Exceeded"}`},
		{-32001, "rate limit", ""},
		{-32601, "Method not found", ""},
		{0, "", ""},
	}
	for _, s := range seeds {
		f.Add(s.code, s.message, s.data)
	}
	f.Fuzz(func(t *testing.T, code int, message string, data string) {
		err := &RPCError{Code: code, Message: message}
		if data != "" {
			err.Data = json.RawMessage(data)
		}
		_ = isAuthRateLimited(err)
		_ = isAuthRateLimited(nil)
		_ = isAuthRateLimited(errors.New("plain"))
	})
}
