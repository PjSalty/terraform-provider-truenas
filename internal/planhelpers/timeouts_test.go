package planhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// value builds a timeouts.Value carrying the given per-operation strings. A
// nil map means the practitioner wrote no block at all.
func value(t *testing.T, attrs map[string]string) timeouts.Value {
	t.Helper()
	typ := map[string]attr.Type{
		"create": types.StringType, "read": types.StringType,
		"update": types.StringType, "delete": types.StringType,
	}
	if attrs == nil {
		return timeouts.Value{Object: basetypes.NewObjectNull(typ)}
	}
	vals := map[string]attr.Value{}
	for k := range typ {
		if s, ok := attrs[k]; ok {
			vals[k] = types.StringValue(s)
		} else {
			vals[k] = types.StringNull()
		}
	}
	o, diags := types.ObjectValue(typ, vals)
	if diags.HasError() {
		t.Fatalf("building the timeouts object: %v", diags)
	}
	return timeouts.Value{Object: o}
}

func deadlineIn(t *testing.T, ctx context.Context) time.Duration {
	t.Helper()
	d, ok := ctx.Deadline()
	if !ok {
		t.Fatal("context carries no deadline, so the timeout was never applied")
	}
	return time.Until(d)
}

// The whole point: a value the practitioner sets has to reach the context.
// Before this, all 68 resources declared a timeouts block and none read it.
func TestWithTimeout_UsesTheConfiguredValue(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		attr string
		want time.Duration
		call func(context.Context, timeouts.Value, *diag.Diagnostics) (context.Context, context.CancelFunc)
	}{
		{"create", "create", 45 * time.Minute, WithCreateTimeout},
		{"read", "read", 90 * time.Second, WithReadTimeout},
		{"update", "update", 3 * time.Minute, WithUpdateTimeout},
		{"delete", "delete", 7 * time.Minute, WithDeleteTimeout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got, cancel := c.call(ctx, value(t, map[string]string{c.attr: c.want.String()}), &diags)
			defer cancel()
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if d := deadlineIn(t, got); d > c.want || d < c.want-time.Minute {
				t.Errorf("deadline in %s, want about %s", d, c.want)
			}
		})
	}
}

// No block at all still bounds the call, at the documented default.
func TestWithTimeout_FallsBackToTheDefault(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name string
		want time.Duration
		call func(context.Context, timeouts.Value, *diag.Diagnostics) (context.Context, context.CancelFunc)
	}{
		{"create", DefaultCreateTimeout, WithCreateTimeout},
		{"read", DefaultReadTimeout, WithReadTimeout},
		{"update", DefaultUpdateTimeout, WithUpdateTimeout},
		{"delete", DefaultDeleteTimeout, WithDeleteTimeout},
	} {
		t.Run(c.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got, cancel := c.call(ctx, value(t, nil), &diags)
			defer cancel()
			if d := deadlineIn(t, got); d > c.want || d < c.want-time.Minute {
				t.Errorf("deadline in %s, want the %s default", d, c.want)
			}
		})
	}
}

// An unparseable duration reports the problem AND still bounds the call: an
// unbounded operation is the worse of the two failures.
func TestWithTimeout_BadValueStillBounds(t *testing.T) {
	var diags diag.Diagnostics
	ctx, cancel := WithCreateTimeout(context.Background(), value(t, map[string]string{"create": "not-a-duration"}), &diags)
	defer cancel()
	if !diags.HasError() {
		t.Error("an unparseable duration produced no diagnostic")
	}
	if d := deadlineIn(t, ctx); d > DefaultCreateTimeout || d < DefaultCreateTimeout-time.Minute {
		t.Errorf("deadline in %s, want the %s fallback", d, DefaultCreateTimeout)
	}
}

// A nil diagnostics pointer must not panic: the helper is called from paths
// that have already collected their own.
func TestWithTimeout_NilDiagnostics(t *testing.T) {
	ctx, cancel := WithReadTimeout(context.Background(), value(t, map[string]string{"read": "2m"}), nil)
	defer cancel()
	if d := deadlineIn(t, ctx); d > 2*time.Minute {
		t.Errorf("deadline in %s, want about 2m", d)
	}
}
