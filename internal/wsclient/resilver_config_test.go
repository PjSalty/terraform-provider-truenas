package wsclient

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// resilverConfigEntity is what pool.resilver.config returns on a fresh
// TrueNAS 25.10 install.
func resilverConfigEntity() map[string]interface{} {
	return map[string]interface{}{
		"id":      1,
		"enabled": false,
		"begin":   "18:00",
		"end":     "09:00",
		"weekday": []int{1, 2, 3, 4, 5, 6, 7},
	}
}

func resilverServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
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

func TestGetResilverConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _ := resilverServer(t, "pool.resilver.config", resilverConfigEntity(), nil).NewClient(ctx)
	got, err := c.GetResilverConfig(ctx)
	if err != nil {
		t.Fatalf("GetResilverConfig: %v", err)
	}
	if got.ID != 1 || got.Enabled || got.Begin != "18:00" || got.End != "09:00" {
		t.Errorf("decoded %+v", got)
	}
	if !slices.Equal(got.Weekday, []int{1, 2, 3, 4, 5, 6, 7}) {
		t.Errorf("weekday = %v", got.Weekday)
	}
}

func TestUpdateResilverConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("sends only what was set", func(t *testing.T) {
		var params []interface{}
		entity := resilverConfigEntity()
		entity["enabled"] = true
		c, _ := resilverServer(t, "pool.resilver.update", entity, &params).NewClient(ctx)

		enabled := true
		got, err := c.UpdateResilverConfig(ctx, &types.ResilverConfigUpdateRequest{Enabled: &enabled})
		if err != nil {
			t.Fatalf("UpdateResilverConfig: %v", err)
		}
		if !got.Enabled {
			t.Errorf("decoded %+v", got)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		if body["enabled"] != true {
			t.Errorf("enabled = %v", body["enabled"])
		}
		// pool.resilver.update is a ForUpdateMetaclass model: an omitted
		// key keeps the stored value, so an unset field must not be sent.
		for _, k := range []string{"begin", "end", "weekday"} {
			if _, present := body[k]; present {
				t.Errorf("%s sent for an update that did not set it: %v", k, body)
			}
		}
	})

	t.Run("sends the whole window", func(t *testing.T) {
		var params []interface{}
		c, _ := resilverServer(t, "pool.resilver.update", resilverConfigEntity(), &params).NewClient(ctx)

		begin, end := "22:30", "06:00"
		req := &types.ResilverConfigUpdateRequest{Begin: &begin, End: &end, Weekday: []int{6, 7}}
		if _, err := c.UpdateResilverConfig(ctx, req); err != nil {
			t.Fatalf("UpdateResilverConfig: %v", err)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		if body["begin"] != "22:30" || body["end"] != "06:00" {
			t.Errorf("window = %v to %v", body["begin"], body["end"])
		}
		days, ok := body["weekday"].([]interface{})
		if !ok || len(days) != 2 || days[0] != float64(6) || days[1] != float64(7) {
			t.Errorf("weekday = %#v", body["weekday"])
		}
	})
}

func TestResilverConfig_serverErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EINVAL] nope"}
	})
	c, _ := ts.NewClient(ctx)

	if _, err := c.GetResilverConfig(ctx); err == nil || !strings.Contains(err.Error(), "getting resilver config") {
		t.Errorf("config: %v", err)
	}
	_, err := c.UpdateResilverConfig(ctx, &types.ResilverConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating resilver config") {
		t.Errorf("update: %v", err)
	}
}

func TestResilverConfig_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := resilverServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}

	if _, err := bad("pool.resilver.config").GetResilverConfig(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("config: %v", err)
	}
	_, err := bad("pool.resilver.update").UpdateResilverConfig(ctx, &types.ResilverConfigUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
}
