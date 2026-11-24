package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func replicationConfigServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
	t.Helper()
	return NewTestServer(t, func(_ context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		if m != method {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		}
		if got != nil {
			*got = params
		}
		return result, nil
	})
}

func TestGetReplicationConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("a limit", func(t *testing.T) {
		entity := map[string]interface{}{"id": 1, "max_parallel_replication_tasks": 5}
		c, _ := replicationConfigServer(t, "replication.config.config", entity, nil).NewClient(ctx)
		got, err := c.GetReplicationConfig(ctx)
		if err != nil {
			t.Fatalf("GetReplicationConfig: %v", err)
		}
		if got.ID != 1 || got.MaxParallelReplicationTasks == nil || *got.MaxParallelReplicationTasks != 5 {
			t.Errorf("decoded %+v", got)
		}
	})

	t.Run("no limit is null", func(t *testing.T) {
		entity := map[string]interface{}{"id": 1, "max_parallel_replication_tasks": nil}
		c, _ := replicationConfigServer(t, "replication.config.config", entity, nil).NewClient(ctx)
		got, err := c.GetReplicationConfig(ctx)
		if err != nil {
			t.Fatalf("GetReplicationConfig: %v", err)
		}
		if got.MaxParallelReplicationTasks != nil {
			t.Errorf("limit = %v, want nil", *got.MaxParallelReplicationTasks)
		}
	})
}

func TestUpdateReplicationConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entity := map[string]interface{}{"id": 1, "max_parallel_replication_tasks": 3}

	body := func(t *testing.T, req *types.ReplicationConfigUpdateRequest) map[string]interface{} {
		t.Helper()
		var params []interface{}
		c, _ := replicationConfigServer(t, "replication.config.update", entity, &params).NewClient(ctx)
		got, err := c.UpdateReplicationConfig(ctx, req)
		if err != nil {
			t.Fatalf("UpdateReplicationConfig: %v", err)
		}
		if got.MaxParallelReplicationTasks == nil || *got.MaxParallelReplicationTasks != 3 {
			t.Errorf("decoded %+v", got)
		}
		b, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		return b
	}

	t.Run("sends only what was set", func(t *testing.T) {
		// A ForUpdateMetaclass model: an omitted key keeps the stored value.
		if b := body(t, &types.ReplicationConfigUpdateRequest{}); len(b) != 0 {
			t.Errorf("an empty request sent %v", b)
		}
	})

	t.Run("sends a limit", func(t *testing.T) {
		b := body(t, &types.ReplicationConfigUpdateRequest{MaxParallelReplicationTasks: types.Value(3)})
		if b["max_parallel_replication_tasks"] != float64(3) {
			t.Errorf("body = %v", b)
		}
	})

	t.Run("sends null to remove the limit", func(t *testing.T) {
		b := body(t, &types.ReplicationConfigUpdateRequest{MaxParallelReplicationTasks: types.Null[int]()})
		v, present := b["max_parallel_replication_tasks"]
		if !present || v != nil {
			t.Errorf("max_parallel_replication_tasks = %v (present %v), want null", v, present)
		}
	})
}

func TestReplicationConfig_serverErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EINVAL] nope"}
	})
	c, _ := ts.NewClient(ctx)

	if _, err := c.GetReplicationConfig(ctx); err == nil || !strings.Contains(err.Error(), "getting replication config") {
		t.Errorf("config: %v", err)
	}
	_, err := c.UpdateReplicationConfig(ctx, &types.ReplicationConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating replication config") {
		t.Errorf("update: %v", err)
	}
}

func TestReplicationConfig_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := replicationConfigServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}

	if _, err := bad("replication.config.config").GetReplicationConfig(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("config: %v", err)
	}
	_, err := bad("replication.config.update").UpdateReplicationConfig(ctx, &types.ReplicationConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
}
