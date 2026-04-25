package client

// Fuzz tests for unexported client helpers. All targets must be panic-free
// for any string input; the underlying functions are reached from untrusted
// HTTP header data and must degrade gracefully.

import (
	"net/http"
	"testing"
)

// FuzzParseRetryAfter feeds arbitrary strings as the Retry-After header and
// verifies that parseRetryAfter never panics. The function is allowed to
// return 0 for malformed input — the property being checked is "no panic",
// not correctness of the parsed value.
func FuzzParseRetryAfter(f *testing.F) {
	// Seed with a mix of valid, invalid, and pathological inputs.
	seeds := []string{
		"",
		"0",
		"1",
		"5",
		"60",
		"-1",
		"9999999999",
		"not-a-number",
		"Wed, 21 Oct 2015 07:28:00 GMT",
		"Thu, 01 Jan 1970 00:00:00 GMT",
		"  7  ",
		"\x00\x01\x02",
		"0.5",
		"1e9",
		"NaN",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		resp := &http.Response{Header: http.Header{}}
		resp.Header.Set("Retry-After", input)
		// Property: never panics, always returns a finite Duration.
		d := parseRetryAfter(resp)
		_ = d
	})
}
