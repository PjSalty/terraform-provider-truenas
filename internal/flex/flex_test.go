package flex_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/flex"
)

func TestStringPointerValue(t *testing.T) {
	t.Parallel()
	if got := flex.StringPointerValue(nil); !got.IsNull() {
		t.Fatalf("nil pointer: want null, got %#v", got)
	}
	s := "hello"
	if got := flex.StringPointerValue(&s); got.ValueString() != "hello" {
		t.Fatalf("value pointer: want hello, got %q", got.ValueString())
	}
	empty := ""
	if got := flex.StringPointerValue(&empty); got.IsNull() || got.ValueString() != "" {
		t.Fatalf("empty pointer must be non-null empty string, got %#v", got)
	}
}

func TestStringFromPointer(t *testing.T) {
	t.Parallel()
	if got := flex.StringFromPointer(types.StringNull()); got != nil {
		t.Fatalf("null: want nil, got %q", *got)
	}
	if got := flex.StringFromPointer(types.StringUnknown()); got != nil {
		t.Fatalf("unknown: want nil, got %q", *got)
	}
	got := flex.StringFromPointer(types.StringValue("abc"))
	if got == nil || *got != "abc" {
		t.Fatalf("value: want abc, got %v", got)
	}
}

func TestInt64PointerValue(t *testing.T) {
	t.Parallel()
	if got := flex.Int64PointerValue(nil); !got.IsNull() {
		t.Fatalf("nil: want null, got %#v", got)
	}
	n := 42
	if got := flex.Int64PointerValue(&n); got.ValueInt64() != 42 {
		t.Fatalf("value: want 42, got %d", got.ValueInt64())
	}
	zero := 0
	if got := flex.Int64PointerValue(&zero); got.IsNull() || got.ValueInt64() != 0 {
		t.Fatalf("zero pointer must be non-null 0, got %#v", got)
	}
}

func TestInt64FromPointer(t *testing.T) {
	t.Parallel()
	if got := flex.Int64FromPointer(types.Int64Null()); got != nil {
		t.Fatalf("null: want nil, got %d", *got)
	}
	if got := flex.Int64FromPointer(types.Int64Unknown()); got != nil {
		t.Fatalf("unknown: want nil, got %d", *got)
	}
	got := flex.Int64FromPointer(types.Int64Value(7))
	if got == nil || *got != 7 {
		t.Fatalf("value: want 7, got %v", got)
	}
}

func TestInt64FromInt(t *testing.T) {
	t.Parallel()
	got := flex.Int64FromInt(123)
	if got.ValueInt64() != 123 {
		t.Fatalf("want 123, got %d", got.ValueInt64())
	}
}

func TestBoolPointerValue(t *testing.T) {
	t.Parallel()
	if got := flex.BoolPointerValue(nil); !got.IsNull() {
		t.Fatalf("nil: want null, got %#v", got)
	}
	tr := true
	if got := flex.BoolPointerValue(&tr); !got.ValueBool() {
		t.Fatalf("true: want true, got false")
	}
	f := false
	if got := flex.BoolPointerValue(&f); got.IsNull() || got.ValueBool() {
		t.Fatalf("false pointer must be non-null false, got %#v", got)
	}
}

func TestBoolFromPointer(t *testing.T) {
	t.Parallel()
	if got := flex.BoolFromPointer(types.BoolNull()); got != nil {
		t.Fatalf("null: want nil, got %v", *got)
	}
	if got := flex.BoolFromPointer(types.BoolUnknown()); got != nil {
		t.Fatalf("unknown: want nil, got %v", *got)
	}
	got := flex.BoolFromPointer(types.BoolValue(true))
	if got == nil || !*got {
		t.Fatalf("value: want true, got %v", got)
	}
}

func TestStringsFromListValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	out, diags := flex.StringsFromListValue(ctx, types.ListNull(types.StringType))
	if diags.HasError() || out != nil {
		t.Fatalf("null: want (nil, no-diags), got (%v, %v)", out, diags)
	}

	out, diags = flex.StringsFromListValue(ctx, types.ListUnknown(types.StringType))
	if diags.HasError() || out != nil {
		t.Fatalf("unknown: want (nil, no-diags), got (%v, %v)", out, diags)
	}

	list, listDiags := types.ListValueFrom(ctx, types.StringType, []string{"a", "b", "c"})
	if listDiags.HasError() {
		t.Fatalf("build list: %v", listDiags)
	}
	out, diags = flex.StringsFromListValue(ctx, list)
	if diags.HasError() || len(out) != 3 || out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Fatalf("value: want [a b c], got %v (%v)", out, diags)
	}

	// Wrong element type must surface diagnostics via ElementsAs.
	wrong, wrongDiags := types.ListValue(types.Int64Type, []attr.Value{types.Int64Value(1)})
	if wrongDiags.HasError() {
		t.Fatalf("build wrong list: %v", wrongDiags)
	}
	_, diags = flex.StringsFromListValue(ctx, wrong)
	if !diags.HasError() {
		t.Fatalf("wrong element type: expected diag error, got none")
	}
}

func TestStringListValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	got, diags := flex.StringListValue(ctx, nil)
	if diags.HasError() || !got.IsNull() {
		t.Fatalf("nil slice: want null list, got (%v, %v)", got, diags)
	}

	got, diags = flex.StringListValue(ctx, []string{"x", "y"})
	if diags.HasError() {
		t.Fatalf("value: unexpected diags: %v", diags)
	}
	elems := got.Elements()
	if len(elems) != 2 {
		t.Fatalf("value: want 2 elems, got %d", len(elems))
	}
	if elems[0].(types.String).ValueString() != "x" || elems[1].(types.String).ValueString() != "y" {
		t.Fatalf("value: want [x y], got %v", elems)
	}

	got, diags = flex.StringListValue(ctx, []string{})
	if diags.HasError() || got.IsNull() || len(got.Elements()) != 0 {
		t.Fatalf("empty slice must produce non-null empty list, got (%v, %v)", got, diags)
	}
}

func TestInt64sFromListValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	out, diags := flex.Int64sFromListValue(ctx, types.ListNull(types.Int64Type))
	if diags.HasError() || out != nil {
		t.Fatalf("null: want (nil, no-diags), got (%v, %v)", out, diags)
	}

	out, diags = flex.Int64sFromListValue(ctx, types.ListUnknown(types.Int64Type))
	if diags.HasError() || out != nil {
		t.Fatalf("unknown: want (nil, no-diags), got (%v, %v)", out, diags)
	}

	list, listDiags := types.ListValueFrom(ctx, types.Int64Type, []int64{1, 2, 3})
	if listDiags.HasError() {
		t.Fatalf("build list: %v", listDiags)
	}
	out, diags = flex.Int64sFromListValue(ctx, list)
	if diags.HasError() || len(out) != 3 || out[0] != 1 || out[2] != 3 {
		t.Fatalf("value: want [1 2 3], got %v (%v)", out, diags)
	}

	// Wrong element type must surface diagnostics via ElementsAs.
	wrong, wrongDiags := types.ListValue(types.StringType, []attr.Value{types.StringValue("nope")})
	if wrongDiags.HasError() {
		t.Fatalf("build wrong list: %v", wrongDiags)
	}
	_, diags = flex.Int64sFromListValue(ctx, wrong)
	if !diags.HasError() {
		t.Fatalf("wrong element type: expected diag error, got none")
	}
}

func TestInt64ListValue(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	got, diags := flex.Int64ListValue(ctx, nil)
	if diags.HasError() || !got.IsNull() {
		t.Fatalf("nil slice: want null list, got (%v, %v)", got, diags)
	}

	got, diags = flex.Int64ListValue(ctx, []int64{10, 20})
	if diags.HasError() {
		t.Fatalf("value: unexpected diags: %v", diags)
	}
	elems := got.Elements()
	if len(elems) != 2 {
		t.Fatalf("value: want 2 elems, got %d", len(elems))
	}
	if elems[0].(types.Int64).ValueInt64() != 10 || elems[1].(types.Int64).ValueInt64() != 20 {
		t.Fatalf("value: want [10 20], got %v", elems)
	}
}
