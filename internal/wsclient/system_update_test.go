package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestGetUpdateAutoDownload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.get_auto_download" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.GetUpdateAutoDownload(ctx)
	if err != nil {
		t.Fatalf("GetUpdateAutoDownload: %v", err)
	}
	if !got {
		t.Errorf("expected true, got false")
	}
}

func TestGetUpdateAutoDownload_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetUpdateAutoDownload(ctx)
	if err == nil || !strings.Contains(err.Error(), "auto_download") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestGetUpdateAutoDownload_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		// Server returns a string instead of a bool; decode fails.
		return "not-a-bool", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetUpdateAutoDownload(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestSetUpdateAutoDownload(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gotEnabled bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.set_auto_download" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		if len(params) != 1 {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "expected 1 param"}
		}
		gotEnabled, _ = params[0].(bool)
		return nil, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := c.SetUpdateAutoDownload(ctx, true); err != nil {
		t.Fatalf("SetUpdateAutoDownload: %v", err)
	}
	if !gotEnabled {
		t.Errorf("server did not receive true")
	}
}

func TestSetUpdateAutoDownload_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = c.SetUpdateAutoDownload(ctx, true)
	if err == nil || !strings.Contains(err.Error(), "setting update auto_download") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestGetUpdateTrains(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.get_trains" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"trains": map[string]interface{}{
				"TrueNAS-25.04": map[string]interface{}{"description": "Fangtooth"},
			},
			"current":  "TrueNAS-25.04",
			"selected": "TrueNAS-25.04",
		}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	trains, err := c.GetUpdateTrains(ctx)
	if err != nil {
		t.Fatalf("GetUpdateTrains: %v", err)
	}
	if trains.Current != "TrueNAS-25.04" {
		t.Errorf("Current: got %q", trains.Current)
	}
	if trains.Selected != "TrueNAS-25.04" {
		t.Errorf("Selected: got %q", trains.Selected)
	}
	if trains.Trains["TrueNAS-25.04"].Description != "Fangtooth" {
		t.Errorf("Trains[TrueNAS-25.04].Description: got %q", trains.Trains["TrueNAS-25.04"].Description)
	}
}

func TestGetUpdateTrains_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetUpdateTrains(ctx)
	if err == nil || !strings.Contains(err.Error(), "trains") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestGetUpdateTrains_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.GetUpdateTrains(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse error, got %v", err)
	}
}

func TestSetUpdateTrain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var gotTrain string
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.set_train" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		if len(params) == 1 {
			gotTrain, _ = params[0].(string)
		}
		return nil, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := c.SetUpdateTrain(ctx, "TrueNAS-25.04"); err != nil {
		t.Fatalf("SetUpdateTrain: %v", err)
	}
	if gotTrain != "TrueNAS-25.04" {
		t.Errorf("server saw train %q, want TrueNAS-25.04", gotTrain)
	}
}

func TestSetUpdateTrain_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = c.SetUpdateTrain(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "setting update train") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestCheckUpdateAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "update.check_available" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"status":  "AVAILABLE",
			"version": "TrueNAS-SCALE-25.04.2",
		}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	got, err := c.CheckUpdateAvailable(ctx)
	if err != nil {
		t.Fatalf("CheckUpdateAvailable: %v", err)
	}
	if got.Status != "AVAILABLE" {
		t.Errorf("status: got %q", got.Status)
	}
	if got.Version != "TrueNAS-SCALE-25.04.2" {
		t.Errorf("version: got %q", got.Version)
	}
}

func TestCheckUpdateAvailable_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CheckUpdateAvailable(ctx)
	if err == nil || !strings.Contains(err.Error(), "checking update availability") {
		t.Errorf("expected wrapped error, got %v", err)
	}
}

func TestCheckUpdateAvailable_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = c.CheckUpdateAvailable(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse error, got %v", err)
	}
}
