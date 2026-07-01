package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// Repro for issue #21 / PR #22: filesystem.setperm completes SUCCESS but the
// immediate filesystem.stat can return STALE uid/gid (TrueNAS caching). The
// plan asked for uid/gid=1000; a stale stat returns 0. State must reflect the
// applied 1000, not the stale 0. Fails on clean main (bug), passes with the fix.
func TestDirectoryResource_StaleStatUidGid_repro(t *testing.T) {
	ctx := context.Background()
	r := newDirResource(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "filesystem.mkdir":
			return map[string]interface{}{"realpath": "/mnt/tank/d", "type": "DIRECTORY", "mode": 0o40755, "uid": 1000, "gid": 1000}, nil
		case "filesystem.setperm":
			return int64(1), nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": int64(1), "state": "SUCCESS"}}, nil
		case "filesystem.stat":
			// STALE read: gid/uid came back 0 despite setperm applying 1000
			return map[string]interface{}{"realpath": "/mnt/tank/d", "type": "DIRECTORY", "mode": 0o40755, "uid": 0, "gid": 0}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	sch := schemaOf(t, ctx, r)
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, dirPlanVals("/mnt/tank/d", "755", false))}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	var m DirectoryResourceModel
	resp.State.Get(ctx, &m)
	if m.UID.ValueInt64() != 1000 || m.GID.ValueInt64() != 1000 {
		t.Fatalf("BUG (issue #21): stale stat leaked into state: uid=%d gid=%d, want 1000/1000",
			m.UID.ValueInt64(), m.GID.ValueInt64())
	}
}
