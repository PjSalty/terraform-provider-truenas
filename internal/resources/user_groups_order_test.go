package resources

// Regression test for issue #19: truenas_user failed with multiple groups
// when the server returned the auxiliary group IDs in a different order
// than the config. With groups declared as a List, Create wrote the
// server-ordered list into state; terraform core then rejected the apply
// with "Provider produced inconsistent result after apply" because the
// applied list order differed from the planned order. With groups as a
// Set, element order is irrelevant and the state matches the plan
// semantically.
//
// The test builds the groups plan value from the schema's declared
// terraform type, so it keeps compiling if the attribute type changes,
// and asserts the consistency semantics terraform core applies for the
// schema's collection kind. A regression back to List makes it fail on
// the mock server's reversed group order below.

import (
	"context"
	"math/big"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// groupsFromRaw extracts the "groups" attribute of a state/plan raw object
// value as a []int64 in stored order.
func groupsFromRaw(t *testing.T, raw tftypes.Value) []int64 {
	t.Helper()
	var obj map[string]tftypes.Value
	if err := raw.As(&obj); err != nil {
		t.Fatalf("raw.As(object): %v", err)
	}
	gv, ok := obj["groups"]
	if !ok {
		t.Fatalf("no groups attribute in raw value")
	}
	if gv.IsNull() {
		t.Fatalf("groups attribute is null")
	}
	var elems []tftypes.Value
	if err := gv.As(&elems); err != nil {
		t.Fatalf("groups.As(elements): %v", err)
	}
	out := make([]int64, 0, len(elems))
	for _, e := range elems {
		var f big.Float
		if err := e.As(&f); err != nil {
			t.Fatalf("group element As(big.Float): %v", err)
		}
		i, _ := f.Int64()
		out = append(out, i)
	}
	return out
}

func int64sEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sortedCopy(a []int64) []int64 {
	c := append([]int64(nil), a...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	return c
}

// assertGroupsConsistent applies terraform core's apply-consistency
// semantics: lists must match the plan element-for-element in order,
// sets only need to contain the same elements.
func assertGroupsConsistent(t *testing.T, phase string, isSet bool, planned, got []int64) {
	t.Helper()
	if isSet {
		if !int64sEqual(sortedCopy(planned), sortedCopy(got)) {
			t.Errorf("%s: state groups %v do not semantically match planned groups %v (set semantics)", phase, got, planned)
		}
		return
	}
	if !int64sEqual(planned, got) {
		t.Errorf("%s: state groups %v differ from planned groups %v in order; terraform core fails this apply with %q (issue #19)",
			phase, got, planned, "Provider produced inconsistent result after apply")
	}
}

func TestUserResource_GroupsOrderDrift_Issue19(t *testing.T) {
	ctx := context.Background()

	plannedGroups := []int64{101, 102}

	// The mock server echoes the user with groups REVERSED relative to
	// the plan order, which is what TrueNAS does in the wild (issue #19).
	body := map[string]interface{}{
		"id": 1, "uid": 1000, "username": "anton", "full_name": "Anton",
		"email": "", "home": "/var/empty", "shell": "/usr/sbin/nologin",
		"locked": false, "smb": false,
		"group":             map[string]interface{}{"id": 100, "bsdgrp_gid": 100},
		"groups":            []interface{}{102, 101}, // reversed vs plan
		"sudo_commands":     []interface{}{},
		"sshpubkey":         "",
		"password_disabled": false,
	}
	c := newWSJSONServerClient(t, body)
	r := &UserResource{client: c}
	sch := schemaOf(t, ctx, r)

	// Build the groups value with the schema-declared collection type so
	// this test runs unchanged against both List (main) and Set (PR #20).
	objType, ok := sch.Schema.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("schema terraform type is not an object")
	}
	gType, ok := objType.AttributeTypes["groups"]
	if !ok {
		t.Fatalf("schema has no groups attribute")
	}
	_, isSet := gType.(tftypes.Set)

	items := make([]tftypes.Value, len(plannedGroups))
	for i, g := range plannedGroups {
		items[i] = tftypes.NewValue(tftypes.Number, g)
	}
	groupsVal := tftypes.NewValue(gType, items)

	planVals := map[string]tftypes.Value{
		"username":  str("anton"),
		"full_name": str("Anton"),
		"password":  str("hunter2"),
		"uid":       num(1000),
		"group":     num(100),
		"groups":    groupsVal,
	}

	// --- Create ---
	plan := planFromValues(t, ctx, sch, planVals)
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create diagnostics: %v", cResp.Diagnostics)
	}
	assertGroupsConsistent(t, "Create", isSet, plannedGroups, groupsFromRaw(t, cResp.State.Raw))

	// --- Read (refresh with plan-ordered groups already in state) ---
	stateVals := map[string]tftypes.Value{
		"id":     str("1"),
		"groups": groupsVal,
	}
	readState := stateFromValues(t, ctx, sch, stateVals)
	rResp := &resource.ReadResponse{State: readState}
	r.Read(ctx, resource.ReadRequest{State: readState}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", rResp.Diagnostics)
	}
	assertGroupsConsistent(t, "Read", isSet, plannedGroups, groupsFromRaw(t, rResp.State.Raw))
}
