package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Import seeds id and path from the import ID plus create_parents=false.
// create_parents is config-only (mkdir -p behavior, meaningless after
// creation); leaving it null made ImportStateVerify fail against the
// post-apply state, which stores the default false.
func TestDirectoryResource_ImportState_CreateParentsFalse(t *testing.T) {
	ctx := context.Background()
	r := NewDirectoryResource().(resource.ResourceWithImportState)
	state, err := newPrimedState(ctx, r.(resource.Resource))
	if err != nil {
		t.Fatalf("newPrimedState: %v", err)
	}
	resp := &resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "/mnt/tank/d"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState: %v", resp.Diagnostics)
	}

	var id, dirPath types.String
	if d := resp.State.GetAttribute(ctx, path.Root("id"), &id); d.HasError() {
		t.Fatalf("GetAttribute(id): %v", d)
	}
	if d := resp.State.GetAttribute(ctx, path.Root("path"), &dirPath); d.HasError() {
		t.Fatalf("GetAttribute(path): %v", d)
	}
	if id.ValueString() != "/mnt/tank/d" || dirPath.ValueString() != "/mnt/tank/d" {
		t.Errorf("id=%q path=%q, want /mnt/tank/d for both", id.ValueString(), dirPath.ValueString())
	}

	var cp types.Bool
	if d := resp.State.GetAttribute(ctx, path.Root("create_parents"), &cp); d.HasError() {
		t.Fatalf("GetAttribute(create_parents): %v", d)
	}
	if cp.IsNull() || cp.IsUnknown() {
		t.Fatalf("create_parents is null/unknown after import, want known false")
	}
	if cp.ValueBool() {
		t.Errorf("create_parents = true after import, want false")
	}
}
