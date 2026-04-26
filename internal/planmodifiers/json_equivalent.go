package planmodifiers

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// jsonEquivalent implements planmodifier.String for attributes that carry
// JSON blobs. When the plan and state values differ only in whitespace
// or key ordering, it rewrites the plan value back to the prior state
// value so Terraform does not report a spurious change.
//
// Use on any attribute declared as a JSON string whose canonical form
// the server may normalize (reorder keys, strip whitespace, drop
// server-added defaults). Pair it with a server-side `mapResponseToModel`
// that filters to the user's original key set — the plan modifier is
// the last defense line for round-trip stability.
type jsonEquivalent struct {
	description string
}

// JSONEquivalent returns a plan modifier that treats two JSON strings
// as equal when their decoded forms match structurally, ignoring
// whitespace and top-level key ordering. Non-JSON or unparseable values
// fall through unchanged.
func JSONEquivalent() planmodifier.String {
	return jsonEquivalent{
		description: "Suppresses spurious diffs on JSON string attributes when the decoded value is unchanged but whitespace or key ordering differs.",
	}
}

func (m jsonEquivalent) Description(_ context.Context) string {
	return m.description
}

func (m jsonEquivalent) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jsonEquivalent) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
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

	var planObj, stateObj any
	if err := json.Unmarshal([]byte(plan), &planObj); err != nil {
		return // not parseable JSON — let the diff stand
	}
	if err := json.Unmarshal([]byte(state), &stateObj); err != nil {
		return
	}

	// Re-encode both sides canonically and compare. json.Marshal can't
	// fail here — the inputs are Go types decoded via json.Unmarshal, so
	// they're guaranteed to be marshalable.
	planCanon, _ := json.Marshal(planObj)
	stateCanon, _ := json.Marshal(stateObj)
	if bytes.Equal(planCanon, stateCanon) {
		// Plan matches state semantically — preserve state so the framework
		// doesn't report a change.
		resp.PlanValue = types.StringValue(state)
	}
}
