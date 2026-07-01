package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// Sibling probe for PR #22: identical stale-stat scenario to the uid/gid
// repro, but the STALE field is mode. Plan asks mode "750"; setperm returns
// SUCCESS; the immediate filesystem.stat returns stale 0o40755. PR #22 pins
// uid/gid from the plan after mapStatToModel but NOT mode, so the stale 755
// leaks into state. On a real apply Terraform core then errors "Provider
// produced inconsistent result after apply" (mode is user-configured in the
// plan). A FAIL here means PR #22 is incomplete for the same bug class.
func TestDirectoryResource_StaleStatMode_siblingProbe(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "filesystem.setperm":
			return int64(1), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": int64(1), "state": "SUCCESS"}}, nil
		case "filesystem.stat":
			// STALE read: mode came back 0755 despite setperm applying 0750.
			// uid/gid are fresh (1000) so mode is the only stale field.
			return map[string]interface{}{"realpath": "/mnt/tank/d", "type": "DIRECTORY", "mode": 0o40755, "uid": 1000, "gid": 1000}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	sch := schemaOf(t, ctx, r)
	vals := dirPlanVals("/mnt/tank/d", "750", false)
	vals["id"] = str("/mnt/tank/d")
	st := stateFromValues(t, ctx, sch, vals)
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	var m DirectoryResourceModel
	resp.State.Get(ctx, &m)
	if got := m.Mode.ValueString(); got != "750" {
		t.Fatalf("SIBLING BUG: stale stat mode leaked into state: mode=%q, want %q (PR #22 pins uid/gid but not mode)", got, "750")
	}
}
