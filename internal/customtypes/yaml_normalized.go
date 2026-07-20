// Package customtypes holds framework custom types cloning the
// terraform-plugin-framework-jsontypes API surface for other formats.
//
// NormalizedYAML mirrors jsontypes.Normalized for YAML documents:
// semantic equality parses both strings and compares the resulting
// documents, so formatting, comments, key order, quoting, and YAML 1.1
// bool spellings (yes/no/on/off) never plan as diffs. Used by
// truenas_app.custom_compose, where the middleware stores the parsed
// compose dict (PyYAML, YAML 1.1) rather than the user's string.
package customtypes

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"gopkg.in/yaml.v3"
)

var (
	_ basetypes.StringTypable                    = NormalizedYAMLType{}
	_ basetypes.StringValuable                   = NormalizedYAML{}
	_ basetypes.StringValuableWithSemanticEquals = NormalizedYAML{}
)

// NormalizedYAMLType is the attr.Type for NormalizedYAML values.
type NormalizedYAMLType struct {
	basetypes.StringType
}

func (t NormalizedYAMLType) Equal(o attr.Type) bool {
	other, ok := o.(NormalizedYAMLType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t NormalizedYAMLType) String() string {
	return "customtypes.NormalizedYAMLType"
}

// ValueFromString converts a StringValue to a NormalizedYAML.
func (t NormalizedYAMLType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return NormalizedYAML{StringValue: in}, nil
}

// ValueFromTerraform converts a tftypes.Value to a NormalizedYAML.
func (t NormalizedYAMLType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	// StringType.ValueFromTerraform is contract-bound to return a
	// basetypes.StringValue on success, the comma-ok keeps the
	// impossible mismatch from panicking
	stringValue, _ := attrValue.(basetypes.StringValue)
	return NormalizedYAML{StringValue: stringValue}, nil
}

// ValueType returns the value type produced by this attr.Type.
func (t NormalizedYAMLType) ValueType(_ context.Context) attr.Value {
	return NormalizedYAML{}
}

// NormalizedYAML is a string value whose equality is semantic over the
// parsed YAML document rather than byte-for-byte.
type NormalizedYAML struct {
	basetypes.StringValue
}

// NewNormalizedYAMLNull returns a null NormalizedYAML.
func NewNormalizedYAMLNull() NormalizedYAML {
	return NormalizedYAML{StringValue: basetypes.NewStringNull()}
}

// NewNormalizedYAMLUnknown returns an unknown NormalizedYAML.
func NewNormalizedYAMLUnknown() NormalizedYAML {
	return NormalizedYAML{StringValue: basetypes.NewStringUnknown()}
}

// NewNormalizedYAMLValue returns a known NormalizedYAML holding s.
func NewNormalizedYAMLValue(s string) NormalizedYAML {
	return NormalizedYAML{StringValue: basetypes.NewStringValue(s)}
}

func (v NormalizedYAML) Equal(o attr.Value) bool {
	other, ok := o.(NormalizedYAML)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v NormalizedYAML) Type(_ context.Context) attr.Type {
	return NormalizedYAMLType{}
}

// StringSemanticEquals keeps the prior state value when the new value
// parses to the same YAML document. Formatting, comments, key order,
// quoting, and YAML 1.1 bool spellings are never drift.
func (v NormalizedYAML) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(NormalizedYAML)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("Expected customtypes.NormalizedYAML, got: %T. "+
				"Please report this to the provider developers.", newValuable),
		)
		return false, diags
	}
	eq, err := YAMLStringsSemanticallyEqual(v.ValueString(), newValue.ValueString())
	if err != nil {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("Could not parse YAML for semantic comparison: %s", err),
		)
		return false, diags
	}
	return eq, diags
}

// YAMLStringsSemanticallyEqual reports whether two YAML strings parse
// to structurally equal documents, see YAMLDocsEqual for the equality
// semantics.
func YAMLStringsSemanticallyEqual(a, b string) (bool, error) {
	var aDoc, bDoc interface{}
	if err := yaml.Unmarshal([]byte(a), &aDoc); err != nil {
		return false, fmt.Errorf("parsing first YAML document: %w", err)
	}
	if err := yaml.Unmarshal([]byte(b), &bDoc); err != nil {
		return false, fmt.Errorf("parsing second YAML document: %w", err)
	}
	return YAMLDocsEqual(aDoc, bDoc), nil
}

// YAMLDocsEqual compares two parsed YAML/JSON documents structurally.
// Pure function, no I/O, safe to fuzz.
//
// Cross-decoder normalization (the server dict arrives over JSON-RPC
// through encoding/json, the user string through yaml.v3):
//   - numbers compare numerically across int, int64, and float64; an
//     int-vs-float pair is equal only when the float is exactly
//     integral and inside the 2^53 exactly-representable range, so
//     precision loss can never fake equality
//   - a numeric STRING ("8080", "8E1") never equals a number, quoting
//     is a semantic choice in compose files
//   - YAML 1.1 bools: the middleware parses with PyYAML, so unquoted
//     yes/no/on/off became bools server-side while yaml.v3 (YAML 1.2)
//     keeps them strings; exactly those four spellings
//     (case-insensitive) compare equal to the matching bool, nothing
//     else ("true" the STRING stays a string)
//
// Deliberate strictness:
//   - lists compare ordered, compose lists (command, ports, volumes)
//     are order-sensitive
//   - null is not an empty map, not an empty slice, and not ""; a key
//     explicitly set to null differs from an absent key
//   - mapping KEYS compare literally via a collision-free type-tagged
//     canonical form (see canonicalKey): int 1, float 1.0, string
//     "1", bool true, and null are five different keys, and none of
//     the numeric or YAML 1.1 value shims apply to keys. Quote any
//     key YAML could type-convert (numbers, yes/no/on/off) to keep
//     it a string on both sides
func YAMLDocsEqual(a, b interface{}) bool {
	a = normalizeKeys(a)
	b = normalizeKeys(b)
	if eq, bothNums := numsEqual(a, b); bothNums {
		return eq
	}
	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bvv, present := bv[k]
			if !present || !YAMLDocsEqual(v, bvv) {
				return false
			}
		}
		return true
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !YAMLDocsEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case string:
		if bb, ok := b.(bool); ok {
			return yaml11BoolEqual(av, bb)
		}
		bs, ok := b.(string)
		return ok && av == bs
	case bool:
		if bs, ok := b.(string); ok {
			return yaml11BoolEqual(bs, av)
		}
		bb, ok := b.(bool)
		return ok && av == bb
	case nil:
		return b == nil
	default:
		// rare decoder products (time.Time, uint64 overflow values,
		// !!binary byte slices): DeepEqual never panics on
		// uncomparable types, unlike ==
		return reflect.DeepEqual(a, b)
	}
}

// normalizeKeys rewrites both map shapes onto the canonicalKey form
// so mixed-key yaml.v3 maps (map[interface{}]interface{}) and
// string-keyed maps (yaml.v3 or json) compare through one key space.
// Nested values normalize on recursion.
func normalizeKeys(v interface{}) interface{} {
	switch m := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			out["s:"+k] = val
		}
		return out
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(m))
		for k, val := range m {
			out[canonicalKey(k)] = val
		}
		return out
	}
	return v
}

// canonicalKey returns a collision-free deterministic string form for
// a YAML mapping key. The type tag keeps int 1, float 1.0, string
// "1", bool true, and null in distinct slots, so two parses of the
// same document always canonicalize identically and equality stays
// deterministic (fmt.Sprint sent int 1 and float 1.0 to the same
// slot, which made map iteration order decide the winner). Integral
// floats format via FormatFloat 'g' to their shortest form ("f:1"),
// still tagged apart from "i:1".
func canonicalKey(k interface{}) string {
	switch v := k.(type) {
	case string:
		return "s:" + v
	case int:
		return "i:" + strconv.FormatInt(int64(v), 10)
	case int64:
		return "i:" + strconv.FormatInt(v, 10)
	case float64:
		return "f:" + strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		return "b:" + strconv.FormatBool(v)
	case nil:
		return "n:"
	default:
		// exotic decoder key products (timestamps): deterministic for
		// equal values, tagged apart from every scalar family above
		return fmt.Sprintf("o:%v", v)
	}
}

// yaml11BoolEqual reports whether s is a YAML 1.1 spelling of b,
// matching PyYAML's bool resolver EXACTLY: the lower, Title, and
// UPPER spellings of yes/no/on/off and nothing else. Mixed-case forms
// like yEs or oN are plain strings to PyYAML too, so they are not
// shimmed, and neither are "true"/"false" as strings or phrases like
// "yes please".
func yaml11BoolEqual(s string, b bool) bool {
	switch s {
	case "yes", "Yes", "YES", "on", "On", "ON":
		return b
	case "no", "No", "NO", "off", "Off", "OFF":
		return !b
	}
	return false
}

// float64ExactIntMax is 2^53, the largest magnitude at which float64
// still represents every integer exactly.
const float64ExactIntMax = int64(1) << 53

// numsEqual compares two values numerically when both are numbers.
// bothNums reports whether the pair was numeric at all.
func numsEqual(a, b interface{}) (equal, bothNums bool) {
	ai, aIsInt := toInt64(a)
	bi, bIsInt := toInt64(b)
	af, aIsFloat := a.(float64)
	bf, bIsFloat := b.(float64)
	switch {
	case aIsInt && bIsInt:
		return ai == bi, true
	case aIsFloat && bIsFloat:
		return af == bf, true
	case aIsInt && bIsFloat:
		return floatEqualsInt(bf, ai), true
	case aIsFloat && bIsInt:
		return floatEqualsInt(af, bi), true
	}
	return false, false
}

// toInt64 widens the integer decoder products (yaml.v3 yields int, or
// int64 on 32-bit platforms). uint64 overflow values fall through to
// the exact-match default in YAMLDocsEqual.
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	}
	return 0, false
}

// floatEqualsInt reports whether f exactly equals i: f must be
// integral, finite, and inside the exactly-representable range, so a
// lossy float64 can never fake equality with a large integer.
func floatEqualsInt(f float64, i int64) bool {
	if math.IsNaN(f) || math.IsInf(f, 0) || f != math.Trunc(f) {
		return false
	}
	if f < -float64(float64ExactIntMax) || f > float64(float64ExactIntMax) {
		return false
	}
	return int64(f) == i
}
