package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListCronJobs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cronjob.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{
				"id": 1, "user": "root", "command": "echo a",
				"enabled": true, "stdout": true, "stderr": false,
				"schedule": map[string]interface{}{
					"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*",
				},
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	jobs, err := c.ListCronJobs(ctx)
	if err != nil {
		t.Fatalf("ListCronJobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].User != "root" {
		t.Errorf("got %+v", jobs)
	}
}

func TestListCronJobs_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListCronJobs(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing cron jobs") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestListCronJobs_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListCronJobs(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestGetCronJob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cronjob.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 5, "user": "root", "command": "echo b", "enabled": true,
			"stdout": true, "stderr": false,
			"schedule": map[string]interface{}{
				"minute": "*/5", "hour": "*", "dom": "*", "month": "*", "dow": "*",
			},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	job, err := c.GetCronJob(ctx, 5)
	if err != nil {
		t.Fatalf("GetCronJob: %v", err)
	}
	if job.ID != 5 || job.Schedule.Minute != "*/5" {
		t.Errorf("got %+v", job)
	}
}

func TestGetCronJob_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCronJob(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "getting cron job") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestGetCronJob_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetCronJob(ctx, 5)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestCreateCronJob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cronjob.create" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{
			"id": 6, "user": "root", "command": "echo c", "enabled": true,
			"schedule": map[string]interface{}{"minute": "0"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	job, err := c.CreateCronJob(ctx, &types.CronJobCreateRequest{
		User: "root", Command: "echo c", Enabled: true,
		Schedule: types.Schedule{Minute: "0"},
	})
	if err != nil {
		t.Fatalf("CreateCronJob: %v", err)
	}
	if job.ID != 6 {
		t.Errorf("got %+v", job)
	}
}

func TestCreateCronJob_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCronJob(ctx, &types.CronJobCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "creating cron job") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestCreateCronJob_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.CreateCronJob(ctx, &types.CronJobCreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestUpdateCronJob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "cronjob.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 6, "command": "updated"}, nil
	})
	c, _ := ts.NewClient(ctx)
	job, err := c.UpdateCronJob(ctx, 6, &types.CronJobUpdateRequest{Command: "updated"})
	if err != nil {
		t.Fatalf("UpdateCronJob: %v", err)
	}
	if job.Command != "updated" {
		t.Errorf("got %q", job.Command)
	}
}

func TestUpdateCronJob_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCronJob(ctx, 6, &types.CronJobUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating cron job") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}

func TestUpdateCronJob_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateCronJob(ctx, 6, &types.CronJobUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parse err, got %v", err)
	}
}

func TestDeleteCronJob(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var saw bool
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "cronjob.delete" {
			saw = true
		}
		return nil, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteCronJob(ctx, 6); err != nil {
		t.Fatalf("DeleteCronJob: %v", err)
	}
	if !saw {
		t.Error("server did not see cronjob.delete")
	}
}

func TestDeleteCronJob_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteCronJob(ctx, 6)
	if err == nil || !strings.Contains(err.Error(), "deleting cron job") {
		t.Errorf("expected wrapped err, got %v", err)
	}
}
