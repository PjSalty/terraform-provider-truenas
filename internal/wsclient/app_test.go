package wsclient

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// =============================================================================
// ListApps
// =============================================================================

func TestListApps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "app.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{"id": "jellyfin", "name": "jellyfin", "state": "RUNNING"},
			map[string]interface{}{"id": "vaultwarden", "name": "vaultwarden", "state": "STOPPED"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	apps, err := c.ListApps(ctx)
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(apps) != 2 || apps[0].ID != "jellyfin" {
		t.Errorf("got %+v", apps)
	}
}

func TestListApps_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListApps(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing apps") {
		t.Errorf("got %v", err)
	}
}

func TestListApps_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListApps(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// GetApp
// =============================================================================

func TestGetApp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "app.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": "jellyfin", "name": "jellyfin", "state": "RUNNING",
			"version": "10.9.4", "human_version": "10.9.4_1.2.3",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	app, err := c.GetApp(ctx, "jellyfin")
	if err != nil {
		t.Fatalf("GetApp: %v", err)
	}
	if app.Version != "10.9.4" || app.State != "RUNNING" {
		t.Errorf("got %+v", app)
	}
}

func TestGetApp_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetApp(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "getting app") {
		t.Errorf("got %v", err)
	}
}

func TestGetApp_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetApp(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// Create / Update / Delete
// =============================================================================

// appJobServer fakes a job-bound app.* method that returns a job ID,
// then either resolves SUCCESS (and answers the follow-up
// app.get_instance with placedApp) or FAILED with jobError.
func appJobServer(t *testing.T, expectMethod string, jobError string, placedApp interface{}) *TestServer {
	t.Helper()
	pollCount := atomic.Int64{}
	const jobID = int64(50)
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case expectMethod:
			return jobID, nil
		case "core.get_jobs":
			pollCount.Add(1)
			state := "SUCCESS"
			if jobError != "" {
				state = "FAILED"
			}
			return []interface{}{map[string]interface{}{
				"id": jobID, "state": state, "error": jobError, "result": nil,
			}}, nil
		case "app.get_instance":
			return placedApp, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
}

func TestCreateApp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := appJobServer(t, "app.create", "",
		map[string]interface{}{"id": "jellyfin", "name": "jellyfin", "state": "RUNNING"})
	c, _ := ts.NewClient(ctx)
	app, err := c.CreateApp(ctx, &types.AppCreateRequest{
		AppName: "jellyfin", CatalogApp: "jellyfin", Train: "stable",
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	if app.ID != "jellyfin" {
		t.Errorf("got %+v", app)
	}
}

func TestCreateApp_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := appJobServer(t, "app.create", "image pull failed", nil)
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateApp(ctx, &types.AppCreateRequest{AppName: "bad"})
	if err == nil || !strings.Contains(err.Error(), "image pull failed") {
		t.Errorf("got %v", err)
	}
}

func TestCreateApp_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "missing app_name"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateApp(ctx, &types.AppCreateRequest{AppName: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating app") {
		t.Errorf("got %v", err)
	}
}

func TestCreateApp_followupGetError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(50)
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "app.create":
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		case "app.get_instance":
			return nil, &RPCError{Code: CodeInternalError, Message: "post-create read failed"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateApp(ctx, &types.AppCreateRequest{AppName: "x"})
	if err == nil || !strings.Contains(err.Error(), "getting app") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateApp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := appJobServer(t, "app.update", "",
		map[string]interface{}{"id": "jellyfin", "name": "jellyfin", "state": "RUNNING", "version": "10.9.5"})
	c, _ := ts.NewClient(ctx)
	app, err := c.UpdateApp(ctx, "jellyfin",
		&types.AppUpdateRequest{Values: map[string]interface{}{"replicas": 2}})
	if err != nil {
		t.Fatalf("UpdateApp: %v", err)
	}
	if app.Version != "10.9.5" {
		t.Errorf("got %+v", app)
	}
}

func TestUpdateApp_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := appJobServer(t, "app.update", "values validation failed", nil)
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateApp(ctx, "x", &types.AppUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "values validation failed") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateApp_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "no such app"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateApp(ctx, "missing", &types.AppUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating app") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateApp_followupGetError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(50)
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "app.update":
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		case "app.get_instance":
			return nil, &RPCError{Code: CodeInternalError, Message: "post-update read failed"}
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateApp(ctx, "x", &types.AppUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "getting app") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteApp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := appJobServer(t, "app.delete", "", nil)
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteApp(ctx, "jellyfin", &types.AppDeleteRequest{RemoveImages: true}); err != nil {
		t.Errorf("DeleteApp: %v", err)
	}
}

func TestDeleteApp_jobFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := appJobServer(t, "app.delete", "pvc still mounted", nil)
	c, _ := ts.NewClient(ctx)
	err := c.DeleteApp(ctx, "x", &types.AppDeleteRequest{})
	if err == nil || !strings.Contains(err.Error(), "pvc still mounted") {
		t.Errorf("got %v", err)
	}
}

func TestDeleteApp_callError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteApp(ctx, "x", &types.AppDeleteRequest{})
	if err == nil || !strings.Contains(err.Error(), "deleting app") {
		t.Errorf("got %v", err)
	}
}

// =============================================================================
// GetAppConfig
// =============================================================================

func TestGetAppConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "app.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		gotParams = params
		return map[string]interface{}{
			"services": map[string]interface{}{
				"app": map[string]interface{}{"image": "busybox:1.36"},
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetAppConfig(ctx, "sleeper")
	if err != nil {
		t.Fatalf("GetAppConfig: %v", err)
	}
	if len(gotParams) != 1 || gotParams[0] != "sleeper" {
		t.Errorf("wire params: %v", gotParams)
	}
	svcs, ok := cfg["services"].(map[string]interface{})
	if !ok || svcs["app"] == nil {
		t.Errorf("decoded config wrong: %#v", cfg)
	}
}

func TestGetAppConfig_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAppConfig(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "getting app config") {
		t.Errorf("got %v", err)
	}
}

func TestGetAppConfig_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAppConfig(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

// TestDeleteApp_forceRemoveCustomAppWire pins the delete wire shape
// for custom apps: options carry force_remove_custom_app true, so a
// broken compose can never wedge a resource destroy.
func TestDeleteApp_forceRemoveCustomAppWire(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(50)
	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "app.delete":
			gotParams = params
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteApp(ctx, "sleeper", &types.AppDeleteRequest{
		RemoveImages:         true,
		ForceRemoveCustomApp: true,
	}); err != nil {
		t.Fatalf("DeleteApp: %v", err)
	}
	if len(gotParams) != 2 {
		t.Fatalf("want [id, options] args, got %d: %v", len(gotParams), gotParams)
	}
	obj, ok := gotParams[1].(map[string]interface{})
	if !ok {
		t.Fatalf("arg 1 is not an object: %T", gotParams[1])
	}
	if obj["force_remove_custom_app"] != true {
		t.Errorf("force_remove_custom_app: got %v", obj["force_remove_custom_app"])
	}
}

// =============================================================================
// Custom compose wire shape (issue #24)
// =============================================================================

// TestCreateApp_customComposeWire pins the app.create wire contract
// for custom installs: one positional object arg carrying custom_app
// plus custom_compose_config_string, with no catalog fields present.
func TestCreateApp_customComposeWire(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(50)
	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "app.create":
			gotParams = params
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		case "app.get_instance":
			return map[string]interface{}{
				"id": "sleeper", "name": "sleeper", "state": "RUNNING", "custom_app": true,
			}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)

	compose := "services:\n  app:\n    image: busybox:1.36\n"
	app, err := c.CreateApp(ctx, &types.AppCreateRequest{
		AppName:                   "sleeper",
		CustomApp:                 true,
		CustomComposeConfigString: compose,
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	if !app.CustomApp {
		t.Errorf("refetched app not flagged custom: %+v", app)
	}

	if len(gotParams) != 1 {
		t.Fatalf("want 1 positional arg, got %d: %v", len(gotParams), gotParams)
	}
	obj, ok := gotParams[0].(map[string]interface{})
	if !ok {
		t.Fatalf("arg 0 is not an object: %T", gotParams[0])
	}
	if obj["app_name"] != "sleeper" {
		t.Errorf("app_name: got %v", obj["app_name"])
	}
	if obj["custom_app"] != true {
		t.Errorf("custom_app: got %v", obj["custom_app"])
	}
	if obj["custom_compose_config_string"] != compose {
		t.Errorf("custom_compose_config_string: got %v", obj["custom_compose_config_string"])
	}
	for _, k := range []string{"catalog_app", "train", "version", "values"} {
		if v, present := obj[k]; present {
			t.Errorf("catalog field %q leaked onto the wire: %v", k, v)
		}
	}
}

// TestUpdateApp_customComposeWire pins the app.update wire contract
// for compose content edits: [id, {custom_compose_config_string}] with
// no values key, proving in-place compose updates need no other field.
func TestUpdateApp_customComposeWire(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const jobID = int64(50)
	var gotParams []interface{}
	ts := NewTestServer(t, func(_ context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "app.update":
			gotParams = params
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
		case "app.get_instance":
			return map[string]interface{}{
				"id": "sleeper", "name": "sleeper", "state": "RUNNING", "custom_app": true,
			}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)

	compose := "services:\n  app:\n    image: busybox:1.36\n    environment:\n      DEMO_ENV: two\n"
	if _, err := c.UpdateApp(ctx, "sleeper", &types.AppUpdateRequest{
		CustomComposeConfigString: compose,
	}); err != nil {
		t.Fatalf("UpdateApp: %v", err)
	}

	if len(gotParams) != 2 {
		t.Fatalf("want [id, body] args, got %d: %v", len(gotParams), gotParams)
	}
	if gotParams[0] != "sleeper" {
		t.Errorf("arg 0 (id): got %v", gotParams[0])
	}
	obj, ok := gotParams[1].(map[string]interface{})
	if !ok {
		t.Fatalf("arg 1 is not an object: %T", gotParams[1])
	}
	if obj["custom_compose_config_string"] != compose {
		t.Errorf("custom_compose_config_string: got %v", obj["custom_compose_config_string"])
	}
	if v, present := obj["values"]; present {
		t.Errorf("values must stay off the wire for compose updates: %v", v)
	}
}
