package customtypes

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestStripSANPrefix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"DNS:example.com", "example.com"},
		{"dns:example.com", "example.com"},
		{"IP:10.0.0.1", "10.0.0.1"},
		{"IP Address:10.0.0.1", "10.0.0.1"},
		{"email:admin@example.com", "admin@example.com"},
		{"URI:https://example.com", "https://example.com"},
		{"RID:1.2.3.4", "1.2.3.4"},
		{"dirName:CN=x", "CN=x"},
		{"otherName:1.2.3;UTF8:x", "1.2.3;UTF8:x"},
		{"RegisteredID:1.2.3", "1.2.3"},

		// left alone
		{"example.com", "example.com"},
		{"", ""},
		{"OTHER:example.com", "OTHER:example.com"},
		// a prefix with nothing after it is not a prefix, it is the whole value
		{"DNS:", "DNS:"},
		// only the first kind is removed; a second one is part of the name
		{"DNS:DNS:example.com", "DNS:example.com"},
	}
	for _, c := range cases {
		if got := StripSANPrefix(c.in); got != c.want {
			t.Errorf("StripSANPrefix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if len(cases) < len(sanTypePrefixes) {
		t.Fatalf("checked %d cases for %d prefixes", len(cases), len(sanTypePrefixes))
	}
	// Every prefix in the table needs a case above, or one could be removed
	// without any test noticing.
	for _, p := range sanTypePrefixes {
		found := false
		for _, c := range cases {
			if strings.HasPrefix(strings.ToLower(c.in), strings.ToLower(p.prefix)) && c.in != c.want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no case exercises the %q prefix", p.prefix)
		}
	}
}

func TestSANEntry_StringSemanticEquals(t *testing.T) {
	cases := []struct {
		name       string
		prior, new string
		want       bool
	}{
		{"server added the kind", "example.com", "DNS:example.com", true},
		{"config already carried it", "DNS:example.com", "DNS:example.com", true},
		{"config carried it, server did not", "DNS:example.com", "example.com", true},
		{"kinds differ, so the entries differ", "DNS:example.com", "IP:example.com", false},
		{"different names", "example.com", "DNS:other.com", false},
		{"same kind, different case of the kind", "dns:example.com", "DNS:example.com", true},
		// OpenSSL prints an IP entry both ways depending on context, and a
		// registered ID likewise. Each pair is one kind, not two.
		{"the two spellings of an IP entry", "IP:10.0.0.1", "IP Address:10.0.0.1", true},
		{"the two spellings of a registered ID", "RID:1.2.3", "RegisteredID:1.2.3", true},
		{"an IP entry is not a DNS entry", "IP:10.0.0.1", "DNS:10.0.0.1", false},
		{"case of the name is significant", "Example.com", "DNS:example.com", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, diags := NewSANEntryValue(c.prior).StringSemanticEquals(
				context.Background(), NewSANEntryValue(c.new),
			)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if got != c.want {
				t.Errorf("%q vs %q = %v, want %v", c.prior, c.new, got, c.want)
			}
		})
	}
}

// A mismatched valuable is a provider bug, not a practitioner one, so it has to
// surface as an error rather than a silent false.
func TestSANEntry_StringSemanticEqualsWrongType(t *testing.T) {
	_, diags := NewSANEntryValue("example.com").StringSemanticEquals(
		context.Background(), basetypes.NewStringValue("example.com"),
	)
	if !diags.HasError() {
		t.Fatal("comparing against a plain StringValue produced no error")
	}
}

func TestSANEntryType_Roundtrip(t *testing.T) {
	ctx := context.Background()
	typ := SANEntryType{}

	if !typ.Equal(SANEntryType{}) {
		t.Error("SANEntryType is not equal to itself")
	}
	if typ.Equal(basetypes.StringType{}) {
		t.Error("SANEntryType compared equal to a plain StringType")
	}
	if typ.String() != "customtypes.SANEntryType" {
		t.Errorf("String() = %q", typ.String())
	}
	if _, ok := typ.ValueType(ctx).(SANEntry); !ok {
		t.Errorf("ValueType() = %T, want SANEntry", typ.ValueType(ctx))
	}

	for _, in := range []tftypes.Value{
		tftypes.NewValue(tftypes.String, "DNS:example.com"),
		tftypes.NewValue(tftypes.String, nil),
		tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	} {
		got, err := typ.ValueFromTerraform(ctx, in)
		if err != nil {
			t.Fatalf("ValueFromTerraform(%v): %v", in, err)
		}
		entry, ok := got.(SANEntry)
		if !ok {
			t.Fatalf("ValueFromTerraform(%v) = %T, want SANEntry", in, got)
		}
		var back tftypes.Value
		back, err = entry.ToTerraformValue(ctx)
		if err != nil {
			t.Fatalf("ToTerraformValue: %v", err)
		}
		if !back.Equal(in) {
			t.Errorf("roundtrip changed %v into %v", in, back)
		}
	}

	if NewSANEntryNull().Equal(attr.Value(NewSANEntryUnknown())) {
		t.Error("a null entry compared equal to an unknown one")
	}
	if NewSANEntryValue("x").Equal(basetypes.NewStringValue("x")) {
		t.Error("a SANEntry compared equal to a plain StringValue")
	}
}
