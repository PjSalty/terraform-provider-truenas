package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// `sudo_commands = []` must put an explicit [] on the wire. Dropped by
// omitempty, the grant survived and the apply failed with an inconsistent
// result.
func TestUserResource_Update_emptySudoCommandsClears(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, passwordlessUserEntity(true), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		"sudo_commands":     strList("ALL"),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
		"sudo_commands":     strList(),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	wireList(t, got, "sudo_commands")
}

func TestGroupResource_Update_emptySudoCommandsClears(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := groupRecorder(ctx, t, sudoGroupEntity(), &got)
	r := &GroupResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
		"sudo_commands": strList("ALL"),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
		"sudo_commands": strList(),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	wireList(t, got, "sudo_commands")
}
