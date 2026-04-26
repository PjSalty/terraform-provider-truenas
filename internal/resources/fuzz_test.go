package resources

// Fuzz tests for JSON helpers used in round-tripping TrueNAS API payloads.
// Property under test: each helper must never panic for any input string.
// Returning an error is fine — the helpers are fed user-supplied and
// server-supplied JSON, both of which may be malformed.

import (
	"testing"
)

// FuzzNormalizeJSON verifies normalizeJSON never panics for arbitrary input.
// Invalid JSON is expected to return an error, not crash.
func FuzzNormalizeJSON(f *testing.F) {
	seeds := []string{
		"",
		"{}",
		"[]",
		"null",
		"true",
		`{"a":1}`,
		`{"nested":{"deep":{"x":1}}}`,
		`[1,2,3]`,
		`{"big":1234567890123456789,"pi":3.14159}`,
		`{not valid`,
		`{"a":}`,
		"\x00\x01\x02",
		`{"unicode":"\u00e9"}`,
		`{"dup":1,"dup":2}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_, _ = normalizeJSON(input)
	})
}

// FuzzStripJSONNulls verifies stripJSONNulls never panics for arbitrary
// input. Recursive traversal over decoded maps/slices is the risk area.
func FuzzStripJSONNulls(f *testing.F) {
	seeds := []string{
		"",
		"{}",
		"[]",
		`{"a":null}`,
		`{"a":null,"b":1}`,
		`{"a":{"b":null,"c":2}}`,
		`[null,null,null]`,
		`[{"a":null},{"a":1}]`,
		`{"deeply":{"nested":{"null":null}}}`,
		`invalid`,
		"\x00",
		`{"mixed":[null,1,"str",true,{"x":null}]}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_, _ = stripJSONNulls(input)
	})
}

// FuzzFilterJSONByKeys verifies filterJSONByKeys never panics for any
// combination of server/reference inputs. Two fuzzer parameters cover the
// Cartesian product.
func FuzzFilterJSONByKeys(f *testing.F) {
	seeds := []struct {
		server, reference string
	}{
		{"", ""},
		{"{}", "{}"},
		{`{"a":1,"b":2}`, `{"a":null}`},
		{`{"a":1,"b":2}`, `{"a":null,"b":null,"c":null}`},
		{`{"a":1}`, ""},
		{"", `{"a":1}`},
		{`[1,2,3]`, `{"a":1}`},
		{`{"a":1}`, `[1,2,3]`},
		{`invalid`, `{"a":1}`},
		{`{"a":1}`, `invalid`},
		{`{"nested":{"deep":1}}`, `{"nested":null}`},
	}
	for _, s := range seeds {
		f.Add(s.server, s.reference)
	}

	f.Fuzz(func(t *testing.T, server, reference string) {
		_, _ = filterJSONByKeys(server, reference)
	})
}
