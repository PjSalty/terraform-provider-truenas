package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestCertificate_ModifyPlan_ImportedMissingCert(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_IMPORTED"),
		"privatekey":  str("-----BEGIN PRIVATE KEY-----..."),
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

func TestCertificate_ModifyPlan_CSRWithCommon(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"common":      str("example.com"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestCertificate_ModifyPlan_CSRWithSAN(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	sanList := tftypes.NewValue(
		tftypes.List{ElementType: tftypes.String},
		[]tftypes.Value{str("example.com")},
	)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_CSR"),
		"san":         sanList,
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
		"privatekey":  str("-----BEGIN PRIVATE KEY-----..."),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
	}
}

func TestCertificate_ModifyPlan_ACMENoCheck(t *testing.T) {
	r := NewCertificateResource().(*CertificateResource)
	resp := callModifyPlan(t, r, map[string]tftypes.Value{
		"name":        str("c1"),
		"create_type": str("CERTIFICATE_CREATE_ACME"),
	})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected: %v", resp.Diagnostics)
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
// create_type is null — the cross-attribute rules can't fire without knowing
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
