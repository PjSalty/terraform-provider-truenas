package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// The update client was rewritten onto update.config / update.update after
// issue #32. The five methods it used to call have no Args model in any
// versioned API directory and no @api_method definition anywhere in
// middleware, so they were never registered on any supported release.

func updateServer(t *testing.T, handler TestHandler) *TestServer {
	t.Helper()
	return NewTestServer(t, handler)
}

func TestGetUpdateConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.config" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "autocheck": true, "profile": "GENERAL"}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetUpdateConfig(ctx)
	if err != nil {
		t.Fatalf("GetUpdateConfig: %v", err)
	}
	if !cfg.Autocheck {
		t.Error("autocheck not decoded")
	}
	if cfg.Profile == nil || *cfg.Profile != "GENERAL" {
		t.Errorf("profile = %v", cfg.Profile)
	}
}

// profile is nullable on the wire; a pointer keeps "not set" distinct from
// the empty string.
func TestGetUpdateConfig_nullProfile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return map[string]interface{}{"id": 1, "autocheck": false, "profile": nil}, nil
	})
	c, _ := ts.NewClient(ctx)
	cfg, err := c.GetUpdateConfig(ctx)
	if err != nil {
		t.Fatalf("GetUpdateConfig: %v", err)
	}
	if cfg.Profile != nil {
		t.Errorf("a null profile decoded to %v, want nil", cfg.Profile)
	}
}

func TestGetUpdateConfig_errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.GetUpdateConfig(ctx); err == nil || !strings.Contains(err.Error(), "getting update config") {
		t.Errorf("got %v", err)
	}

	ts2 := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c2, _ := ts2.NewClient(ctx)
	if _, err := c2.GetUpdateConfig(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestSetUpdateConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var body map[string]interface{}
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		if len(params) > 0 {
			if m, ok := params[0].(map[string]interface{}); ok {
				body = m
			}
		}
		return map[string]interface{}{"id": 1, "autocheck": true, "profile": "GENERAL"}, nil
	})
	c, _ := ts.NewClient(ctx)
	autocheck := true
	profile := "GENERAL"
	if _, err := c.SetUpdateConfig(ctx, &types.UpdateConfigUpdateRequest{Autocheck: &autocheck, Profile: &profile}); err != nil {
		t.Fatalf("SetUpdateConfig: %v", err)
	}
	if body["autocheck"] != true || body["profile"] != "GENERAL" {
		t.Errorf("body = %v", body)
	}
}

// Nil fields must be omitted, not sent as zero values. Sending autocheck=false
// for an unset attribute would silently disable nightly update checks.
func TestSetUpdateConfig_omitsUnsetFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var body map[string]interface{}
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if len(params) > 0 {
			if m, ok := params[0].(map[string]interface{}); ok {
				body = m
			}
		}
		return map[string]interface{}{"id": 1, "autocheck": false, "profile": nil}, nil
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.SetUpdateConfig(ctx, &types.UpdateConfigUpdateRequest{}); err != nil {
		t.Fatalf("SetUpdateConfig: %v", err)
	}
	if _, present := body["autocheck"]; present {
		t.Errorf("autocheck sent for an unset field: %v", body)
	}
	if _, present := body["profile"]; present {
		t.Errorf("profile sent for an unset field: %v", body)
	}
}

func TestSetUpdateConfig_errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.SetUpdateConfig(ctx, &types.UpdateConfigUpdateRequest{}); err == nil ||
		!strings.Contains(err.Error(), "updating update config") {
		t.Errorf("got %v", err)
	}

	ts2 := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "nope", nil
	})
	c2, _ := ts2.NewClient(ctx)
	if _, err := c2.SetUpdateConfig(ctx, &types.UpdateConfigUpdateRequest{}); err == nil ||
		!strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetUpdateProfileChoices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.profile_choices" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"GENERAL":   map[string]interface{}{"name": "GENERAL", "footnote": "", "description": "d", "available": true},
			"DEVELOPER": map[string]interface{}{"name": "DEVELOPER", "footnote": "", "description": "d", "available": false},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	choices, err := c.GetUpdateProfileChoices(ctx)
	if err != nil {
		t.Fatalf("GetUpdateProfileChoices: %v", err)
	}
	if !choices["GENERAL"].Available {
		t.Error("GENERAL should be available")
	}
	if choices["DEVELOPER"].Available {
		t.Error("DEVELOPER should not be available")
	}
}

func TestGetUpdateProfileChoices_errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.GetUpdateProfileChoices(ctx); err == nil ||
		!strings.Contains(err.Error(), "getting update profile choices") {
		t.Errorf("got %v", err)
	}

	ts2 := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "nope", nil
	})
	c2, _ := ts2.NewClient(ctx)
	if _, err := c2.GetUpdateProfileChoices(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetUpdateStatus(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.status" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"code": "NORMAL",
			"status": map[string]interface{}{
				"current_version": map[string]interface{}{"version": "25.10.4", "profile": "GENERAL"},
				"new_version":     map[string]interface{}{"version": "25.10.6", "profile": "GENERAL"},
			},
			"error": nil,
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	st, err := c.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatalf("GetUpdateStatus: %v", err)
	}
	if st.Code != "NORMAL" {
		t.Errorf("code = %q", st.Code)
	}
	if st.Status == nil || st.Status.CurrentVersion.Version != "25.10.4" {
		t.Errorf("current version not decoded: %+v", st.Status)
	}
	if st.Status.NewVersion == nil || st.Status.NewVersion.Version != "25.10.6" {
		t.Errorf("new version not decoded")
	}
}

// status and error are nullable; a nil status must decode cleanly rather than
// panicking or being mistaken for "no update available".
func TestGetUpdateStatus_nullMembers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return map[string]interface{}{
			"code":   "ERROR",
			"status": nil,
			"error":  map[string]interface{}{"errname": "ENONET", "reason": "no network"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	st, err := c.GetUpdateStatus(ctx)
	if err != nil {
		t.Fatalf("GetUpdateStatus: %v", err)
	}
	if st.Status != nil {
		t.Error("a null status decoded to a non-nil value")
	}
	if st.Error == nil || st.Error.Errname != "ENONET" {
		t.Errorf("error detail not decoded: %+v", st.Error)
	}
}

func TestGetUpdateStatus_errors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.GetUpdateStatus(ctx); err == nil || !strings.Contains(err.Error(), "getting update status") {
		t.Errorf("got %v", err)
	}

	ts2 := updateServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "nope", nil
	})
	c2, _ := ts2.NewClient(ctx)
	if _, err := c2.GetUpdateStatus(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}
