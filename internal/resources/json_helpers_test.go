package resources

import (
	"strings"
	"testing"
)

// TestNormalizeJSON verifies normalizeJSON's behavior for empty input,
// key ordering, number preservation (via json.Number), and array values.
func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty input yields empty object",
			input: "",
			want:  "{}",
		},
		{
			name:  "object reorders keys alphabetically",
			input: `{"zebra":1,"apple":2,"mango":3}`,
			want:  `{"apple":2,"mango":3,"zebra":1}`,
		},
		{
			name:  "numbers preserved via json.Number",
			input: `{"big":1234567890123456789,"pi":3.14159}`,
			want:  `{"big":1234567890123456789,"pi":3.14159}`,
		},
		{
			name:  "array preserved in order",
			input: `{"items":[3,1,2]}`,
			want:  `{"items":[3,1,2]}`,
		},
		{
			name:  "nested object keys sorted",
			input: `{"outer":{"z":1,"a":2}}`,
			want:  `{"outer":{"a":2,"z":1}}`,
		},
		{
			name:    "malformed JSON returns error",
			input:   `{not valid`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeJSON(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeJSON(%q) = %s, want error", tt.input, string(got))
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeJSON(%q) unexpected error: %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Errorf("normalizeJSON(%q) = %s, want %s", tt.input, string(got), tt.want)
			}
		})
	}
}

// TestStripJSONNulls verifies nulls are removed at the top level and in
// nested objects, preserved inside arrays, and non-null values are kept.
func TestStripJSONNulls(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty input is passthrough",
			input: "",
			want:  "",
		},
		{
			name:  "top-level null is removed",
			input: `{"keep":1,"drop":null}`,
			want:  `{"keep":1}`,
		},
		{
			name:  "nested null is removed",
			input: `{"outer":{"keep":"yes","drop":null}}`,
			want:  `{"outer":{"keep":"yes"}}`,
		},
		{
			name:  "null inside array is preserved",
			input: `{"list":[1,null,3]}`,
			want:  `{"list":[1,null,3]}`,
		},
		{
			name:  "non-null values preserved",
			input: `{"a":1,"b":"two","c":true,"d":0}`,
			// Order: stripNullsFromValue builds a map which gets sorted by json.Marshal.
			want: `{"a":1,"b":"two","c":true,"d":0}`,
		},
		{
			name:    "malformed JSON returns error",
			input:   `{bad`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stripJSONNulls(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("stripJSONNulls(%q) = %q, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("stripJSONNulls(%q) unexpected error: %v", tt.input, err)
			}
			// Use string comparison with an allowance for map-order when
			// multiple keys exist: compare after normalization via
			// normalizeJSON which sorts keys.
			gotNorm, nerr := normalizeJSON(got)
			if nerr != nil && got != "" {
				t.Fatalf("could not normalize got %q: %v", got, nerr)
			}
			wantNorm, nerr := normalizeJSON(tt.want)
			if nerr != nil && tt.want != "" {
				t.Fatalf("could not normalize want %q: %v", tt.want, nerr)
			}
			if got == "" && tt.want == "" {
				return
			}
			if string(gotNorm) != string(wantNorm) {
				t.Errorf("stripJSONNulls(%q) = %s, want %s", tt.input, string(gotNorm), string(wantNorm))
			}
		})
	}
}

// TestNormalizeJSON_Extra adds additional edge cases beyond the base TestNormalizeJSON.
func TestNormalizeJSON_Extra(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "bool false preserved", input: `{"b":false}`, want: `{"b":false}`},
		{name: "bool true preserved", input: `{"b":true}`, want: `{"b":true}`},
		{name: "null preserved as-is", input: `{"k":null}`, want: `{"k":null}`},
		{name: "empty object", input: `{}`, want: `{}`},
		{name: "deeply nested sorted", input: `{"a":{"c":1,"b":{"y":1,"x":2}}}`, want: `{"a":{"b":{"x":2,"y":1},"c":1}}`},
		{name: "array of objects", input: `{"arr":[{"z":1},{"a":2}]}`, want: `{"arr":[{"z":1},{"a":2}]}`},
		{name: "mixed types", input: `{"s":"x","n":1,"b":true,"l":[1,2]}`, want: `{"b":true,"l":[1,2],"n":1,"s":"x"}`},
		{name: "negative numbers", input: `{"n":-42}`, want: `{"n":-42}`},
		{name: "float zero", input: `{"f":0.0}`, want: `{"f":0.0}`},
		{name: "string with spaces", input: `{"greeting":"hello world"}`, want: `{"greeting":"hello world"}`},
		{name: "unicode string", input: `{"emoji":"caf\u00e9"}`, want: `{"emoji":"café"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeJSON(tt.input)
			if err != nil {
				t.Fatalf("normalizeJSON(%q) error: %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Errorf("normalizeJSON(%q) = %s, want %s", tt.input, string(got), tt.want)
			}
		})
	}
}

// TestStripJSONNulls_Extra exercises additional null-stripping scenarios.
func TestStripJSONNulls_Extra(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "all nulls become empty object", input: `{"a":null,"b":null}`, want: `{}`},
		{name: "deeply nested null removed", input: `{"outer":{"inner":{"drop":null,"keep":1}}}`, want: `{"outer":{"inner":{"keep":1}}}`},
		{name: "null inside object inside array removed", input: `{"list":[{"a":null,"b":1}]}`, want: `{"list":[{"b":1}]}`},
		{name: "null preserved inside flat array", input: `{"list":[1,null,3]}`, want: `{"list":[1,null,3]}`},
		{name: "bool false not stripped", input: `{"enabled":false,"disabled":null}`, want: `{"enabled":false}`},
		{name: "zero not stripped", input: `{"n":0,"null_n":null}`, want: `{"n":0}`},
		{name: "empty string not stripped", input: `{"s":"","null_s":null}`, want: `{"s":""}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stripJSONNulls(tt.input)
			if err != nil {
				t.Fatalf("stripJSONNulls(%q) error: %v", tt.input, err)
			}
			gotNorm, _ := normalizeJSON(got)
			wantNorm, _ := normalizeJSON(tt.want)
			if string(gotNorm) != string(wantNorm) {
				t.Errorf("stripJSONNulls(%q) = %s, want %s", tt.input, string(gotNorm), string(wantNorm))
			}
		})
	}
}

// TestFilterJSONByKeys verifies that filterJSONByKeys keeps only the keys
// that appear in the reference JSON, and that edge cases around empty
// inputs and non-object references are handled correctly.
func TestFilterJSONByKeys(t *testing.T) {
	tests := []struct {
		name    string
		server  string
		ref     string
		want    string
		wantErr bool
	}{
		{
			name:   "server superset filtered to reference keys",
			server: `{"a":1,"b":2,"c":3,"d":4}`,
			ref:    `{"a":"x","c":"y"}`,
			want:   `{"a":1,"c":3}`,
		},
		{
			name:   "empty reference yields full server canonicalized",
			server: `{"z":1,"a":2}`,
			ref:    "",
			want:   `{"a":2,"z":1}`,
		},
		{
			name:   "non-JSON-object reference yields full server",
			server: `{"z":1,"a":2}`,
			ref:    `[1,2,3]`,
			want:   `{"a":2,"z":1}`,
		},
		{
			name:   "both empty yields empty object",
			server: "",
			ref:    "",
			want:   "{}",
		},
		{
			name:   "reference key missing from server is skipped",
			server: `{"a":1}`,
			ref:    `{"a":1,"missing":2}`,
			want:   `{"a":1}`,
		},
		{
			name:    "invalid server JSON returns error",
			server:  `{bad`,
			ref:     `{"a":1}`,
			wantErr: true,
		},
		{
			name:   "nested objects preserved under known keys",
			server: `{"creds":{"id":1,"name":"x"},"drop":2}`,
			ref:    `{"creds":null}`,
			want:   `{"creds":{"id":1,"name":"x"}}`,
		},
		{
			name:   "array value at known key kept intact",
			server: `{"list":[1,2,3],"extra":true}`,
			ref:    `{"list":[]}`,
			want:   `{"list":[1,2,3]}`,
		},
		{
			name:   "boolean and numeric values preserved",
			server: `{"enabled":true,"count":42,"extra":"x"}`,
			ref:    `{"enabled":false,"count":0}`,
			want:   `{"enabled":true,"count":42}`,
		},
		{
			name:   "zero-length ref object yields empty filtered",
			server: `{"a":1,"b":2}`,
			ref:    `{}`,
			want:   `{}`,
		},
		{
			name:   "ref with all missing keys yields empty",
			server: `{"real":1}`,
			ref:    `{"x":1,"y":2}`,
			want:   `{}`,
		},
		{
			name:   "unicode key preserved",
			server: `{"café":1,"drop":2}`,
			ref:    `{"café":""}`,
			want:   `{"café":1}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := filterJSONByKeys(tt.server, tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("filterJSONByKeys(%q,%q) = %q, want error", tt.server, tt.ref, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("filterJSONByKeys(%q,%q) unexpected error: %v", tt.server, tt.ref, err)
			}
			// Normalize both so key-order differences don't cause false failures.
			gotNorm, _ := normalizeJSON(got)
			wantNorm, _ := normalizeJSON(tt.want)
			if !strings.EqualFold(string(gotNorm), string(wantNorm)) {
				t.Errorf("filterJSONByKeys(%q,%q) = %s, want %s",
					tt.server, tt.ref, string(gotNorm), string(wantNorm))
			}
		})
	}
}
