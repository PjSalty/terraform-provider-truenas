package resources

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// The whole point of exposing `exclude` is that an unset attribute must NOT
// clear whatever the box already holds. A nil return means "send nothing".
func TestExcludeFromList_nullAndUnknownSendNothing(t *testing.T) {
	if got := excludeFromList(types.ListNull(types.StringType)); got != nil {
		t.Fatalf("null list must send nothing, got %#v", got)
	}
	if got := excludeFromList(types.ListUnknown(types.StringType)); got != nil {
		t.Fatalf("unknown list must send nothing, got %#v", got)
	}
}

func TestExcludeRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
	}{
		{"empty", []string{}},
		{"one", []string{"tank/users/matthew"}},
		{"many", []string{"tank/users/matthew", "tank/users/renea"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ptr := excludeFromList(excludeToList(tc.in))
			if ptr == nil {
				t.Fatal("a known list must be sent, got nil")
			}
			out := *ptr
			if len(out) != len(tc.in) {
				t.Fatalf("length changed: %d -> %d", len(tc.in), len(out))
			}
			for i := range tc.in {
				if out[i] != tc.in[i] {
					t.Fatalf("element %d: %q != %q", i, out[i], tc.in[i])
				}
			}
		})
	}
}

// A nil slice from the API must become an EMPTY list, never null: the attribute
// is Computed, so a null here leaves it unknown after apply.
func TestExcludeToList_nilBecomesEmptyNotNull(t *testing.T) {
	l := excludeToList(nil)
	if l.IsNull() || l.IsUnknown() {
		t.Fatalf("nil slice must map to a known empty list, got null=%v unknown=%v", l.IsNull(), l.IsUnknown())
	}
	if n := len(l.Elements()); n != 0 {
		t.Fatalf("expected 0 elements, got %d", n)
	}
}

// Guards the regression this attribute exists to prevent: a populated exclude
// on the box must survive a read into state, so `terraform plan` can show it.
func TestExcludeSurvivesReadIntoModel(t *testing.T) {
	var elems []attr.Value
	for _, s := range []string{"tank/users/timm"} {
		elems = append(elems, types.StringValue(s))
	}
	want := types.ListValueMust(types.StringType, elems)
	got := excludeToList([]string{"tank/users/timm"})
	if !got.Equal(want) {
		t.Fatalf("read mapping lost the exclude list: %v != %v", got, want)
	}
}

// Asserts the wire, not the Go slice. The request structs tag exclude with
// omitempty, and encoding/json omits an EMPTY slice under omitempty as well as
// a nil one, so the two cases the resource must tell apart only stay apart if
// the field is a pointer: nil omits the key, a pointer to [] sends "exclude":[].
func TestExcludeWire_emptyListIsSentNullIsOmitted(t *testing.T) {
	for _, tc := range []struct {
		name    string
		in      types.List
		want    string
		present bool
	}{
		{"null keeps the box's value", types.ListNull(types.StringType), "", false},
		{"unknown keeps the box's value", types.ListUnknown(types.StringType), "", false},
		{"explicit empty clears it", excludeToList([]string{}), `"exclude":[]`, true},
		{"populated is sent", excludeToList([]string{"tank/a"}), `"exclude":["tank/a"]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, body := range []any{
				truenas.SnapshotTaskCreateRequest{Exclude: excludeFromList(tc.in)},
				truenas.SnapshotTaskUpdateRequest{Exclude: excludeFromList(tc.in)},
			} {
				b, err := json.Marshal(body)
				if err != nil {
					t.Fatalf("marshal %T: %v", body, err)
				}
				got := string(b)
				if has := strings.Contains(got, `"exclude"`); has != tc.present {
					t.Errorf("%T %s: exclude key present=%v, want %v", body, got, has, tc.present)
				}
				if tc.present && !strings.Contains(got, tc.want) {
					t.Errorf("%T %s: want %s", body, got, tc.want)
				}
			}
		})
	}
}
