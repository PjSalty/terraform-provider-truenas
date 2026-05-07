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
