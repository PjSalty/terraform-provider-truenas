package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// --- helpers ---

// An unset attribute must NOT clear whatever the box already grants. A nil
// return means "send nothing".
func TestNopasswdFromList_nullAndUnknownSendNothing(t *testing.T) {
	if got := nopasswdFromList(types.ListNull(types.StringType)); got != nil {
		t.Fatalf("null list must send nothing, got %#v", got)
	}
	if got := nopasswdFromList(types.ListUnknown(types.StringType)); got != nil {
		t.Fatalf("unknown list must send nothing, got %#v", got)
	}
}

func TestNopasswdRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   []string
	}{
		{"empty", []string{}},
		{"all", []string{"ALL"}},
		{"several", []string{"/usr/bin/qemu-img", "/usr/bin/mount"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ptr := nopasswdFromList(nopasswdToList(tc.in))
			if ptr == nil {
				t.Fatal("a known list must be sent, got nil")
			}
			if len(*ptr) != len(tc.in) {
				t.Fatalf("length changed: %d -> %d", len(tc.in), len(*ptr))
			}
			for i := range tc.in {
				if (*ptr)[i] != tc.in[i] {
					t.Fatalf("element %d: %q != %q", i, (*ptr)[i], tc.in[i])
				}
			}
		})
	}
}

// A nil slice from the API must become an EMPTY list, never null: the attribute
// is Computed, so null would leave it unknown after apply.
func TestNopasswdToList_nilBecomesEmptyNotNull(t *testing.T) {
	l := nopasswdToList(nil)
	if l.IsNull() || l.IsUnknown() {
		t.Fatalf("nil slice must map to a known empty list, got null=%v unknown=%v", l.IsNull(), l.IsUnknown())
	}
	if n := len(l.Elements()); n != 0 {
		t.Fatalf("expected 0 elements, got %d", n)
	}
}

// wireList asserts a captured request field is a JSON list of exactly want.
func wireList(t *testing.T, got map[string]interface{}, key string, want ...string) {
	t.Helper()
	v, present := got[key]
	if !present {
		t.Fatalf("%s absent from the request: %v", key, got)
	}
	list, ok := v.([]interface{})
	if !ok {
		t.Fatalf("%s = %#v, want a list", key, v)
	}
	if len(list) != len(want) {
		t.Fatalf("%s = %v, want %v", key, list, want)
	}
	for i := range want {
		if list[i] != want[i] {
			t.Fatalf("%s[%d] = %v, want %q", key, i, list[i], want[i])
		}
	}
}

// --- truenas_user ---

func sudoUserEntity(nopasswd ...string) map[string]interface{} {
	e := passwordlessUserEntity(true)
	e["sudo_commands_nopasswd"] = nopasswd
	return e
}

// The point of the attribute: an automation account that needs
// non-interactive sudo can be created in one apply.
func TestUserResource_Create_sendsSudoNopasswd(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userCreateRecorder(ctx, t, sudoUserEntity("ALL"), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"username": str("svc"), "full_name": str("svc"),
		"password_disabled":      tftypes.NewValue(tftypes.Bool, true),
		"sudo_commands_nopasswd": strList("ALL"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
	wireList(t, got, "sudo_commands_nopasswd", "ALL")
}

func TestUserResource_Create_unsetSudoNopasswdOmitsTheKey(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userCreateRecorder(ctx, t, sudoUserEntity(), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"username": str("svc"), "full_name": str("svc"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
	if _, present := got["sudo_commands_nopasswd"]; present {
		t.Errorf("sudo_commands_nopasswd sent on a create that never set it: %v", got)
	}
}

// Removing a grant must put an explicit [] on the wire. Dropped by omitempty,
// the account would keep passwordless sudo while state said it had none.
func TestUserResource_Update_emptySudoNopasswdClears(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, sudoUserEntity(), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
		"password_disabled":      tftypes.NewValue(tftypes.Bool, true),
		"sudo_commands_nopasswd": strList("ALL"),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
		"password_disabled":      tftypes.NewValue(tftypes.Bool, true),
		"sudo_commands_nopasswd": strList(),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	wireList(t, got, "sudo_commands_nopasswd")
}

// Left out of the config, the attribute is not Terraform's to change, so an
// update about something else must not touch it.
func TestUserResource_Update_unsetSudoNopasswdOmitsTheKey(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, sudoUserEntity("ALL"), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("old"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("new"),
		"password_disabled": tftypes.NewValue(tftypes.Bool, true),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	if _, present := got["sudo_commands_nopasswd"]; present {
		t.Errorf("sudo_commands_nopasswd sent on an update that did not set it: %v", got)
	}
}

// A grant made outside Terraform has to reach state, or it is invisible drift.
func TestUserResource_Read_sudoNopasswdRoundTrips(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := userUpdateRecorder(ctx, t, sudoUserEntity("ALL"), &got)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "username": str("svc"), "full_name": str("svc"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", rResp.Diagnostics)
	}
	var m UserResourceModel
	rResp.State.Get(ctx, &m)
	var cmds []string
	m.SudoNopasswd.ElementsAs(ctx, &cmds, false)
	if len(cmds) != 1 || cmds[0] != "ALL" {
		t.Errorf("sudo_commands_nopasswd = %v, want [ALL]", cmds)
	}
}

// --- truenas_group ---

// groupRecorder captures group.create and group.update bodies.
func groupRecorder(ctx context.Context, t *testing.T, entity map[string]interface{}, got *map[string]interface{}) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "group.create":
			if len(params) > 0 {
				if m, ok := params[0].(map[string]interface{}); ok {
					*got = m
				}
			}
			return entity, nil
		case "group.update":
			if len(params) > 1 {
				if m, ok := params[1].(map[string]interface{}); ok {
					*got = m
				}
			}
			return entity, nil
		case "group.get_instance":
			return entity, nil
		case "group.query":
			return []interface{}{entity}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func sudoGroupEntity(nopasswd ...string) map[string]interface{} {
	return map[string]interface{}{
		"id": 1, "gid": 3000, "name": "automation", "builtin": false, "smb": false,
		"sudo_commands": []string{}, "sudo_commands_nopasswd": nopasswd, "users": []int{},
	}
}

func TestGroupResource_Create_sendsSudoNopasswd(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := groupRecorder(ctx, t, sudoGroupEntity("/usr/bin/qemu-img"), &got)
	r := &GroupResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":                   str("automation"),
		"sudo_commands_nopasswd": strList("/usr/bin/qemu-img"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}
	wireList(t, got, "sudo_commands_nopasswd", "/usr/bin/qemu-img")
}

func TestGroupResource_Update_emptySudoNopasswdClears(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := groupRecorder(ctx, t, sudoGroupEntity(), &got)
	r := &GroupResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
		"sudo_commands_nopasswd": strList("ALL"),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
		"sudo_commands_nopasswd": strList(),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	wireList(t, got, "sudo_commands_nopasswd")
}

func TestGroupResource_Update_unsetSudoNopasswdOmitsTheKey(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := groupRecorder(ctx, t, sudoGroupEntity("ALL"), &got)
	r := &GroupResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
	})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
	})
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}
	if _, present := got["sudo_commands_nopasswd"]; present {
		t.Errorf("sudo_commands_nopasswd sent on an update that did not set it: %v", got)
	}
}

func TestGroupResource_Read_sudoNopasswdRoundTrips(t *testing.T) {
	ctx := context.Background()
	var got map[string]interface{}
	c := groupRecorder(ctx, t, sudoGroupEntity("ALL"), &got)
	r := &GroupResource{client: c}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("1"), "name": str("automation"),
	})
	rResp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", rResp.Diagnostics)
	}
	var m GroupResourceModel
	rResp.State.Get(ctx, &m)
	var cmds []string
	m.SudoNopasswd.ElementsAs(ctx, &cmds, false)
	if len(cmds) != 1 || cmds[0] != "ALL" {
		t.Errorf("sudo_commands_nopasswd = %v, want [ALL]", cmds)
	}
}
