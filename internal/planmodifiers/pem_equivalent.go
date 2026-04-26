package planmodifiers

import (
	"context"
	"encoding/pem"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// pemEquivalent implements planmodifier.String for attributes that carry
// PEM-encoded certificate or key material. When the plan and state values
// decode to the same PEM blocks — ignoring line endings, trailing
// whitespace, and re-encoding differences the server may introduce on
// read-back — it rewrites the plan value back to the state value so
// Terraform does not report a spurious change.
//
// TrueNAS SCALE normalizes the PEM it receives (CRLF → LF, chain
// reassembly, canonical base64 wrap width). Without this modifier, an
// otherwise-unchanged apply appears as a diff on `certificate` or
// `privatekey`, which combined with `RequiresReplace` forces a delete +
// create roundtrip even when only a peer attribute like `name` changed.
type pemEquivalent struct {
	description string
}

// PEMEquivalent returns a plan modifier that treats two PEM strings as
// equal when their decoded blocks match byte-for-byte, ignoring
// whitespace differences and the trailing newline. Non-PEM or
// unparseable values fall through unchanged so downstream modifiers
// (RequiresReplace, validators) still see the raw diff.
func PEMEquivalent() planmodifier.String {
	return pemEquivalent{
		description: "Suppresses spurious diffs on PEM-encoded string attributes when the decoded block sequence is unchanged but whitespace, line endings, or base64 line wrapping differ.",
	}
}

func (m pemEquivalent) Description(_ context.Context) string {
	return m.description
}

func (m pemEquivalent) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m pemEquivalent) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Nothing to compare during create or when plan is still unknown.
	if req.StateValue.IsNull() || req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	plan := req.PlanValue.ValueString()
	state := req.StateValue.ValueString()

	// Fast path: byte-identical. No change needed.
	if plan == state {
		return
	}

	planCanon, ok := canonicalizePEM(plan)
	if !ok {
		return // not parseable PEM — let the diff stand
	}
	stateCanon, ok := canonicalizePEM(state)
	if !ok {
		return
	}
	if planCanon == stateCanon {
		// Plan matches state semantically — preserve state so the framework
		// doesn't report a change and downstream RequiresReplace does not
		// trigger on cosmetic normalization. This is the interesting
		// positive case so emit a trace breadcrumb for operator visibility.
		tflog.Trace(ctx, "PEMEquivalent: semantic match, suppressing diff", map[string]interface{}{
			"attribute":   req.Path.String(),
			"canon_bytes": len(planCanon),
		})
		resp.PlanValue = types.StringValue(state)
	}
}

// canonicalizePEM decodes every PEM block in data and re-encodes each one
// through encoding/pem. The output has:
//
//   - exactly one LF between blocks and at end-of-string
//   - no carriage returns
//   - canonical base64 line-wrap width (64 chars)
//   - header keys sorted (encoding/pem does this on encode)
//
// Two PEM strings that canonicalize to the same bytes encode the same
// block sequence regardless of how the source was formatted. Returns
// false when the input contains zero valid PEM blocks so non-PEM text
// (e.g. an accidentally-plain string or a CSR chain with extra garbage)
// falls through to a normal string diff.
func canonicalizePEM(data string) (string, bool) {
	rest := []byte(strings.TrimSpace(data))
	var out strings.Builder
	found := false
	for {
		block, next := pem.Decode(rest)
		if block == nil {
			break
		}
		found = true
		out.Write(pem.EncodeToMemory(block))
		rest = next
	}
	if !found {
		return "", false
	}
	return strings.TrimRight(out.String(), "\n"), true
}
