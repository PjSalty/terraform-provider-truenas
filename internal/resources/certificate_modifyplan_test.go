package resources

import (
	"strings"
	"testing"

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
