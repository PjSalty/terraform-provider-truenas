package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetAlertService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "alertservice.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":      7,
			"name":    "ops",
			"type":    "Mail",
			"enabled": true,
			"level":   "WARNING",
		}, nil
	})
	c, _ := ts.NewClient(ctx)

	svc, err := c.GetAlertService(ctx, 7)
	if err != nil {
		t.Fatalf("GetAlertService: %v", err)
	}
	if svc.ID != 7 || svc.Name != "ops" {
		t.Errorf("got %+v", svc)
	}
}

func TestGetAlertService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAlertService(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting alert service") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetAlertService_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetAlertService(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateAlertService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "alertservice.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":      8,
			"name":    "new",
			"type":    "Slack",
			"enabled": true,
			"level":   "ERROR",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	svc, err := c.CreateAlertService(ctx, &types.AlertServiceCreateRequest{
		Name:    "new",
		Enabled: true,
		Level:   "ERROR",
	})
	if err != nil {
		t.Fatalf("CreateAlertService: %v", err)
	}
	if svc.ID != 8 || svc.Name != "new" {
		t.Errorf("got %+v", svc)
	}
}

func TestCreateAlertService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateAlertService(ctx, &types.AlertServiceCreateRequest{Name: "x"})
	if err == nil || !strings.Contains(err.Error(), "creating alert service") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateAlertService_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateAlertService(ctx, &types.AlertServiceCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateAlertService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "alertservice.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id":   9,
			"name": "updated",
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	svc, err := c.UpdateAlertService(ctx, 9, &types.AlertServiceUpdateRequest{Name: "updated"})
	if err != nil {
		t.Fatalf("UpdateAlertService: %v", err)
	}
	if svc.Name != "updated" {
		t.Errorf("got %q", svc.Name)
	}
}

func TestUpdateAlertService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateAlertService(ctx, 9, &types.AlertServiceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating alert service") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateAlertService_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateAlertService(ctx, 9, &types.AlertServiceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteAlertService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var sawDelete bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "alertservice.delete" {
			sawDelete = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteAlertService(ctx, 9); err != nil {
		t.Fatalf("DeleteAlertService: %v", err)
	}
	if !sawDelete {
		t.Error("server did not see alertservice.delete")
	}
}

func TestDeleteAlertService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteAlertService(ctx, 9)
	if err == nil || !strings.Contains(err.Error(), "deleting alert service") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
