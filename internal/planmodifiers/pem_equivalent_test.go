package planmodifiers_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/planmodifiers"
)

// samplePEM is a valid but not actually-trusted ECDSA P-256 self-signed
// certificate. Generated once for the test suite so the tests run
// without depending on crypto/x509 at runtime. The point is to exercise
// canonicalization, not to verify the cert is usable.
const samplePEM = `-----BEGIN CERTIFICATE-----
MIIBazCCAQ+gAwIBAgIBATAKBggqhkjOPQQDAjAVMRMwEQYDVQQDEwp0ZXN0LXBl
bS1hMB4XDTI2MDQxNDAwMDAwMFoXDTI3MDQxNDAwMDAwMFowFTETMBEGA1UEAxMK
dGVzdC1wZW0tYTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABF1FkQVJkI17vTmk
ZpqHlz/sz5q4GvdkORlrCcRMjEmvCH/LDXfLZgtGWXwBYkdiiuaTdGBAxHRx5HUR
VvWPD9ejPjA8MA4GA1UdDwEB/wQEAwIBBjAPBgNVHRMBAf8EBTADAQH/MBkGA1Ud
DgQSBBBWhnWsZBrw0XMZYhjtsgVwMAoGCCqGSM49BAMCA0gAMEUCIGuJGATnFSvf
8tYbsN8rm/9adYxT+BN7OeXgUZ76i98aAiEA0CmGwyuQRVtbYqVR5+7KxT8bSywG
LI9r6XlGI+a2+Q8=
-----END CERTIFICATE-----
`

func runPEMEquivalent(state, plan string) *planmodifier.StringResponse {
	m := planmodifiers.PEMEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue(state),
		PlanValue:  types.StringValue(plan),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	return resp
}

func TestPEMEquivalent_Identical(t *testing.T) {
	resp := runPEMEquivalent(samplePEM, samplePEM)
	if resp.PlanValue.ValueString() != samplePEM {
		t.Errorf("byte-identical plan should stay unchanged")
	}
}

func TestPEMEquivalent_CRLFvsLF(t *testing.T) {
	lf := samplePEM
	crlf := strings.ReplaceAll(samplePEM, "\n", "\r\n")
	resp := runPEMEquivalent(lf, crlf)
	if resp.PlanValue.ValueString() != lf {
		t.Errorf("CRLF-only diff should normalize to state LF form")
	}
}

func TestPEMEquivalent_TrailingWhitespace(t *testing.T) {
	state := samplePEM
	plan := samplePEM + "\n\n   \n"
	resp := runPEMEquivalent(state, plan)
	if resp.PlanValue.ValueString() != state {
		t.Errorf("trailing whitespace diff should normalize to state")
	}
}

func TestPEMEquivalent_LeadingWhitespace(t *testing.T) {
	state := samplePEM
	plan := "\n\n" + samplePEM
	resp := runPEMEquivalent(state, plan)
	if resp.PlanValue.ValueString() != state {
		t.Errorf("leading whitespace diff should normalize to state")
	}
}

func TestPEMEquivalent_RealDiffStays(t *testing.T) {
	// Two different (fake) PEM payloads — modifier must NOT suppress.
	stateDifferent := strings.Replace(samplePEM, "MIIBazCCAQ+", "DIFFERENT123", 1)
	resp := runPEMEquivalent(samplePEM, stateDifferent)
	if resp.PlanValue.ValueString() != stateDifferent {
		t.Errorf("semantically different PEM must not be suppressed")
	}
}

func TestPEMEquivalent_NonPEMInputs(t *testing.T) {
	// Non-PEM strings fall through unchanged (lets downstream modifiers
	// and validators still see the raw diff).
	resp := runPEMEquivalent("not-pem-state", "not-pem-plan")
	if resp.PlanValue.ValueString() != "not-pem-plan" {
		t.Errorf("non-PEM inputs should pass through: got %q", resp.PlanValue.ValueString())
	}
}

func TestPEMEquivalent_PEMStateNonPEMPlan(t *testing.T) {
	// User typo'd the PEM — fall through so real diff surfaces.
	resp := runPEMEquivalent(samplePEM, "totally not a cert")
	if resp.PlanValue.ValueString() != "totally not a cert" {
		t.Errorf("non-PEM plan must not be normalized against PEM state")
	}
}

// TestPEMEquivalent_PEMPlanNonPEMState exercises the `stateCanon, ok :=
// canonicalizePEM(state); if !ok { return }` branch which is NOT hit
// by the sibling test above — that one fails at the PLAN canonicalization
// step before stateCanon is evaluated. This case passes plan canonicalization
// and then fails state canonicalization, closing the last 6.7% gap on
// PlanModifyString.
func TestPEMEquivalent_PEMPlanNonPEMState(t *testing.T) {
	// State is garbage (maybe an import from a half-migrated state file);
	// plan is a valid cert. No canonicalization possible — fall through
	// so the real diff surfaces normally.
	resp := runPEMEquivalent("garbage-state-not-pem", samplePEM)
	if resp.PlanValue.ValueString() != samplePEM {
		t.Errorf("PEM plan with non-PEM state must not be rewritten")
	}
}

func TestPEMEquivalent_NullState(t *testing.T) {
	m := planmodifiers.PEMEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue(samplePEM),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	if resp.PlanValue.ValueString() != samplePEM {
		t.Errorf("null state should leave plan unchanged")
	}
}

func TestPEMEquivalent_UnknownPlan(t *testing.T) {
	m := planmodifiers.PEMEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue(samplePEM),
		PlanValue:  types.StringUnknown(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	if !resp.PlanValue.IsUnknown() {
		t.Error("unknown plan should stay unknown")
	}
}

func TestPEMEquivalent_NullPlan(t *testing.T) {
	m := planmodifiers.PEMEquivalent()
	req := planmodifier.StringRequest{
		StateValue: types.StringValue(samplePEM),
		PlanValue:  types.StringNull(),
	}
	resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
	m.PlanModifyString(context.Background(), req, resp)
	if !resp.PlanValue.IsNull() {
		t.Error("null plan should stay null")
	}
}

func TestPEMEquivalent_Descriptions(t *testing.T) {
	m := planmodifiers.PEMEquivalent()
	if d := m.Description(context.Background()); d == "" {
		t.Error("Description must be non-empty")
	}
	if d := m.MarkdownDescription(context.Background()); d == "" {
		t.Error("MarkdownDescription must be non-empty")
	}
}
