package customtypes_test

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	ct "github.com/PjSalty/terraform-provider-truenas/internal/customtypes"
)

func TestNormalizedYAMLType_Equal(t *testing.T) {
	typ := ct.NormalizedYAMLType{}
	if !typ.Equal(ct.NormalizedYAMLType{}) {
		t.Error("type must equal itself")
	}
	if typ.Equal(basetypes.StringType{}) {
		t.Error("type must not equal plain StringType")
	}
}

func TestNormalizedYAMLType_String(t *testing.T) {
	if got := (ct.NormalizedYAMLType{}).String(); got != "customtypes.NormalizedYAMLType" {
		t.Errorf("String(): %q", got)
	}
}

func TestNormalizedYAMLType_ValueFromString(t *testing.T) {
	v, diags := ct.NormalizedYAMLType{}.ValueFromString(
		context.Background(), basetypes.NewStringValue("a: 1\n"))
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	ny, ok := v.(ct.NormalizedYAML)
	if !ok || ny.ValueString() != "a: 1\n" {
		t.Errorf("got %#v", v)
	}
}

func TestNormalizedYAMLType_ValueFromTerraform(t *testing.T) {
	typ := ct.NormalizedYAMLType{}
	v, err := typ.ValueFromTerraform(context.Background(),
		tftypes.NewValue(tftypes.String, "a: 1\n"))
	if err != nil {
		t.Fatalf("ValueFromTerraform: %v", err)
	}
	ny, ok := v.(ct.NormalizedYAML)
	if !ok || ny.ValueString() != "a: 1\n" {
		t.Errorf("got %#v", v)
	}
	if _, err := typ.ValueFromTerraform(context.Background(),
		tftypes.NewValue(tftypes.Bool, true)); err == nil {
		t.Error("expected error for non-string tftypes value")
	}
}

func TestNormalizedYAMLType_ValueType(t *testing.T) {
	v := ct.NormalizedYAMLType{}.ValueType(context.Background())
	if _, ok := v.(ct.NormalizedYAML); !ok {
		t.Errorf("ValueType: %T", v)
	}
}

func TestNormalizedYAML_EqualAndType(t *testing.T) {
	a := ct.NewNormalizedYAMLValue("a: 1\n")
	if !a.Equal(ct.NewNormalizedYAMLValue("a: 1\n")) {
		t.Error("equal values must compare equal")
	}
	// Equal is byte-level by framework contract, semantic equality is
	// StringSemanticEquals' job
	if a.Equal(ct.NewNormalizedYAMLValue("a:  1\n")) {
		t.Error("byte-different values must not be Equal")
	}
	if a.Equal(basetypes.NewStringValue("a: 1\n")) {
		t.Error("must not equal a plain StringValue")
	}
	if _, ok := a.Type(context.Background()).(ct.NormalizedYAMLType); !ok {
		t.Errorf("Type(): %T", a.Type(context.Background()))
	}
}

func TestNormalizedYAML_Constructors(t *testing.T) {
	if !ct.NewNormalizedYAMLNull().IsNull() {
		t.Error("NewNormalizedYAMLNull not null")
	}
	if !ct.NewNormalizedYAMLUnknown().IsUnknown() {
		t.Error("NewNormalizedYAMLUnknown not unknown")
	}
	if ct.NewNormalizedYAMLValue("x: 1").ValueString() != "x: 1" {
		t.Error("NewNormalizedYAMLValue lost its value")
	}
}

func TestNormalizedYAML_StringSemanticEquals(t *testing.T) {
	ctx := context.Background()
	base := ct.NewNormalizedYAMLValue("services:\n  app:\n    image: busybox:1.36\n    replicas: 2\n")

	cases := []struct {
		name    string
		other   basetypes.StringValuable
		want    bool
		wantErr bool
	}{
		{
			// key order, indentation, comments, quoting all differ
			name: "reformatted equal",
			other: ct.NewNormalizedYAMLValue(
				"# a comment\nservices:\n    app:\n        replicas: 2\n        image: \"busybox:1.36\"\n"),
			want: true,
		},
		{
			name:  "changed value is different",
			other: ct.NewNormalizedYAMLValue("services:\n  app:\n    image: busybox:1.37\n    replicas: 2\n"),
			want:  false,
		},
		{
			name:    "unparseable other errors",
			other:   ct.NewNormalizedYAMLValue("not: [valid"),
			want:    false,
			wantErr: true,
		},
		{
			name:    "wrong valuable type errors",
			other:   basetypes.NewStringValue("a: 1\n"),
			want:    false,
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := base.StringSemanticEquals(ctx, tc.other)
			if diags.HasError() != tc.wantErr {
				t.Fatalf("diags error = %v, want %v (%v)", diags.HasError(), tc.wantErr, diags)
			}
			if got != tc.want {
				t.Errorf("semantic equals = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestYAMLStringsSemanticallyEqual(t *testing.T) {
	cases := []struct {
		name    string
		a, b    string
		want    bool
		wantErr bool
	}{
		{"identical", "a: 1\n", "a: 1\n", true, false},
		{"reordered keys", "a: 1\nb: 2\n", "b: 2\na: 1\n", true, false},
		{"comments and spacing", "a: 1\n", "# c\na:   1\n", true, false},
		{"int vs float spelling", "p: 80\n", "p: 80.0\n", true, false},
		{"yaml11 yes vs bool true", "v: yes\n", "v: true\n", true, false},
		{"yaml11 on vs bool true", "v: On\n", "v: true\n", true, false},
		{"yaml11 no vs bool false", "v: no\n", "v: false\n", true, false},
		{"yaml11 off vs bool false", "v: OFF\n", "v: false\n", true, false},
		{"yaml11 yes vs bool false", "v: yes\n", "v: false\n", false, false},
		{"bool true vs yaml11 yes reversed", "v: true\n", "v: yes\n", true, false},
		{"plain string vs bool", "v: maybe\n", "v: true\n", false, false},
		{"string mismatch", "v: a\n", "v: b\n", false, false},
		{"string vs number", "v: a\n", "v: 1\n", false, false},
		{"number vs string", "v: 1\n", "v: a\n", false, false},
		{"changed nested value", "s:\n  x: 1\n", "s:\n  x: 2\n", false, false},
		{"missing key", "s:\n  x: 1\n", "s:\n  y: 1\n", false, false},
		{"extra key", "s:\n  x: 1\n", "s:\n  x: 1\n  y: 2\n", false, false},
		{"map vs scalar", "s:\n  x: 1\n", "s: 1\n", false, false},
		{"lists equal", "l: [1, 2, 3]\n", "l:\n  - 1\n  - 2\n  - 3\n", true, false},
		{"list length differs", "l: [1, 2]\n", "l: [1, 2, 3]\n", false, false},
		{"list element differs", "l: [1, 2]\n", "l: [1, 3]\n", false, false},
		{"list vs scalar", "l: [1]\n", "l: 1\n", false, false},
		{"null vs null", "v: null\n", "v: ~\n", true, false},
		{"null vs value", "v: null\n", "v: 1\n", false, false},
		{"bool vs bool equal", "v: true\n", "v: true\n", true, false},
		{"bool vs bool differ", "v: true\n", "v: false\n", false, false},
		{"bool vs number", "v: true\n", "v: 1\n", false, false},
		{"timestamps equal", "t: 2015-01-02T03:04:05Z\n", "t: 2015-01-02T03:04:05Z\n", true, false},
		{"timestamps differ", "t: 2015-01-02T03:04:05Z\n", "t: 2016-01-02T03:04:05Z\n", false, false},
		{"first unparseable", "not: [valid", "a: 1\n", false, true},
		{"second unparseable", "a: 1\n", "not: [valid", false, true},
		{"empty docs equal", "", "# just a comment\n", true, false},

		// reformat property table: every pair MUST compare equal
		{"block vs flow style", "s:\n  a: 1\n  b: [1, 2]\n", "s: {a: 1, b: [1, 2]}\n", true, false},
		{"quote styles on plain strings", "v: hello\n", "v: \"hello\"\n", true, false},
		{"single vs double quotes", "v: 'hello'\n", "v: \"hello\"\n", true, false},
		{"indentation changed", "s:\n  a: 1\n", "s:\n        a: 1\n", true, false},
		{"trailing newline differences", "a: 1", "a: 1\n\n", true, false},
		{
			"aliases expanded vs referenced",
			"base: &b\n  image: busybox\nsvc:\n  <<: *b\n  extra: 1\n",
			"base:\n  image: busybox\nsvc:\n  image: busybox\n  extra: 1\n",
			true, false,
		},

		// mutation table: every pair MUST compare unequal
		{"added service", "services:\n  a: {image: x}\n", "services:\n  a: {image: x}\n  b: {image: y}\n", false, false},
		{"removed env var", "e:\n  A: 1\n  B: 2\n", "e:\n  A: 1\n", false, false},
		{"command list order changed", "command: [sleep, infinity]\n", "command: [infinity, sleep]\n", false, false},
		{"nested list of maps equal", "p:\n  - {a: 1}\n  - {b: 2}\n", "p:\n  - a: 1\n  - b: 2\n", true, false},
		{"nested list of maps differ", "p:\n  - {a: 1}\n  - {b: 2}\n", "p:\n  - {a: 1}\n  - {b: 3}\n", false, false},

		// numeric string vs number: quoting is semantic
		{"quoted number vs number", "v: \"8080\"\n", "v: 8080\n", false, false},
		{"scientific-looking string stays string", "v: \"8E1\"\n", "v: \"8E1\"\n", true, false},
		{"scientific string vs its numeric reading", "v: \"8E1\"\n", "v: 80\n", false, false},

		// yaml 1.1 shim precision: PyYAML's exact resolver spellings
		// only (lower, Title, UPPER), mixed case is a plain string
		{"shim ignores phrases", "v: yes please\n", "v: true\n", false, false},
		{"on is not the string true", "v: on\n", "v: \"true\"\n", false, false},
		{"quoted yes is still shimmed against bool", "v: \"yes\"\n", "v: true\n", true, false},
		{"Title case Yes shimmed", "v: \"Yes\"\n", "v: true\n", true, false},
		{"UPPER case NO shimmed", "v: \"NO\"\n", "v: false\n", true, false},
		{"mixed case yEs not shimmed", "v: \"yEs\"\n", "v: true\n", false, false},
		{"mixed case oN not shimmed", "v: \"oN\"\n", "v: true\n", false, false},
		{"mixed case nO not shimmed", "v: \"nO\"\n", "v: false\n", false, false},
		{"mixed case oFf not shimmed", "v: \"oFf\"\n", "v: false\n", false, false},

		// mapping keys are literal and collision-free: int 1 and
		// float 1.0 are distinct keys (values would compare equal)
		{"int and float keys reflexive", "1: a\n1.0: b\n", "1: a\n1.0: b\n", true, false},
		{"int key vs float key differ", "x:\n  1: a\n", "x:\n  1.0: a\n", false, false},
		{"null key reflexive", "~: a\n", "~: a\n", true, false},
		{"null key vs its sprint form", "~: a\n", "\"<nil>\": a\n", false, false},

		// null vs absent, null vs empty containers (documented strict)
		{"explicit null vs absent key", "a: 1\nb: null\n", "a: 1\n", false, false},
		{"null vs empty map", "v: null\n", "v: {}\n", false, false},
		{"null vs empty list", "v: null\n", "v: []\n", false, false},
		{"null vs empty string", "v: null\n", "v: \"\"\n", false, false},

		// weird but legal compose content
		{
			"unicode values",
			"e:\n  GREETING: \"héllo wörld 🚀\"\n",
			"e: {GREETING: \"héllo wörld 🚀\"}\n",
			true, false,
		},
		{
			"literal block scalar env value",
			"e:\n  SCRIPT: |\n    line one\n    line two\n",
			"e:\n  SCRIPT: \"line one\\nline two\\n\"\n",
			true, false,
		},
		{
			"minimal services-only compose",
			"services:\n  app:\n    image: busybox:1.36\n",
			"services: {app: {image: \"busybox:1.36\"}}\n",
			true, false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ct.YAMLStringsSemanticallyEqual(tc.a, tc.b)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("equal = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestYAMLDocsEqual_CrossDecoderNumerics drives the exported pure
// comparator with decoder products the string layer cannot produce
// directly: json float64 against yaml int/int64, precision-loss
// boundaries, and mixed containers.
func TestYAMLDocsEqual_CrossDecoderNumerics(t *testing.T) {
	const big = int64(1) << 60 // outside float64 exact range
	cases := []struct {
		name string
		a, b interface{}
		want bool
	}{
		// the canonical false-drift bug: yaml int vs json float
		{"port int vs json float64", 8080, float64(8080), true},
		{"int64 vs json float64", int64(8080), float64(8080), true},
		{"int vs int64", 8080, int64(8080), true},
		{"float vs float", 1.5, 1.5, true},
		{"non-integral float vs int", float64(8080.5), 8080, false},
		{"lossy float never equals big int", float64(big), big, false},
		{"negative lossy float never equals big negative int", -float64(big), -big, false},
		{"2^53 boundary is genuinely exact", int64(1) << 53, float64(int64(1) << 53), true},
		{"just inside exact range", (int64(1) << 53) - 1, float64((int64(1) << 53) - 1), true},
		{"first non-representable int never equals its rounded float", (int64(1) << 53) + 1, float64(int64(1) << 53), false},
		{"NaN equals nothing", math.NaN(), 1, false},
		{"infinity equals nothing integral", math.Inf(1), int64(1), false},
		{"number vs numeric string", float64(8080), "8080", false},
		{"nested map with json numbers", // server-side shape
			map[string]interface{}{"services": map[string]interface{}{"app": map[string]interface{}{"replicas": 2}}},
			map[string]interface{}{"services": map[string]interface{}{"app": map[string]interface{}{"replicas": float64(2)}}},
			true},
		{"list with mixed decoder numbers", []interface{}{1, int64(2), 3.0}, []interface{}{float64(1), float64(2), float64(3)}, true},
		{"map vs list", map[string]interface{}{}, []interface{}{}, false},
		{"list vs map", []interface{}{}, map[string]interface{}{}, false},
		{"uint64 same-type exact match", uint64(1) << 63, uint64(1) << 63, true},
		{"uint64 vs int not shimmed", uint64(5), 5, false},
		// fuzz-found regression: non-string yaml keys must not panic;
		// keys compare literally through the type-tagged canonical
		// form, so an int key never equals its string spelling
		{"int-keyed map vs string-keyed map",
			map[interface{}]interface{}{0: "a"},
			map[string]interface{}{"0": "a"},
			false},
		{"int-keyed maps reflexive", map[interface{}]interface{}{0: nil}, map[interface{}]interface{}{0: nil}, true},
		{"int-keyed map value differs", map[interface{}]interface{}{0: "a"}, map[string]interface{}{"0": "b"}, false},
		// collision-freedom: int 1 and float 1.0 keys coexist and
		// self-compare deterministically (fmt.Sprint used to collapse
		// them into one random winner)
		{"int and float 1 keys coexist reflexively",
			map[interface{}]interface{}{1: "a", 1.0: "b"},
			map[interface{}]interface{}{1: "a", 1.0: "b"},
			true},
		{"int-1 key never matches float-1.0 key",
			map[interface{}]interface{}{1: "a"},
			map[interface{}]interface{}{1.0: "a"},
			false},
		{"nil key vs string sprint form",
			map[interface{}]interface{}{nil: "a"},
			map[string]interface{}{"<nil>": "a"},
			false},
		{"bool key vs string key",
			map[interface{}]interface{}{true: "a"},
			map[string]interface{}{"true": "a"},
			false},
		{"timestamp keys reflexive",
			map[interface{}]interface{}{time.Date(2015, 1, 2, 3, 4, 5, 0, time.UTC): "a"},
			map[interface{}]interface{}{time.Date(2015, 1, 2, 3, 4, 5, 0, time.UTC): "a"},
			true},
		// int and int64 keys share the "i:" tag, same integer domain
		// (yaml.v3 yields int64 only on 32-bit platforms)
		{"int64 key equals int key",
			map[interface{}]interface{}{int64(1): "a"},
			map[interface{}]interface{}{1: "a"},
			true},
		// mixed-key maps: string keys inside a map[interface{}] map
		// still compare against themselves through the "s:" tag
		{"mixed string and int keys reflexive",
			map[interface{}]interface{}{"a": 1, 2: "b"},
			map[interface{}]interface{}{"a": 1, 2: "b"},
			true},
		{"mixed keys string value differs",
			map[interface{}]interface{}{"a": 1, 2: "b"},
			map[interface{}]interface{}{"a": 9, 2: "b"},
			false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ct.YAMLDocsEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("YAMLDocsEqual(%#v, %#v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
			// equality must be symmetric
			if got := ct.YAMLDocsEqual(tc.b, tc.a); got != tc.want {
				t.Errorf("YAMLDocsEqual(%#v, %#v) (reversed) = %v, want %v", tc.b, tc.a, got, tc.want)
			}
		})
	}
}

// TestNormalizedYAML_ReformatDoesNotDiff pins the exact hook the
// framework consults during plan: when StringSemanticEquals returns
// true for prior-state vs planned config, the framework keeps the
// prior state value and the plan shows no diff. A config-side
// reformat (or the canonical server dump vs the user's original
// string) therefore never plans a change.
func TestNormalizedYAML_ReformatDoesNotDiff(t *testing.T) {
	ctx := context.Background()
	// state as Read leaves it after drift repair: canonical dump form
	state := ct.NewNormalizedYAMLValue("services:\n    app:\n        command:\n            - sleep\n            - infinity\n        image: busybox:1.36\n")
	// user's differently formatted config of the same structure
	config := ct.NewNormalizedYAMLValue("# my app\nservices:\n  app:\n    image: \"busybox:1.36\"\n    command: [sleep, infinity]\n")
	eq, diags := state.StringSemanticEquals(ctx, config)
	if diags.HasError() {
		t.Fatalf("diags: %v", diags)
	}
	if !eq {
		t.Fatal("reformatted config must be semantically equal to state, otherwise a reformat would plan a diff")
	}
}

// FuzzYAMLStringsSemanticallyEqual pins two properties under fuzzing:
// no input pair may panic, and any string that parses must equal
// itself (reflexivity).
func FuzzYAMLStringsSemanticallyEqual(f *testing.F) {
	f.Add("services:\n  app:\n    image: busybox:1.36\n", "services: {app: {image: busybox:1.36}}\n")
	f.Add("v: yes\n", "v: true\n")
	f.Add("p: 8080\n", "p: 8080.0\n")
	f.Add("", "# comment\n")
	f.Add("not: [valid", "a: 1\n")
	f.Add("x: .nan\n", "x: .nan\n")
	f.Add("a: &x [1]\nb: *x\n", "a: [1]\nb: [1]\n")
	f.Fuzz(func(t *testing.T, a, b string) {
		_, _ = ct.YAMLStringsSemanticallyEqual(a, b) // must not panic
		if eqA, err := ct.YAMLStringsSemanticallyEqual(a, a); err == nil && !eqA {
			// NaN-containing docs are the one legal reflexivity hole
			// (NaN != NaN numerically), skip those
			if !strings.Contains(a, ".nan") && !strings.Contains(a, ".NaN") && !strings.Contains(a, ".NAN") {
				t.Errorf("reflexivity violated for %q", a)
			}
		}
	})
}

// keep attr import anchored for the interface-compliance asserts in
// the package under test
var _ attr.Value = ct.NormalizedYAML{}
