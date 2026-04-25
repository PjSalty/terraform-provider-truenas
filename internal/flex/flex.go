// Package flex provides conversion helpers between native Go types and the
// Terraform plugin framework types (types.String, types.Int64, types.Bool,
// types.List). The patterns encoded here appear dozens of times across the
// resource layer — the package exists so future resources and reviews can
// ask "why didn't you use flex?" rather than re-implementing each
// nil-check and ElementsAs call by hand.
//
// The layout mirrors hashicorp/terraform-provider-aws:internal/flex. Helpers
// are intentionally tiny — each one encodes one well-understood rule so
// callers can compose them without leaking nil checks throughout the
// Create/Read/Update paths.
package flex

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringPointerValue returns a types.String that is null when p is nil and
// otherwise wraps the pointee. This is the standard way to flatten an
// optional string field from the TrueNAS API into the plan/state.
func StringPointerValue(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

// StringFromPointer is the inverse of StringPointerValue: it returns a
// *string from a framework types.String, producing nil for null/unknown
// values. Callers use this when marshaling the plan into an API request
// body where a missing field must be encoded as JSON null/absent rather
// than the empty string.
func StringFromPointer(s types.String) *string {
	if s.IsNull() || s.IsUnknown() {
		return nil
	}
	v := s.ValueString()
	return &v
}

// Int64PointerValue returns a types.Int64 that is null when p is nil and
// otherwise wraps the pointee. The TrueNAS API returns many numeric fields
// as *int in the typed client, so this pattern repeats for every resource
// with an optional integer attribute.
func Int64PointerValue(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}

// Int64FromPointer converts a framework types.Int64 into a *int suitable
// for the typed client. Null/unknown values produce nil — that is the
// correct "leave this field alone" signal in every TrueNAS PATCH body we
// emit today.
func Int64FromPointer(i types.Int64) *int {
	if i.IsNull() || i.IsUnknown() {
		return nil
	}
	v := int(i.ValueInt64())
	return &v
}

// Int64FromInt is a tiny convenience wrapper that casts a native int to
// the framework's Int64Value. It exists so resource code can write
// `flex.Int64FromInt(user.UID)` instead of the noisier
// `types.Int64Value(int64(user.UID))` — the latter is the single most
// copied idiom across the Read paths.
func Int64FromInt(i int) types.Int64 {
	return types.Int64Value(int64(i))
}

// BoolPointerValue returns a types.Bool that is null when p is nil and
// otherwise wraps the pointee. Used for tri-state API fields (true,
// false, unset) that our resource models surface as Optional+Computed.
func BoolPointerValue(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}

// BoolFromPointer converts a framework types.Bool into a *bool, returning
// nil for null/unknown. Same rationale as Int64FromPointer: nil means
// "leave this field alone" when marshaling an API request.
func BoolFromPointer(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// StringsFromListValue expands a framework types.List of strings into a
// native []string. It wraps the typical ElementsAs pattern so callers get
// a properly diag-annotated error path in one line. A null or unknown
// list returns nil without any diagnostics — callers that want strictness
// must check IsNull/IsUnknown themselves before calling.
func StringsFromListValue(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	out := make([]string, 0, len(l.Elements()))
	diags := l.ElementsAs(ctx, &out, false)
	return out, diags
}

// StringListValue is the inverse of StringsFromListValue: given a native
// []string it constructs a framework types.List. A nil slice produces a
// null list which preserves the "unset" semantics across a round-trip.
func StringListValue(ctx context.Context, s []string) (types.List, diag.Diagnostics) {
	if s == nil {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, s)
}

// Int64sFromListValue expands a framework types.List of int64 values into
// a native []int64 using the ElementsAs pattern. Same nil semantics as
// StringsFromListValue.
func Int64sFromListValue(ctx context.Context, l types.List) ([]int64, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	out := make([]int64, 0, len(l.Elements()))
	diags := l.ElementsAs(ctx, &out, false)
	return out, diags
}

// Int64ListValue constructs a types.List of int64 values from a native
// slice. A nil slice produces a null list.
func Int64ListValue(ctx context.Context, s []int64) (types.List, diag.Diagnostics) {
	if s == nil {
		return types.ListNull(types.Int64Type), nil
	}
	return types.ListValueFrom(ctx, types.Int64Type, s)
}
