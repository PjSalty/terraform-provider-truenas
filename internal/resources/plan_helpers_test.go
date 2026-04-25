package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// rawFromValues builds a tftypes.Value for the schema's object type with
// the given top-level attribute values. Any attribute not present in vals
// is set to null using the schema-declared type. Nested objects/blocks
// get null-per-sub-attribute initialization so the framework decoder
// accepts them. If a value in vals is a string/number/bool/null placeholder
// with a mismatched type, this function will coerce it where possible.
func rawFromValues(t *testing.T, ctx context.Context, sch resource.SchemaResponse, vals map[string]tftypes.Value) tftypes.Value {
	t.Helper()
	typ := sch.Schema.Type().TerraformType(ctx)
	objType, ok := typ.(tftypes.Object)
	if !ok {
		t.Fatalf("schema type is not object: %T", typ)
	}
	m := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		if v, present := vals[name]; present {
			// If the provided value's type differs from the schema's
			// expected type, try to substitute a null of the correct type.
			if !v.Type().Equal(at) {
				// Placeholder collection types like strMapNull() use
				// element-type String, but the schema may declare a map
				// of objects. Treat those as nulls of the correct type.
				m[name] = tftypes.NewValue(at, nil)
				continue
			}
			m[name] = v
			continue
		}
		// For object-typed attributes (timeouts block, nested objects),
		// build a null-valued object with all sub-attribute types set to null.
		if obj, ok := at.(tftypes.Object); ok && len(obj.AttributeTypes) > 0 {
			sub := make(map[string]tftypes.Value, len(obj.AttributeTypes))
			for sname, st := range obj.AttributeTypes {
				sub[sname] = tftypes.NewValue(st, nil)
			}
			m[name] = tftypes.NewValue(obj, sub)
			continue
		}
		m[name] = tftypes.NewValue(at, nil)
	}
	return tftypes.NewValue(objType, m)
}

// primedStateV2 returns a tfsdk.State with Raw built via rawFromValues so
// block-typed attributes (like the timeouts block) are properly initialized.
func primedStateV2(t *testing.T, ctx context.Context, sch resource.SchemaResponse) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, nil)}
}

// planFromValues builds a tfsdk.Plan from a map of attribute -> tftypes.Value.
// Unmentioned attributes are null; the timeouts block is null-initialized.
func planFromValues(t *testing.T, ctx context.Context, sch resource.SchemaResponse, vals map[string]tftypes.Value) tfsdk.Plan {
	t.Helper()
	return tfsdk.Plan{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, vals)}
}

// stateFromValues builds a tfsdk.State from a map of attribute -> tftypes.Value.
func stateFromValues(t *testing.T, ctx context.Context, sch resource.SchemaResponse, vals map[string]tftypes.Value) tfsdk.State {
	t.Helper()
	return tfsdk.State{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, vals)}
}

// str is a shortcut for tftypes.NewValue(tftypes.String, s).
func str(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

// num is a shortcut for tftypes.NewValue(tftypes.Number, n).
func num(n int64) tftypes.Value { return tftypes.NewValue(tftypes.Number, n) }

// flag is a shortcut for tftypes.NewValue(tftypes.Bool, b).
func flag(b bool) tftypes.Value { return tftypes.NewValue(tftypes.Bool, b) }

// strListNull returns a null list[string] value.
func strListNull() tftypes.Value {
	return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil)
}

// strMapNull returns a null map[string]string value.
func strMapNull() tftypes.Value {
	return tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, nil)
}

// schemaOf runs r.Schema and returns the SchemaResponse for test setup.
func schemaOf(t *testing.T, ctx context.Context, r resource.Resource) resource.SchemaResponse {
	t.Helper()
	sch := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &sch)
	if sch.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", sch.Diagnostics)
	}
	return sch
}

// allJSON is a convenience wrapper for http handlers that echo a single
// JSON response for every request regardless of path/method.
var _ = tftypes.NewValue // keep import stable
