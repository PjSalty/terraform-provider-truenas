package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCertificate_ModifyPlan_ImportedMissingCert(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestCertificate_ModifyPlan_ImportedMissingKey(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate": str("-----BEGIN CERTIFICATE-----..."),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

func TestCertificate_ModifyPlan_CSRMissingIdentity(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error")
	}
}

// A common name alone is NOT enough for a CSR. This used to assert the
// opposite, which is what let the configuration in GitHub issue 33 plan clean
// and then fail at apply: TrueNAS validates a CSR against a model declaring
// san with min_length=1, and requires key_length and digest_algorithm for an
// RSA key.
func TestCertificate_ModifyPlan_CSRWithCommonAloneIsRejected(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"common":      str("example.com"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("a CSR with only a common name was accepted; the server rejects it")
	}
	want := []string{"san", "key_length"}
	got := resp.Diagnostics.Errors().Errors()
	for _, w := range want {
		found := false
		for _, d := range got {
			if strings.Contains(d.Detail()+d.Summary(), w) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no diagnostic mentioned %q; got %v", w, got)
		}
	}
}

func TestCertificate_ModifyPlan_CSRWithSAN(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	sanList := tftypes.NewValue(
		tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{str("example.com")},
	)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":             str("c1"),
		"create_type":      str("CERTIFICATE_CREATE_CSR"),
		"san":              sanList,
		"key_type":         str("RSA"),
		"key_length":       num(4096),
		"digest_algorithm": str("SHA256"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// digest_algorithm carries a model default of SHA256, so the server never sees
// it missing and the provider must not demand it. Requiring it would refuse a
// configuration that applies cleanly.
func TestCertificate_ModifyPlan_CSRWithoutDigestAlgorithm(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	sanList := tftypes.NewValue(
		tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{str("example.com")},
	)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"san":         sanList,
		"key_type":    str("RSA"),
		"key_length":  num(4096),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// An EC key needs no key_length: the server only demands it when the key is
// not EC.
func TestCertificate_ModifyPlan_CSRWithECKeyNeedsNoKeyLength(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	sanList := tftypes.NewValue(
		tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{str("example.com")},
	)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"san":         sanList,
		"key_type":    str("EC"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestCertificate_ModifyPlan_ImportedValid(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate": str("-----BEGIN CERTIFICATE-----..."),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// ACME used to be unchecked here, and unusable in practice: every field the
// server requires for it was missing from the provider entirely, so the create
// failed with three EINVALs naming fields nobody could set. All five are now
// required at plan time instead.
func TestCertificate_ModifyPlan_ACMERequiresItsFields(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_ACME"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("an ACME certificate with no ACME fields was accepted")
	}
	for _, want := range []string{"tos", "csr_id", "acme_directory_uri", "dns_mapping"} {
		found := false
		for _, d := range resp.Diagnostics.Errors().Errors() {
			if strings.Contains(d.Summary(), want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no diagnostic named %q", want)
		}
	}
}

// The same fields are rejected on any other create_type, so a practitioner who
// sets one by mistake hears it at plan rather than from middleware.
func TestCertificate_ModifyPlan_ACMEFieldsRejectedElsewhere(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate": str("-----BEGIN CERTIFICATE-----..."),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
		"tos":         tftypes.NewValue(tftypes.Bool, true),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("tos was accepted on an IMPORTED certificate")
	}
}

func TestCertificate_ModifyPlan_Delete(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlanDelete(t, r)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// TestCertificate_ModifyPlan_CreateTypeNull exercises the early return when
// create_type is null, the cross-attribute rules can't fire without knowing
// the intended create type, so ModifyPlan must leave the plan untouched.
func TestCertificate_ModifyPlan_CreateTypeNull(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name": str("c1"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// renew_days defaults to 10 upstream, so an ACME certificate that omits it is
// valid and the provider must not ask for it.
func TestCertificate_ModifyPlan_ACMEWithoutRenewDays(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":               str("c1"),
		"create_type":        str("CERTIFICATE_CREATE_ACME"),
		"tos":                tftypes.NewValue(tftypes.Bool, true),
		"csr_id":             num(1),
		"acme_directory_uri": str("https://acme-v02.api.letsencrypt.org/directory"),
		"dns_mapping": tftypes.NewValue(
			tftypes.Map{ElementType: tftypes.Number},
			map[string]tftypes.Value{"example.com": num(1)},
		),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// unknownOf returns a tftypes value that is present in configuration but whose
// value Terraform cannot resolve yet, which is what an interpolation from
// another resource looks like at plan time.
func unknownOf(t tftypes.Type) tftypes.Value {
	return tftypes.NewValue(t, tftypes.UnknownValue)
}

// The documented two-resource ACME workflow wires
// csr_id = truenas_certificate.csr.id, which is UNKNOWN on a first apply.
// Treating unknown as unset rejected the provider's own published example with
// "Missing csr_id", the same class of unusable-create_type error this change
// set out to remove.
func TestCertificate_ModifyPlan_ACMEAcceptsUnknownFields(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":               str("acme"),
		"create_type":        str("CERTIFICATE_CREATE_ACME"),
		"tos":                tftypes.NewValue(tftypes.Bool, true),
		"csr_id":             unknownOf(tftypes.Number),
		"acme_directory_uri": str("https://acme-v02.api.letsencrypt.org/directory"),
		"dns_mapping": tftypes.NewValue(
			tftypes.Map{ElementType: tftypes.Number},
			map[string]tftypes.Value{"example.com": unknownOf(tftypes.Number)},
		),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("an unknown csr_id was treated as missing: %v", resp.Diagnostics)
	}
}

// Every ACME field can be interpolated, not just csr_id.
func TestCertificate_ModifyPlan_ACMEAcceptsAllUnknown(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":               str("acme"),
		"create_type":        str("CERTIFICATE_CREATE_ACME"),
		"tos":                unknownOf(tftypes.Bool),
		"csr_id":             unknownOf(tftypes.Number),
		"acme_directory_uri": unknownOf(tftypes.String),
		"renew_days":         unknownOf(tftypes.Number),
		"dns_mapping":        unknownOf(tftypes.Map{ElementType: tftypes.Number}),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown ACME fields were treated as missing: %v", resp.Diagnostics)
	}
}

// The rejection direction wants the opposite reading of unknown: an unknown
// value means the practitioner DID write the attribute, so it is still wrong
// on a create_type that has no use for it.
func TestCertificate_ModifyPlan_UnknownACMEFieldStillRejectedElsewhere(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate": str("-----BEGIN CERTIFICATE-----"),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
		"csr_id":      unknownOf(tftypes.Number),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("an unknown csr_id was accepted on an IMPORTED certificate")
	}
}

// A CSR whose san comes from a variable or another resource cannot be counted
// at plan time, so it has to be left to the server rather than called missing.
func TestCertificate_ModifyPlan_CSRAcceptsUnknownSAN(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("csr"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"san":         unknownOf(tftypes.List{ElementType: tftypes.String}),
		"key_type":    str("RSA"),
		"key_length":  num(4096),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("an unknown san was treated as missing: %v", resp.Diagnostics)
	}
}

// An unknown key_type could resolve to EC, so the key_length rule cannot be
// decided at plan time and must not fire.
func TestCertificate_ModifyPlan_CSRUnknownKeyTypeDoesNotDemandKeyLength(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	sanList := tftypes.NewValue(
		tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{str("example.com")},
	)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("csr"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"san":         sanList,
		"key_type":    unknownOf(tftypes.String),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("an unknown key_type demanded key_length: %v", resp.Diagnostics)
	}
}

// CERTIFICATE_CREATE_IMPORTED_CSR was in the schema's OneOf and in the docs and
// could never succeed: its payload model is name + CSR + privatekey, and the
// provider had no csr attribute to send.
func TestCertificate_ModifyPlan_ImportedCSRRequiresCSRAndKey(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("imported-csr"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED_CSR"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("an IMPORTED_CSR with neither csr nor privatekey was accepted")
	}
	for _, want := range []string{"Missing csr", "Missing privatekey"} {
		found := false
		for _, d := range resp.Diagnostics.Errors().Errors() {
			if d.Summary() == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no diagnostic %q", want)
		}
	}
}

func TestCertificate_ModifyPlan_ImportedCSRValid(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("imported-csr"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED_CSR"),
		"csr":         str("-----BEGIN CERTIFICATE REQUEST-----"),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

// csr belongs to exactly one create_type, the same way the ACME fields do.
func TestCertificate_ModifyPlan_CSRFieldRejectedElsewhere(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"certificate": str("-----BEGIN CERTIFICATE-----"),
		"privatekey":  str("PRIVATE_KEY_PLACEHOLDER"),
		"csr":         str("-----BEGIN CERTIFICATE REQUEST-----"),
	})
	if !resp.Diagnostics.HasError() {
		t.Fatal("csr was accepted on an IMPORTED certificate")
	}
}

// An empty string and an empty map both satisfied "is this attribute set" and
// were then erased from the request by omitempty, so the server saw them
// missing and answered with the same error this change exists to prevent.
// Both are now refused by the schema, which is where a value rule belongs:
// validators run before ModifyPlan, so the check never even gets there.
func TestCertificate_EmptyACMEValuesAreRefusedBySchema(t *testing.T) {
	ctx := context.Background()
	r := NewCertificateResource()
	sr := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sr)
	if sr.Diagnostics.HasError() {
		t.Fatalf("schema: %v", sr.Diagnostics)
	}
	attrs := sr.Schema.GetAttributes()

	uri, ok := attrs["acme_directory_uri"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("acme_directory_uri is %T", attrs["acme_directory_uri"])
	}
	var uriResp validator.StringResponse
	for _, v := range uri.Validators {
		v.ValidateString(ctx, validator.StringRequest{
			Path:        path.Root("acme_directory_uri"),
			ConfigValue: types.StringValue(""),
		}, &uriResp)
	}
	if !uriResp.Diagnostics.HasError() {
		t.Error("acme_directory_uri accepted an empty string")
	}

	csr, ok := attrs["csr"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("csr is %T", attrs["csr"])
	}
	var csrResp validator.StringResponse
	for _, v := range csr.Validators {
		v.ValidateString(ctx, validator.StringRequest{
			Path:        path.Root("csr"),
			ConfigValue: types.StringValue(""),
		}, &csrResp)
	}
	if !csrResp.Diagnostics.HasError() {
		t.Error("csr accepted an empty string")
	}

	dm, ok := attrs["dns_mapping"].(schema.MapAttribute)
	if !ok {
		t.Fatalf("dns_mapping is %T", attrs["dns_mapping"])
	}
	empty, diags := types.MapValue(types.Int64Type, map[string]attr.Value{})
	if diags.HasError() {
		t.Fatalf("building an empty map: %v", diags)
	}
	var dmResp validator.MapResponse
	for _, v := range dm.Validators {
		v.ValidateMap(ctx, validator.MapRequest{
			Path:        path.Root("dns_mapping"),
			ConfigValue: empty,
		}, &dmResp)
	}
	if !dmResp.Diagnostics.HasError() {
		t.Error("dns_mapping accepted an empty map")
	}
}
