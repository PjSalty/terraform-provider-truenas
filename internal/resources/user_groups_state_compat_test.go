package resources_test

// State-compat regression tests for the groups List -> Set change
// (issue #19). Provider v2.1.0 and earlier wrote groups as a JSON array
// produced by a List attribute at schema version 0. The Set attribute
// keeps schema version 0, so terraform hands old state to the provider
// through a same-version UpgradeResourceState call that re-decodes the
// raw JSON against the current schema. These tests feed real v2.1.0-era
// version-0 state through the protocol-6 server and prove the decode is
// clean: zero diagnostics, no schema version bump required.

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/provider"
)

// v210UserStateJSON renders a version-0 truenas_user state body the way
// provider v2.1.0 wrote it, with the given groups JSON array literal.
func v210UserStateJSON(groups string) string {
	return fmt.Sprintf(`{
		"id": "17",
		"uid": 3001,
		"username": "svc_backup",
		"full_name": "Backup Service",
		"email": "",
		"password": "hunter2",
		"group": 100,
		"group_create": false,
		"groups": %s,
		"home": "/var/empty",
		"shell": "/usr/sbin/nologin",
		"locked": false,
		"smb": false,
		"sshpubkey": "",
		"sudo_commands": [],
		"timeouts": null
	}`, groups)
}

// upgradeV0UserState pushes raw version-0 state JSON through the real
// protocol-6 UpgradeResourceState path, asserts zero diagnostics, and
// returns the groups decoded from the upgraded state.
func upgradeV0UserState(t *testing.T, rawJSON string) []int64 {
	t.Helper()
	ctx := context.Background()

	srv := providerserver.NewProtocol6(provider.New("test")())()

	schemaResp, err := srv.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatalf("GetProviderSchema: %v", err)
	}
	userSchema, ok := schemaResp.ResourceSchemas["truenas_user"]
	if !ok {
		t.Fatalf("provider does not declare truenas_user")
	}

	resp, err := srv.UpgradeResourceState(ctx, &tfprotov6.UpgradeResourceStateRequest{
		TypeName: "truenas_user",
		Version:  0,
		RawState: &tfprotov6.RawState{JSON: []byte(rawJSON)},
	})
	if err != nil {
		t.Fatalf("UpgradeResourceState: %v", err)
	}
	for _, d := range resp.Diagnostics {
		t.Errorf("unexpected diagnostic: [%s] %s: %s", d.Severity, d.Summary, d.Detail)
	}
	if t.Failed() {
		t.FailNow()
	}
	if resp.UpgradedState == nil {
		t.Fatalf("UpgradeResourceState returned nil state")
	}

	val, err := resp.UpgradedState.Unmarshal(userSchema.ValueType())
	if err != nil {
		t.Fatalf("unmarshal upgraded state: %v", err)
	}

	var obj map[string]tftypes.Value
	if err := val.As(&obj); err != nil {
		t.Fatalf("state.As(object): %v", err)
	}
	var elems []tftypes.Value
	if err := obj["groups"].As(&elems); err != nil {
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

// TestUserV0State_GroupsListDecodesIntoSet proves a v2.1.0 state where
// groups was written in server order ([30,10,20]) loads under the Set
// schema with zero diagnostics and keeps all three IDs.
func TestUserV0State_GroupsListDecodesIntoSet(t *testing.T) {
	got := upgradeV0UserState(t, v210UserStateJSON("[30, 10, 20]"))

	want := map[int64]bool{10: true, 20: true, 30: true}
	if len(got) != len(want) {
		t.Fatalf("groups after upgrade = %v, want the 3 IDs 10, 20, 30", got)
	}
	for _, g := range got {
		if !want[g] {
			t.Fatalf("groups after upgrade = %v, contains unexpected ID %d", got, g)
		}
	}
}

// TestUserV0State_GroupsDuplicatesDecode documents the edge a List could
// technically hold but a Set cannot: duplicate IDs. Such state decodes
// with zero diagnostics instead of erroring the plan. The wire value out
// of UpgradeResourceState still carries the duplicate; terraform core
// collapses it when it builds the cty set, so users see a silent dedupe.
func TestUserV0State_GroupsDuplicatesDecode(t *testing.T) {
	got := upgradeV0UserState(t, v210UserStateJSON("[10, 10, 20]"))

	distinct := map[int64]bool{}
	for _, g := range got {
		if g != 10 && g != 20 {
			t.Fatalf("groups after upgrade = %v, contains unexpected ID %d", got, g)
		}
		distinct[g] = true
	}
	if len(distinct) != 2 {
		t.Fatalf("groups after upgrade = %v, want IDs 10 and 20 present", got)
	}
	t.Logf("duplicate-bearing v0 state decoded to %v (terraform core dedupes when building the set)", got)
}
