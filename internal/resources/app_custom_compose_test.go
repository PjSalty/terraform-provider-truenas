package resources

// Unit tests for custom Docker Compose app support (issue #24).
//
// The recording mock asserts the WIRE CONTRACT of AppResource against
// app.create / app.update: custom installs send custom_app plus the
// raw compose string and no catalog fields, catalog installs stay
// byte-for-byte unchanged, and the config-authoritative compose string
// survives every refetch (the middleware stores the parsed compose,
// the original string cannot round-trip).

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/customtypes"
	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

const (
	testComposeOne = "services:\n  app:\n    image: busybox:1.36\n    command: [\"sleep\", \"infinity\"]\n"
	testComposeTwo = "services:\n  app:\n    image: busybox:1.36\n    command: [\"sleep\", \"infinity\"]\n    environment:\n      DEMO_ENV: two\n"
)

// appWireRecorder captures the request objects app.create, app.update,
// and app.delete receive so tests can assert exact wire payloads.
type appWireRecorder struct {
	mu     sync.Mutex
	create []map[string]interface{}
	update []map[string]interface{}
	del    []map[string]interface{}
}

func appComposeBody(custom bool) map[string]interface{} {
	return map[string]interface{}{
		"id": "sleeper", "name": "sleeper", "state": "RUNNING",
		"upgrade_available": false, "human_version": "1.0.0",
		"version": "1.0.0", "custom_app": custom,
	}
}

// appComposeCfgOne is the server-side parsed document for
// testComposeOne, deliberately not key-ordered like the string.
func appComposeCfgOne() map[string]interface{} {
	return map[string]interface{}{
		"services": map[string]interface{}{
			"app": map[string]interface{}{
				"command": []interface{}{"sleep", "infinity"},
				"image":   "busybox:1.36",
			},
		},
	}
}

// appComposeCfgTwo is the server-side parsed document for
// testComposeTwo.
func appComposeCfgTwo() map[string]interface{} {
	return map[string]interface{}{
		"services": map[string]interface{}{
			"app": map[string]interface{}{
				"command":     []interface{}{"sleep", "infinity"},
				"environment": map[string]interface{}{"DEMO_ENV": "two"},
				"image":       "busybox:1.36",
			},
		},
	}
}

// appCfgNotFound is a sentinel cfg making the mock's app.config fail
// with an ENOENT-shaped error (IsNotFound true).
var appCfgNotFound = map[string]interface{}{"__notfound__": true}

// newAppComposeTestClient wires a recording job-aware mock: app.create
// and app.update return a job id, core.get_jobs resolves SUCCESS, the
// follow-up app.get_instance answers with body, and app.config serves
// cfg. nil cfg makes app.config fail hard (internal error), the
// appCfgNotFound sentinel makes it fail ENOENT.
func newAppComposeTestClient(t *testing.T, body, cfg map[string]interface{}) (*wsclient.Client, *appWireRecorder) {
	t.Helper()
	rec := &appWireRecorder{}
	const jobID = int64(7)
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		switch method {
		case "app.create":
			if len(params) > 0 {
				if m, ok := params[0].(map[string]interface{}); ok {
					rec.mu.Lock()
					rec.create = append(rec.create, m)
					rec.mu.Unlock()
				}
			}
			return jobID, nil
		case "app.update":
			if len(params) > 1 {
				if m, ok := params[1].(map[string]interface{}); ok {
					rec.mu.Lock()
					rec.update = append(rec.update, m)
					rec.mu.Unlock()
				}
			}
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		case "app.get_instance":
			return body, nil
		case "app.config":
			if cfg == nil {
				return nil, &wsclient.RPCError{Code: wsclient.CodeInternalError, Message: "compose config unavailable"}
			}
			if _, gone := cfg["__notfound__"]; gone {
				return nil, &wsclient.RPCError{
					Code:    wsclient.CodeMethodCallError,
					Message: "Method call error",
					Data:    json.RawMessage(`{"errname":"ENOENT","reason":"[ENOENT] app sleeper does not exist"}`),
				}
			}
			return cfg, nil
		case "app.delete":
			if len(params) > 1 {
				if m, ok := params[1].(map[string]interface{}); ok {
					rec.mu.Lock()
					rec.del = append(rec.del, m)
					rec.mu.Unlock()
				}
			}
			return jobID, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(context.Background())
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c, rec
}

// TestAppCustomCompose_CreateSendsCustomApp: Create with custom_compose
// set must send custom_app true plus the raw compose string and must
// NOT put any catalog field on the wire. The planned compose string
// must land in state verbatim, and Create must never call app.config
// (nil cfg would fail the test if it did): plan wins on apply, drift
// detection is Read's job.
func TestAppCustomCompose_CreateSendsCustomApp(t *testing.T) {
	c, rec := newAppComposeTestClient(t, appComposeBody(true), nil)
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create(custom) diagnostics: %v", cResp.Diagnostics)
	}
	if len(rec.create) != 1 {
		t.Fatalf("want 1 app.create call, got %d", len(rec.create))
	}
	payload := rec.create[0]
	if payload["custom_app"] != true {
		t.Errorf("custom_app: got %v", payload["custom_app"])
	}
	if payload["custom_compose_config_string"] != testComposeOne {
		t.Errorf("custom_compose_config_string: got %v", payload["custom_compose_config_string"])
	}
	for _, k := range []string{"catalog_app", "train", "version", "values"} {
		if v, present := payload[k]; present {
			t.Errorf("catalog field %q must not be sent for custom installs, got %v", k, v)
		}
	}

	var got AppResourceModel
	if d := cResp.State.Get(ctx, &got); d.HasError() {
		t.Fatalf("state get: %v", d)
	}
	if got.CustomCompose.ValueString() != testComposeOne {
		t.Errorf("state custom_compose clobbered by refetch: %q", got.CustomCompose.ValueString())
	}
}

// TestAppCustomCompose_CreateCatalogUnchanged: a catalog install keeps
// the pre-#24 wire shape, no custom_app and no compose string leak in.
func TestAppCustomCompose_CreateCatalogUnchanged(t *testing.T) {
	// nil cfg: app.config must never be called on the catalog path
	c, rec := newAppComposeTestClient(t, appComposeBody(false), nil)
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"app_name":    str("sleeper"),
		"catalog_app": str("minio"),
		"train":       str("stable"),
		"version":     str("1.0.0"),
		"values":      str(`{"x":1}`),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create(catalog) diagnostics: %v", cResp.Diagnostics)
	}
	if len(rec.create) != 1 {
		t.Fatalf("want 1 app.create call, got %d", len(rec.create))
	}
	payload := rec.create[0]
	if payload["catalog_app"] != "minio" || payload["train"] != "stable" || payload["version"] != "1.0.0" {
		t.Errorf("catalog fields wrong: %v", payload)
	}
	if _, present := payload["values"]; !present {
		t.Errorf("values missing from catalog install payload: %v", payload)
	}
	for _, k := range []string{"custom_app", "custom_compose_config_string"} {
		if v, present := payload[k]; present {
			t.Errorf("custom field %q leaked into catalog install, got %v", k, v)
		}
	}
}

// TestAppCustomCompose_UpdateSendsComposeString: a compose content
// edit goes out as custom_compose_config_string on app.update (in
// place), values stays off the wire, and the planned string lands in
// state verbatim; Update never calls app.config (nil cfg would fail
// the test if it did).
func TestAppCustomCompose_UpdateSendsComposeString(t *testing.T) {
	c, rec := newAppComposeTestClient(t, appComposeBody(true), nil)
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	stateVals := map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	}
	planVals := map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeTwo),
	}
	state := stateFromValues(t, ctx, sch, stateVals)
	plan := planFromValues(t, ctx, sch, planVals)
	uResp := &resource.UpdateResponse{State: state}
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update(custom) diagnostics: %v", uResp.Diagnostics)
	}
	if len(rec.update) != 1 {
		t.Fatalf("want 1 app.update call, got %d", len(rec.update))
	}
	payload := rec.update[0]
	if payload["custom_compose_config_string"] != testComposeTwo {
		t.Errorf("custom_compose_config_string: got %v", payload["custom_compose_config_string"])
	}
	if v, present := payload["values"]; present {
		t.Errorf("values must not be sent for custom compose updates, got %v", v)
	}

	var got AppResourceModel
	if d := uResp.State.Get(ctx, &got); d.HasError() {
		t.Fatalf("state get: %v", d)
	}
	if got.CustomCompose.ValueString() != testComposeTwo {
		t.Errorf("state custom_compose clobbered by refetch: %q", got.CustomCompose.ValueString())
	}
}

// TestAppCustomCompose_MapResponsePreservesCompose is the plan-wins
// core: mapResponseToModel must never touch CustomCompose, whatever
// the server response looks like.
func TestAppCustomCompose_MapResponsePreservesCompose(t *testing.T) {
	r := &AppResource{}
	m := AppResourceModel{CustomCompose: customtypes.NewNormalizedYAMLValue(testComposeOne)}
	app := &truenas.App{
		ID: "sleeper", Name: "sleeper", State: "RUNNING",
		HumanVersion: "1.0.0", Version: "1.0.0", CustomApp: true,
	}
	r.mapResponseToModel(app, &m)
	if m.CustomCompose.ValueString() != testComposeOne {
		t.Errorf("mapResponseToModel touched custom_compose: %q", m.CustomCompose.ValueString())
	}
}

// TestAppCustomCompose_ReadClearsImportedCatalogSeed: after importing
// a custom app the state carries the catalog_app "" seed and a null
// custom_compose. The first Read must clear the seed to null AND
// populate custom_compose with a canonical dump of the server's
// stored compose so a structurally matching config plans clean.
func TestAppCustomCompose_ReadClearsImportedCatalogSeed(t *testing.T) {
	c, _ := newAppComposeTestClient(t, appComposeBody(true), appComposeCfgOne())
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":          str("sleeper"),
		"app_name":    str("sleeper"),
		"catalog_app": str(""),
	})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", rResp.Diagnostics)
	}
	var got AppResourceModel
	if d := rResp.State.Get(ctx, &got); d.HasError() {
		t.Fatalf("state get: %v", d)
	}
	if !got.CatalogApp.IsNull() {
		t.Errorf("catalog_app import seed not cleared for custom app: %q", got.CatalogApp.ValueString())
	}
	if got.CustomCompose.IsNull() {
		t.Fatal("custom_compose not populated from server compose after import")
	}
	eq, err := customtypes.YAMLStringsSemanticallyEqual(got.CustomCompose.ValueString(), testComposeOne)
	if err != nil || !eq {
		t.Errorf("populated compose not semantically equal to server doc (eq=%v err=%v): %q",
			eq, err, got.CustomCompose.ValueString())
	}
}

// TestAppCustomCompose_ReadKeepsCatalogSlugForCatalogApps: catalog
// installs keep their slug through Read, the seed-clearing branch is
// gated on the server's custom_app flag.
func TestAppCustomCompose_ReadKeepsCatalogSlugForCatalogApps(t *testing.T) {
	// nil cfg: app.config must never be called for catalog apps
	c, _ := newAppComposeTestClient(t, appComposeBody(false), nil)
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":          str("sleeper"),
		"app_name":    str("sleeper"),
		"catalog_app": str("minio"),
	})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", rResp.Diagnostics)
	}
	var got AppResourceModel
	if d := rResp.State.Get(ctx, &got); d.HasError() {
		t.Fatalf("state get: %v", d)
	}
	if got.CatalogApp.ValueString() != "minio" {
		t.Errorf("catalog_app changed on Read: %q", got.CatalogApp.ValueString())
	}
}

// TestAppCustomCompose_ReadPreservesComposeString: when the server's
// stored compose is structurally equal (the mock cfg carries different
// key order than the string), Read keeps the user's string byte for
// byte, formatting is never drift.
func TestAppCustomCompose_ReadPreservesComposeString(t *testing.T) {
	c, _ := newAppComposeTestClient(t, appComposeBody(true), appComposeCfgOne())
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", rResp.Diagnostics)
	}
	var got AppResourceModel
	if d := rResp.State.Get(ctx, &got); d.HasError() {
		t.Fatalf("state get: %v", d)
	}
	if got.CustomCompose.ValueString() != testComposeOne {
		t.Errorf("custom_compose changed on Read: %q", got.CustomCompose.ValueString())
	}
}

// TestAppCustomCompose_ReadSurfacesSemanticDrift: when the server's
// stored compose STRUCTURALLY differs from state, Read writes a
// canonical dump of the server document so the next plan shows the
// drift.
func TestAppCustomCompose_ReadSurfacesSemanticDrift(t *testing.T) {
	c, _ := newAppComposeTestClient(t, appComposeBody(true), appComposeCfgTwo())
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read diagnostics: %v", rResp.Diagnostics)
	}
	var got AppResourceModel
	if d := rResp.State.Get(ctx, &got); d.HasError() {
		t.Fatalf("state get: %v", d)
	}
	if got.CustomCompose.ValueString() == testComposeOne {
		t.Fatal("semantic drift not surfaced, state still holds the stale compose string")
	}
	eq, err := customtypes.YAMLStringsSemanticallyEqual(got.CustomCompose.ValueString(), testComposeTwo)
	if err != nil || !eq {
		t.Errorf("drifted state not semantically equal to server doc (eq=%v err=%v): %q",
			eq, err, got.CustomCompose.ValueString())
	}
}

// TestAppCustomCompose_ReadComposeConfigNotFound: an ENOENT from
// app.config means the app vanished mid-read; the resource follows
// the normal Read remove-from-state flow instead of hard-erroring.
func TestAppCustomCompose_ReadComposeConfigNotFound(t *testing.T) {
	c, _ := newAppComposeTestClient(t, appComposeBody(true), appCfgNotFound)
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("not-found must not hard-error: %v", rResp.Diagnostics)
	}
	if !rResp.State.Raw.IsNull() {
		t.Error("resource not removed from state after app.config ENOENT")
	}
}

// TestAppApplyComposeDrift_EmptyServerConfig: an empty server dict
// cannot be expressed as a valid custom_compose, the state value
// stays untouched (string or null).
func TestAppApplyComposeDrift_EmptyServerConfig(t *testing.T) {
	r := &AppResource{}
	var diags diag.Diagnostics

	withString := &AppResourceModel{
		ID:            types.StringValue("sleeper"),
		CustomCompose: customtypes.NewNormalizedYAMLValue(testComposeOne),
	}
	r.applyComposeDrift(withString, map[string]interface{}{}, &diags)
	if diags.HasError() || withString.CustomCompose.ValueString() != testComposeOne {
		t.Errorf("empty cfg must keep the state string (diags=%v, got %q)",
			diags, withString.CustomCompose.ValueString())
	}

	withNull := &AppResourceModel{
		ID:            types.StringValue("sleeper"),
		CustomCompose: customtypes.NewNormalizedYAMLNull(),
	}
	r.applyComposeDrift(withNull, nil, &diags)
	if diags.HasError() || !withNull.CustomCompose.IsNull() {
		t.Errorf("empty cfg must keep null state (diags=%v)", diags)
	}
}

// TestAppCustomCompose_ReadComposeConfigError: app.config failing for
// a custom app must surface as a Read error, never be swallowed.
func TestAppCustomCompose_ReadComposeConfigError(t *testing.T) {
	c, _ := newAppComposeTestClient(t, appComposeBody(true), nil)
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if !rResp.Diagnostics.HasError() {
		t.Fatal("expected Read to fail when app.config fails")
	}
}

// TestAppApplyComposeDrift_RenderError covers the canonical-dump
// failure branch with a document yaml.Marshal cannot encode.
func TestAppApplyComposeDrift_RenderError(t *testing.T) {
	r := &AppResource{}
	model := &AppResourceModel{
		ID:            types.StringValue("sleeper"),
		CustomCompose: customtypes.NewNormalizedYAMLValue(testComposeOne),
	}
	var diags diag.Diagnostics
	r.applyComposeDrift(model, map[string]interface{}{"bad": func() {}}, &diags)
	if !diags.HasError() {
		t.Fatal("expected render error diagnostics")
	}
	if model.CustomCompose.ValueString() != testComposeOne {
		t.Error("model compose must stay untouched on render failure")
	}
}

// TestCanonicalComposeYAML_MarshalerError covers the non-panic error
// return of yaml.Marshal via a failing yaml.Marshaler.
func TestCanonicalComposeYAML_MarshalerError(t *testing.T) {
	if _, err := canonicalComposeYAML(map[string]interface{}{"bad": failingYAMLMarshaler{}}); err == nil {
		t.Fatal("expected error from failing yaml.Marshaler")
	}
}

type failingYAMLMarshaler struct{}

func (failingYAMLMarshaler) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("marshaler boom")
}

// TestAppCustomCompose_DeleteForce: custom apps delete with
// force_remove_custom_app true so a broken compose cannot wedge the
// destroy; catalog apps never send the flag.
func TestAppCustomCompose_DeleteForce(t *testing.T) {
	ctx := context.Background()

	// custom app: flag on the wire
	c, rec := newAppComposeTestClient(t, appComposeBody(true), appComposeCfgOne())
	r := &AppResource{client: c}
	sch := schemaOf(t, ctx, r)
	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Fatalf("Delete(custom) diagnostics: %v", dResp.Diagnostics)
	}
	if len(rec.del) != 1 || rec.del[0]["force_remove_custom_app"] != true {
		t.Errorf("custom delete options wrong: %v", rec.del)
	}

	// catalog app: flag absent (omitempty)
	c2, rec2 := newAppComposeTestClient(t, appComposeBody(false), nil)
	r2 := &AppResource{client: c2}
	state2 := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":          str("sleeper"),
		"app_name":    str("sleeper"),
		"catalog_app": str("minio"),
	})
	dResp2 := &resource.DeleteResponse{State: state2}
	r2.Delete(ctx, resource.DeleteRequest{State: state2}, dResp2)
	if dResp2.Diagnostics.HasError() {
		t.Fatalf("Delete(catalog) diagnostics: %v", dResp2.Diagnostics)
	}
	if len(rec2.del) != 1 {
		t.Fatalf("want 1 app.delete call, got %d", len(rec2.del))
	}
	if v, present := rec2.del[0]["force_remove_custom_app"]; present {
		t.Errorf("catalog delete must not carry force_remove_custom_app, got %v", v)
	}
}

// TestAppComposeKindFlip pins the replacement matrix of the
// custom_compose plan modifier: content edits and imported-custom-app
// first plans stay in place, kind conversions replace.
func TestAppComposeKindFlip(t *testing.T) {
	ctx := context.Background()
	r := &AppResource{}
	sch := schemaOf(t, ctx, r)
	nullStr := tftypes.NewValue(tftypes.String, nil)

	cases := []struct {
		name        string
		stateVal    types.String
		planVal     types.String
		catalog     tftypes.Value
		wantReplace bool
	}{
		{"content edit in place", types.StringValue("a: 1\n"), types.StringValue("a: 2\n"), nullStr, false},
		{"both null no-op", types.StringNull(), types.StringNull(), str("plex"), false},
		{"custom to catalog replaces", types.StringValue("a: 1\n"), types.StringNull(), nullStr, true},
		{"catalog to custom replaces", types.StringNull(), types.StringValue("a: 1\n"), str("plex"), true},
		{"imported custom empty seed in place", types.StringNull(), types.StringValue("a: 1\n"), str(""), false},
		{"imported custom null catalog in place", types.StringNull(), types.StringValue("a: 1\n"), nullStr, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			composeState := nullStr
			if !tc.stateVal.IsNull() {
				composeState = str(tc.stateVal.ValueString())
			}
			req := planmodifier.StringRequest{
				Path:       path.Root("custom_compose"),
				StateValue: tc.stateVal,
				PlanValue:  tc.planVal,
				State: stateFromValues(t, ctx, sch, map[string]tftypes.Value{
					"catalog_app":    tc.catalog,
					"custom_compose": composeState,
				}),
			}
			resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
			appComposeKindFlip(ctx, req, resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if resp.RequiresReplace != tc.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.wantReplace)
			}
		})
	}
}

// TestAppComposeKindFlip_StateReadError: a state whose schema lacks
// catalog_app makes the modifier's GetAttribute fail; the diagnostic
// must surface and replacement must NOT be forced on bad data.
func TestAppComposeKindFlip_StateReadError(t *testing.T) {
	ctx := context.Background()
	minSchema := schema.Schema{Attributes: map[string]schema.Attribute{
		"id": schema.StringAttribute{Computed: true},
	}}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}},
		map[string]tftypes.Value{"id": str("x")},
	)
	req := planmodifier.StringRequest{
		Path:       path.Root("custom_compose"),
		StateValue: types.StringNull(),
		PlanValue:  types.StringValue("a: 1\n"),
		State:      tfsdk.State{Schema: minSchema, Raw: raw},
	}
	resp := &stringplanmodifier.RequiresReplaceIfFuncResponse{}
	appComposeKindFlip(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics when catalog_app is missing from the state schema")
	}
	if resp.RequiresReplace {
		t.Error("must not force replace when the state read fails")
	}
}

// TestAppConfigValidators drives every AppResource ConfigValidator
// against the catalog/custom config matrix: exactly one of
// catalog_app / custom_compose, and no catalog-only knob combined
// with custom_compose.
func TestAppConfigValidators(t *testing.T) {
	ctx := context.Background()
	r := &AppResource{}
	sch := schemaOf(t, ctx, r)
	compose := str(testComposeOne)

	cases := []struct {
		name    string
		vals    map[string]tftypes.Value
		wantErr bool
	}{
		{"catalog only ok", map[string]tftypes.Value{"catalog_app": str("minio")}, false},
		{"custom only ok", map[string]tftypes.Value{"custom_compose": compose}, false},
		{"both fails", map[string]tftypes.Value{"catalog_app": str("minio"), "custom_compose": compose}, true},
		{"neither fails", map[string]tftypes.Value{}, true},
		{"compose plus train fails", map[string]tftypes.Value{"custom_compose": compose, "train": str("stable")}, true},
		{"compose plus version fails", map[string]tftypes.Value{"custom_compose": compose, "version": str("1.0.0")}, true},
		{"compose plus values fails", map[string]tftypes.Value{"custom_compose": compose, "values": str("{}")}, true},
		{"catalog plus train ok", map[string]tftypes.Value{"catalog_app": str("minio"), "train": str("stable")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vals := map[string]tftypes.Value{"app_name": str("demo")}
			for k, v := range tc.vals {
				vals[k] = v
			}
			cfg := tfsdk.Config{Schema: sch.Schema, Raw: rawFromValues(t, ctx, sch, vals)}
			hasErr := false
			summaries := []string{}
			for _, cv := range r.ConfigValidators(ctx) {
				resp := &resource.ValidateConfigResponse{}
				cv.ValidateResource(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
				if resp.Diagnostics.HasError() {
					hasErr = true
					for _, d := range resp.Diagnostics.Errors() {
						summaries = append(summaries, d.Summary())
					}
				}
			}
			if hasErr != tc.wantErr {
				t.Fatalf("hasErr = %v, want %v (summaries: %v)", hasErr, tc.wantErr, summaries)
			}
			if tc.wantErr {
				// framework convention: too many attributes set reports
				// "Invalid Attribute Combination", none set reports
				// "Missing Attribute Configuration"
				found := false
				for _, s := range summaries {
					if s == "Invalid Attribute Combination" || s == "Missing Attribute Configuration" {
						found = true
					}
				}
				if !found {
					t.Errorf("expected a canonical framework validator summary, got: %v", summaries)
				}
			}
		})
	}
}

// the marshal-error branch of applyComposeDrift, and Read's guard behind
// it, cannot fire through JSON-shaped configs from the real client; the
// renderComposeCanonical seam makes the defensive path testable.
func TestAppReadComposeRenderError(t *testing.T) {
	orig := renderComposeCanonical
	renderComposeCanonical = func(map[string]interface{}) (string, error) {
		return "", fmt.Errorf("injected render failure")
	}
	t.Cleanup(func() { renderComposeCanonical = orig })

	c, _ := newAppComposeTestClient(t, appComposeBody(true), appComposeCfgOne())
	r := &AppResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)
	st := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":             str("sleeper"),
		"app_name":       str("sleeper"),
		"custom_compose": str(testComposeOne),
	})
	resp := &resource.ReadResponse{State: st}
	r.Read(ctx, resource.ReadRequest{State: st}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("Read should surface the compose render failure as a diagnostic")
	}
}
