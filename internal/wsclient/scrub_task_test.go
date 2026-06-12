package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestGetScrubTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.scrub.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 1, "pool": 1, "threshold": 35}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.GetScrubTask(ctx, 1)
	if err != nil {
		t.Fatalf("GetScrubTask: %v", err)
	}
	if r.Threshold != 35 {
		t.Errorf("got %+v", r)
	}
}

func TestGetScrubTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetScrubTask(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting scrub task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetScrubTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetScrubTask(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateScrubTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.scrub.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "pool": 1}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.CreateScrubTask(ctx, &types.ScrubTaskCreateRequest{Pool: 1})
	if err != nil {
		t.Fatalf("CreateScrubTask: %v", err)
	}
	if r.ID != 2 {
		t.Errorf("got %+v", r)
	}
}

func TestCreateScrubTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateScrubTask(ctx, &types.ScrubTaskCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating scrub task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateScrubTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateScrubTask(ctx, &types.ScrubTaskCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateScrubTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "pool.scrub.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 2, "threshold": 7}, nil
	})
	c, _ := ts.NewClient(ctx)
	r, err := c.UpdateScrubTask(ctx, 2, &types.ScrubTaskUpdateRequest{Threshold: 7})
	if err != nil {
		t.Fatalf("UpdateScrubTask: %v", err)
	}
	if r.Threshold != 7 {
		t.Errorf("got %+v", r)
	}
}

func TestUpdateScrubTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateScrubTask(ctx, 2, &types.ScrubTaskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating scrub task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateScrubTask_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateScrubTask(ctx, 2, &types.ScrubTaskUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteScrubTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "pool.scrub.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteScrubTask(ctx, 2); err != nil {
		t.Fatalf("DeleteScrubTask: %v", err)
	}
	if !saw {
		t.Error("server did not see pool.scrub.delete")
	}
}

func TestDeleteScrubTask_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteScrubTask(ctx, 2)
	if err == nil || !strings.Contains(err.Error(), "deleting scrub task") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
