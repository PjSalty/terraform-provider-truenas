package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// truenas_system_update was rewritten onto update.config / update.update after
// issue #32: the five update.* methods it used to call do not exist in
// middleware and never did on any release this provider supports.
//
// These tests assert the new surface, and in particular that the dead methods
// are never emitted again.

var deadUpdateMethods = []string{
	"update.get_auto_download",
	"update.set_auto_download",
	"update.get_trains",
	"update.set_train",
	"update.check_available",
}

type updateRecorder struct {
	methods []string
	body    map[string]interface{}
}

func updateConfigServer(ctx context.Context, t *testing.T, rec *updateRecorder, cfg, status, choices interface{}, failMethod string) *wsclient.Client {
	t.Helper()
	return newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		rec.methods = append(rec.methods, method)
		if method == failMethod {
			return nil, &wsclient.RPCError{Code: wsclient.CodeMethodCallError, Message: "boom"}
		}
		switch method {
		case "update.config":
			return cfg, nil
		case "update.update":
			if len(params) > 0 {
				if m, ok := params[0].(map[string]interface{}); ok {
					rec.body = m
				}
			}
			return cfg, nil
		case "update.profile_choices":
			return choices, nil
		case "update.status":
			return status, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
}

func updateFixtures() (cfg, status, choices map[string]interface{}) {
	cfg = map[string]interface{}{"id": 1, "autocheck": true, "profile": "GENERAL"}
	status = map[string]interface{}{
		"code": "NORMAL",
		"status": map[string]interface{}{
			"current_version": map[string]interface{}{"version": "25.10.4", "profile": "GENERAL"},
			"new_version":     map[string]interface{}{"version": "25.10.6", "profile": "GENERAL"},
		},
		"error": nil,
	}
	choices = map[string]interface{}{
		"GENERAL":          map[string]interface{}{"name": "GENERAL", "footnote": "", "description": "d", "available": true},
		"MISSION_CRITICAL": map[string]interface{}{"name": "MISSION_CRITICAL", "footnote": "", "description": "d", "available": true},
		"DEVELOPER":        map[string]interface{}{"name": "DEVELOPER", "footnote": "", "description": "d", "available": false},
	}
	return
}

func assertNoDeadMethods(t *testing.T, rec *updateRecorder) {
	t.Helper()
	for _, m := range rec.methods {
		for _, dead := range deadUpdateMethods {
			if m == dead {
				t.Errorf("emitted %q, which does not exist in middleware (issue #32)", m)
			}
		}
	}
}

func TestSystemUpdateResource_Read(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	assertNoDeadMethods(t, rec)

	var got SystemUpdateResourceModel
	resp.State.Get(ctx, &got)
	if !got.Autocheck.ValueBool() {
		t.Error("autocheck not read back")
	}
	if got.Profile.ValueString() != "GENERAL" {
		t.Errorf("profile = %q, want GENERAL", got.Profile.ValueString())
	}
	if got.CurrentVersion.ValueString() != "25.10.4" {
		t.Errorf("current_version = %q", got.CurrentVersion.ValueString())
	}
	if got.AvailableVersion.ValueString() != "25.10.6" {
		t.Errorf("available_version = %q", got.AvailableVersion.ValueString())
	}
	if got.Status.ValueString() != "NORMAL" {
		t.Errorf("status = %q", got.Status.ValueString())
	}
}

// status and new_version are nullable. A nil status means "no information",
// not "no update", and must not be dereferenced.
func TestSystemUpdateResource_Read_nullStatus(t *testing.T) {
	ctx := context.Background()
	cfg, _, choices := updateFixtures()
	status := map[string]interface{}{"code": "ERROR", "status": nil, "error": map[string]interface{}{"errname": "ENONET", "reason": "offline"}}
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read with a null status: %v", resp.Diagnostics)
	}
	var got SystemUpdateResourceModel
	resp.State.Get(ctx, &got)
	if got.Status.ValueString() != "ERROR" {
		t.Errorf("status = %q, want ERROR", got.Status.ValueString())
	}
	if got.CurrentVersion.ValueString() != "" || got.AvailableVersion.ValueString() != "" {
		t.Error("version fields must be empty when the server reports no status")
	}
}

// profile is nullable; an unset one must read back known-empty so plans do not
// show a phantom diff.
func TestSystemUpdateResource_Read_nullProfile(t *testing.T) {
	ctx := context.Background()
	_, status, choices := updateFixtures()
	cfg := map[string]interface{}{"id": 1, "autocheck": false, "profile": nil}
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	var got SystemUpdateResourceModel
	resp.State.Get(ctx, &got)
	if got.Profile.IsNull() || got.Profile.IsUnknown() {
		t.Error("a null profile must read back as known-empty")
	}
}

func TestSystemUpdateResource_Create_sendsOneUpdateCall(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"autocheck": tftypes.NewValue(tftypes.Bool, true),
		"profile":   str("MISSION_CRITICAL"),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	assertNoDeadMethods(t, rec)

	if rec.body == nil {
		t.Fatal("update.update was never called")
	}
	if rec.body["autocheck"] != true {
		t.Errorf("autocheck = %v, want true", rec.body["autocheck"])
	}
	if rec.body["profile"] != "MISSION_CRITICAL" {
		t.Errorf("profile = %v", rec.body["profile"])
	}
}

// An unavailable profile is rejected client-side with the valid choices
// listed, rather than failing server-side with an opaque error.
func TestSystemUpdateResource_Create_rejectsUnavailableProfile(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"autocheck": tftypes.NewValue(tftypes.Bool, false),
		"profile":   str("DEVELOPER"), // available: false
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("an unavailable profile was accepted")
	}
	if rec.body != nil {
		t.Error("update.update was called despite the profile being unselectable")
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	if !strings.Contains(detail, "GENERAL") {
		t.Errorf("the diagnostic should list the available profiles, got: %s", detail)
	}
	if strings.Contains(detail, "DEVELOPER,") || strings.Contains(detail, ", DEVELOPER") {
		t.Errorf("an unavailable profile was offered as a choice: %s", detail)
	}
}

func TestSystemUpdateResource_Create_rejectsUnknownProfile(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"autocheck": tftypes.NewValue(tftypes.Bool, false),
		"profile":   str("NOPE"),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a nonexistent profile was accepted")
	}
}

// Omitting profile must leave the server's value alone rather than clearing it.
func TestSystemUpdateResource_Update_omittedProfileNotSent(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":        str("system_update"),
		"autocheck": tftypes.NewValue(tftypes.Bool, false),
	})
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if rec.body == nil {
		t.Fatal("update.update was never called")
	}
	if _, present := rec.body["profile"]; present {
		t.Errorf("profile was sent for a plan that did not set it: %v", rec.body)
	}
	if rec.body["autocheck"] != false {
		t.Errorf("autocheck = %v, want false", rec.body["autocheck"])
	}
}

func TestSystemUpdateResource_apiErrorsSurface(t *testing.T) {
	cases := []string{"update.config", "update.status", "update.update", "update.profile_choices"}
	for _, failMethod := range cases {
		t.Run(failMethod, func(t *testing.T) {
			ctx := context.Background()
			cfg, status, choices := updateFixtures()
			rec := &updateRecorder{}
			r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, failMethod)}
			sch := schemaOf(t, ctx, r)

			plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
				"autocheck": tftypes.NewValue(tftypes.Bool, true),
				"profile":   str("GENERAL"),
			})
			resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
			r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
			if !resp.Diagnostics.HasError() {
				t.Errorf("a failing %s was reported as success", failMethod)
			}
		})
	}
}

// Delete is a no-op: the update config is a singleton and removing the
// resource must not reset the system's update policy.
func TestSystemUpdateResource_Delete_isANoop(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})
	resp := &resource.DeleteResponse{State: st}
	r.Delete(ctx, resource.DeleteRequest{State: st}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Delete: %v", resp.Diagnostics)
	}
	if len(rec.methods) != 0 {
		t.Errorf("Delete called the API: %v", rec.methods)
	}
}

func TestSystemUpdateResource_Schema_hasNewAttributes(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}
	sch := schemaOf(t, ctx, r)
	attrs := sch.Schema.Attributes
	for _, want := range []string{"id", "autocheck", "profile", "status", "current_version", "available_version"} {
		if _, ok := attrs[want]; !ok {
			t.Errorf("schema is missing %q", want)
		}
	}
	for _, gone := range []string{"auto_download", "train", "available_status"} {
		if _, present := attrs[gone]; present {
			t.Errorf("schema still declares %q, which the rewrite replaced", gone)
		}
	}
	if sch.Schema.Version != 1 {
		t.Errorf("schema version = %d, want 1 (the rename needs a state upgrader)", sch.Schema.Version)
	}
}

func TestSystemUpdateResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := &SystemUpdateResource{}
	sch := schemaOf(t, ctx, r)

	// The singleton id is the only accepted value.
	resp := &resource.ImportStateResponse{State: primedStateV2(t, ctx, sch)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: systemUpdateSingletonID}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("importing the singleton id failed: %v", resp.Diagnostics)
	}

	bad := &resource.ImportStateResponse{State: primedStateV2(t, ctx, sch)}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "1"}, bad)
	if !bad.Diagnostics.HasError() {
		t.Error("a non-singleton import id was accepted")
	}
}

// Read and Update must surface a refresh failure rather than writing a
// half-populated model into state.
func TestSystemUpdateResource_ReadAndUpdate_refreshFailures(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()

	for _, failMethod := range []string{"update.config", "update.status"} {
		t.Run(failMethod, func(t *testing.T) {
			rec := &updateRecorder{}
			r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, failMethod)}
			sch := schemaOf(t, ctx, r)
			st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})

			rResp := &resource.ReadResponse{State: st}
			r.Read(ctx, resource.ReadRequest{State: st}, rResp)
			if !rResp.Diagnostics.HasError() {
				t.Errorf("Read succeeded despite %s failing", failMethod)
			}

			plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
				"id": str("system_update"), "autocheck": tftypes.NewValue(tftypes.Bool, true),
			})
			uResp := &resource.UpdateResponse{State: st}
			r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
			if !uResp.Diagnostics.HasError() {
				t.Errorf("Update succeeded despite %s failing", failMethod)
			}
		})
	}
}

// With no profile marked available the diagnostic must say so rather than
// printing an empty list.
func TestSystemUpdateResource_noAvailableProfiles(t *testing.T) {
	ctx := context.Background()
	cfg, status, _ := updateFixtures()
	choices := map[string]interface{}{
		"DEVELOPER": map[string]interface{}{"name": "DEVELOPER", "footnote": "", "description": "d", "available": false},
	}
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "")}
	sch := schemaOf(t, ctx, r)
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"autocheck": tftypes.NewValue(tftypes.Bool, false),
		"profile":   str("DEVELOPER"),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("an unavailable profile was accepted")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "none reported as available") {
		t.Errorf("diagnostic should say no profiles are available: %s", resp.Diagnostics.Errors()[0].Detail())
	}
}

// Update must surface a write failure, distinct from a refresh failure.
func TestSystemUpdateResource_Update_applyFailure(t *testing.T) {
	ctx := context.Background()
	cfg, status, choices := updateFixtures()
	rec := &updateRecorder{}
	r := &SystemUpdateResource{client: updateConfigServer(ctx, t, rec, cfg, status, choices, "update.update")}
	sch := schemaOf(t, ctx, r)

	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("system_update")})
	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id": str("system_update"), "autocheck": tftypes.NewValue(tftypes.Bool, true),
	})
	resp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("a failed update.update was reported as success")
	}
}

// truenas_pool forwards encryption_options_json straight into a strict
// middleware submodel, so a key that the connected server rejects must be
// refused before the create rather than failing mid-pool.
func TestPoolResource_Create_rejectsRemovedEncryptionKey(t *testing.T) {
	ctx := context.Background()
	c := newWSTestClient(ctx, t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		if method == "system.info" {
			return map[string]interface{}{"version": "26.0.0-BETA.2"}, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	r := &PoolResource{client: c}
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":                    str("tank"),
		"topology_json":           str(`{"data":[{"type":"STRIPE","disks":["sda"]}]}`),
		"encryption":              tftypes.NewValue(tftypes.Bool, true),
		"encryption_options_json": str(`{"algorithm":"AES-256-GCM"}`),
	})
	resp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("a 26.0-removed encryption key was forwarded to middleware")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "algorithm") {
		t.Errorf("diagnostic should name the offending key: %s", resp.Diagnostics.Errors()[0].Detail())
	}
}
