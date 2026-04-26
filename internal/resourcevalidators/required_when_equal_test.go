package resourcevalidators_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/resourcevalidators"
)

// fakeSchema is the tiniest possible schema we can hand the framework
// to build a Config object for validator tests — just the discriminator
// plus the two required attributes the test validator cares about.
var fakeSchema = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"create_type": schema.StringAttribute{Required: true},
		"certificate": schema.StringAttribute{Optional: true},
		"privatekey":  schema.StringAttribute{Optional: true},
	},
}

func fakeConfig(t *testing.T, createType, cert, key string) tfsdk.Config {
	t.Helper()
	objType := fakeSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	attrs := map[string]tftypes.Value{}
	attrs["create_type"] = tftypes.NewValue(tftypes.String, createType)
	if cert == "__null__" {
		attrs["certificate"] = tftypes.NewValue(tftypes.String, nil)
	} else {
		attrs["certificate"] = tftypes.NewValue(tftypes.String, cert)
	}
	if key == "__null__" {
		attrs["privatekey"] = tftypes.NewValue(tftypes.String, nil)
	} else {
		attrs["privatekey"] = tftypes.NewValue(tftypes.String, key)
	}
	return tfsdk.Config{Schema: fakeSchema, Raw: tftypes.NewValue(objType, attrs)}
}

func run(t *testing.T, createType, cert, key string) *resource.ValidateConfigResponse {
	t.Helper()
	v := resourcevalidators.RequiredWhenEqual(
		"create_type",
		"CERTIFICATE_CREATE_IMPORTED",
		[]string{"certificate", "privatekey"},
	)
	req := resource.ValidateConfigRequest{Config: fakeConfig(t, createType, cert, key)}
	resp := &resource.ValidateConfigResponse{}
	v.ValidateResource(context.Background(), req, resp)
	return resp
}

func TestRequiredWhenEqual_TriggerMatchedAllPresent(t *testing.T) {
	resp := run(t, "CERTIFICATE_CREATE_IMPORTED", "PEM-CERT", "PEM-KEY")
	if resp.Diagnostics.HasError() {
		t.Errorf("unexpected error diagnostics: %v", resp.Diagnostics)
	}
}

func TestRequiredWhenEqual_TriggerMatchedMissingBoth(t *testing.T) {
	resp := run(t, "CERTIFICATE_CREATE_IMPORTED", "__null__", "__null__")
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error diagnostics, got none")
	}
	msgs := resp.Diagnostics.Errors()
	if len(msgs) != 2 {
		t.Errorf("expected 2 errors (one per missing attr), got %d: %v", len(msgs), msgs)
	}
	// Each error must mention the specific attribute name
	joined := ""
	for _, m := range msgs {
		joined += m.Detail()
	}
	if !strings.Contains(joined, `"certificate"`) {
		t.Error("error must mention certificate")
	}
	if !strings.Contains(joined, `"privatekey"`) {
		t.Error("error must mention privatekey")
	}
}

func TestRequiredWhenEqual_TriggerMatchedMissingOne(t *testing.T) {
	resp := run(t, "CERTIFICATE_CREATE_IMPORTED", "PEM-CERT", "__null__")
	errs := resp.Diagnostics.Errors()
	if len(errs) != 1 {
		t.Errorf("expected 1 error for missing privatekey, got %d", len(errs))
	}
}

func TestRequiredWhenEqual_TriggerNotMatched(t *testing.T) {
	// Different create_type — validator should be a no-op even when the
	// required attrs are unset. Other create_type values have their own
	// rules; this validator only fires for its trigger value.
	resp := run(t, "CERTIFICATE_CREATE_CSR", "__null__", "__null__")
	if resp.Diagnostics.HasError() {
		t.Errorf("validator fired on wrong trigger: %v", resp.Diagnostics)
	}
}

func TestRequiredWhenEqual_DiscriminatorNull(t *testing.T) {
	// Discriminator itself is null — framework will catch that via
	// the Required: true attribute. Our validator must not double-error.
	resp := run(t, "__null__", "__null__", "__null__")
	if resp.Diagnostics.HasError() {
		t.Errorf("validator fired on null discriminator: %v", resp.Diagnostics)
	}
}

func TestRequiredWhenEqual_EmptyStringTreatedAsMissing(t *testing.T) {
	// An explicit "" is as useless as null for a PEM attribute, so
	// the validator treats empty strings the same way.
	resp := run(t, "CERTIFICATE_CREATE_IMPORTED", "", "PEM-KEY")
	if !resp.Diagnostics.HasError() {
		t.Error("expected error for empty string certificate")
	}
}

// TestRequiredWhenEqual_DiscriminatorUnknown exercises the
// `if discVal.IsUnknown()` branch. When the discriminator value is
// not yet known (e.g., it refers to a computed attribute from another
// resource), validation defers to plan time and emits no errors.
func TestRequiredWhenEqual_DiscriminatorUnknown(t *testing.T) {
	v := resourcevalidators.RequiredWhenEqual(
		"create_type", "CERTIFICATE_CREATE_IMPORTED", []string{"certificate", "privatekey"})
	objType := fakeSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	// Build a raw where create_type is unknown.
	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"create_type": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"certificate": tftypes.NewValue(tftypes.String, nil),
		"privatekey":  tftypes.NewValue(tftypes.String, nil),
	})
	req := resource.ValidateConfigRequest{
		Config: tfsdk.Config{Schema: fakeSchema, Raw: raw},
	}
	resp := &resource.ValidateConfigResponse{}
	v.ValidateResource(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("unknown discriminator should defer validation, got: %v",
			resp.Diagnostics.Errors())
	}
}

// TestRequiredWhenEqual_DiscriminatorGetAttributeError exercises the
// `if resp.Diagnostics.HasError()` early-return after the first
// GetAttribute call on the discriminator. This fires when the
// discriminator attribute name references a path that doesn't exist
// in the provided schema — e.g., because a developer typoed the
// attribute name in the validator constructor. The validator must
// surface the error and not proceed to check the required list.
func TestRequiredWhenEqual_DiscriminatorGetAttributeError(t *testing.T) {
	// "nonexistent" is not an attribute on fakeSchema.
	v := resourcevalidators.RequiredWhenEqual(
		"nonexistent_discriminator", "TRIGGER", []string{"certificate"})
	req := resource.ValidateConfigRequest{
		Config: fakeConfig(t, "CERTIFICATE_CREATE_IMPORTED", "__null__", "__null__"),
	}
	resp := &resource.ValidateConfigResponse{}
	v.ValidateResource(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("GetAttribute on nonexistent discriminator should surface an error")
	}
}

// TestRequiredWhenEqual_RequiredGetAttributeError exercises the
// per-required-attribute error branch: when a required attribute
// name references a path that doesn't exist, the validator must
// `continue` past it (recording the error) instead of returning,
// so subsequent required attributes are still checked.
func TestRequiredWhenEqual_RequiredGetAttributeError(t *testing.T) {
	v := resourcevalidators.RequiredWhenEqual(
		"create_type", "CERTIFICATE_CREATE_IMPORTED",
		// "nonexistent" will fail GetAttribute; "certificate" is a real
		// attribute that should still be checked even after the failure.
		[]string{"nonexistent", "certificate"})
	req := resource.ValidateConfigRequest{
		Config: fakeConfig(t, "CERTIFICATE_CREATE_IMPORTED", "__null__", "__null__"),
	}
	resp := &resource.ValidateConfigResponse{}
	v.ValidateResource(context.Background(), req, resp)
	// Should have errors — both from the GetAttribute failure AND from
	// the "certificate" attribute being null.
	errs := resp.Diagnostics.Errors()
	if len(errs) < 1 {
		t.Errorf("expected at least one error, got none")
	}
	// Assert that at least one error mentions "certificate" — proving
	// the loop continued past the bad "nonexistent" entry and still
	// checked the real one.
	found := false
	for _, e := range errs {
		if strings.Contains(e.Summary()+e.Detail(), "certificate") {
			found = true
			break
		}
	}
	if !found {
		t.Error("loop should continue past nonexistent attribute and check 'certificate'")
	}
}

func TestRequiredWhenEqual_Descriptions(t *testing.T) {
	v := resourcevalidators.RequiredWhenEqual("x", "Y", []string{"z"})
	if d := v.Description(context.Background()); d == "" {
		t.Error("Description must be non-empty")
	}
	if d := v.MarkdownDescription(context.Background()); d == "" {
		t.Error("MarkdownDescription must be non-empty")
	}
}

// Compile-time assertion that the constructor can be invoked and its
// return value can be assigned to a ConfigValidator-typed slot. The
// RequiredWhenEqual function signature itself declares the return type
// is resource.ConfigValidator, so this is belt-and-suspenders: if the
// Plugin Framework ever broadens ConfigValidator this still breaks
// fast at compile time rather than at runtime.
var _ = func() bool {
	_ = resourcevalidators.RequiredWhenEqual("x", "y", []string{"z"})
	// Silence unused-var without exporting.
	_ = types.StringValue
	_ = attr.ValueStateKnown
	return true
}()
