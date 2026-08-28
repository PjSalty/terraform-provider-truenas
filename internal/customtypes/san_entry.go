package customtypes

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	_ basetypes.StringTypable                    = SANEntryType{}
	_ basetypes.StringValuable                   = SANEntry{}
	_ basetypes.StringValuableWithSemanticEquals = SANEntry{}
)

// sanTypePrefixes are the general-name kinds a certificate's subject
// alternative name can carry. TrueNAS accepts a bare name on create and reads
// the parsed certificate back, where each entry is rendered with its kind, so
// "example.com" goes in and "DNS:example.com" comes out.
//
// Only these are collapsed. An unrecognized prefix compares verbatim, which
// shows a diff the practitioner can see rather than hiding one: a spurious
// diff is a nuisance, a hidden diff is a bug.
//
// The second column is the kind two entries are compared on, so the two
// spellings OpenSSL uses for an IP entry ("IP:" and "IP Address:") and for a
// registered ID ("RID:" and "RegisteredID:") are one kind each rather than two
// that never match.
var sanTypePrefixes = []struct{ prefix, kind string }{
	{"DNS:", "dns"},
	{"IP Address:", "ip"},
	{"IP:", "ip"},
	{"email:", "email"},
	{"URI:", "uri"},
	{"RegisteredID:", "rid"},
	{"RID:", "rid"},
	{"dirName:", "dirname"},
	{"otherName:", "othername"},
}

// StripSANPrefix removes a leading general-name kind from a SAN entry. It
// returns the entry unchanged when the prefix is not one it recognizes.
func StripSANPrefix(s string) string {
	_, name := splitSAN(s)
	return name
}

// splitSAN separates a SAN entry into its general-name kind and the name
// itself. The kind is returned lowercased and without its colon, and is empty
// for a bare entry or one carrying a kind this package does not know.
func splitSAN(s string) (kind, name string) {
	for _, p := range sanTypePrefixes {
		if len(s) > len(p.prefix) && strings.EqualFold(s[:len(p.prefix)], p.prefix) {
			return p.kind, s[len(p.prefix):]
		}
	}
	return "", s
}

// SANEntryType is the attr.Type for SANEntry values.
type SANEntryType struct {
	basetypes.StringType
}

func (t SANEntryType) Equal(o attr.Type) bool {
	other, ok := o.(SANEntryType)
	if !ok {
		return false
	}
	return t.StringType.Equal(other.StringType)
}

func (t SANEntryType) String() string {
	return "customtypes.SANEntryType"
}

// ValueFromString converts a StringValue to a SANEntry.
func (t SANEntryType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return SANEntry{StringValue: in}, nil
}

// ValueFromTerraform converts a tftypes.Value to a SANEntry.
func (t SANEntryType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	stringValue, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T from StringType.ValueFromTerraform", attrValue)
	}
	stringValuable, diags := t.ValueFromString(ctx, stringValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to SANEntry: %v", diags)
	}
	return stringValuable, nil
}

func (t SANEntryType) ValueType(_ context.Context) attr.Value {
	return SANEntry{}
}

// SANEntry is one subject alternative name, compared without its general-name
// kind so the server adding one is not drift.
type SANEntry struct {
	basetypes.StringValue
}

// NewSANEntryNull returns a null SANEntry.
func NewSANEntryNull() SANEntry {
	return SANEntry{StringValue: basetypes.NewStringNull()}
}

// NewSANEntryUnknown returns an unknown SANEntry.
func NewSANEntryUnknown() SANEntry {
	return SANEntry{StringValue: basetypes.NewStringUnknown()}
}

// NewSANEntryValue returns a known SANEntry holding s.
func NewSANEntryValue(s string) SANEntry {
	return SANEntry{StringValue: basetypes.NewStringValue(s)}
}

func (v SANEntry) Equal(o attr.Value) bool {
	other, ok := o.(SANEntry)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

func (v SANEntry) Type(_ context.Context) attr.Type {
	return SANEntryType{}
}

// StringSemanticEquals keeps the prior value when the two entries name the same
// thing, ignoring the general-name kind the server adds on read.
//
// Without this, a configuration of san = ["example.com"] fails the apply it
// asked for: TrueNAS returns "DNS:example.com" and Terraform reports "Provider
// produced inconsistent result after apply", which is a provider bug by
// Terraform's definition and was reported as one.
func (v SANEntry) StringSemanticEquals(_ context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	newValue, ok := newValuable.(SANEntry)
	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			fmt.Sprintf("Expected customtypes.SANEntry, got: %T. "+
				"Please report this to the provider developers.", newValuable),
		)
		return false, diags
	}
	oldKind, oldName := splitSAN(v.ValueString())
	newKind, newName := splitSAN(newValue.ValueString())
	if oldName != newName {
		return false, diags
	}
	// One side naming no kind is the config's bare form against the server's
	// rendered one. Two DIFFERENT kinds are two different names that happen to
	// share a string, so they stay unequal.
	return oldKind == "" || newKind == "" || oldKind == newKind, nil
}
