package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func lxcConfigEntity() map[string]interface{} {
	return map[string]interface{}{
		"id":             1,
		"preferred_pool": "tank",
		"bridge":         nil,
		"v4_network":     "172.200.0.0/24",
		"v6_network":     "fd42:4c58:43ae::/64",
	}
}

func lxcServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
	t.Helper()
	return NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		if m != method {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		}
		if got != nil {
			*got = params
		}
		return result, nil
	})
}

func TestLXCConfig_getAndSet(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("config", func(t *testing.T) {
		c, _ := lxcServer(t, "lxc.config", lxcConfigEntity(), nil).NewClient(ctx)
		got, err := c.GetLXCConfig(ctx)
		if err != nil {
			t.Fatalf("GetLXCConfig: %v", err)
		}
		if got.ID != 1 || got.PreferredPool == nil || *got.PreferredPool != "tank" {
			t.Errorf("decoded %+v", got)
		}
		// bridge is null upstream when TrueNAS manages one. It must stay a
		// nil pointer, because "unconfigured" is a real state distinct from
		// a bridge literally named "".
		if got.Bridge != nil {
			t.Errorf("a null bridge decoded as %q, not nil", *got.Bridge)
		}
		if got.V4Network != "172.200.0.0/24" || got.V6Network != "fd42:4c58:43ae::/64" {
			t.Errorf("networks decoded %+v", got)
		}
	})

	t.Run("update sends only what was set", func(t *testing.T) {
		var params []interface{}
		c, _ := lxcServer(t, "lxc.update", lxcConfigEntity(), &params).NewClient(ctx)
		pool := "tank"
		if _, err := c.SetLXCConfig(ctx, &types.LXCConfigUpdateRequest{PreferredPool: &pool}); err != nil {
			t.Fatalf("SetLXCConfig: %v", err)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		if body["preferred_pool"] != "tank" {
			t.Errorf("preferred_pool = %v", body["preferred_pool"])
		}
		// lxc.update is a ForUpdateMetaclass model: an omitted key leaves
		// the stored value alone, so an unset field must not be sent at all.
		for _, k := range []string{"bridge", "v4_network", "v6_network"} {
			if _, present := body[k]; present {
				t.Errorf("%s sent for an update that did not set it: %v", k, body)
			}
		}
	})

	t.Run("bridge_choices", func(t *testing.T) {
		c, _ := lxcServer(t, "lxc.bridge_choices",
			map[string]interface{}{"[AUTO]": "Automatic", "br0": "br0"}, nil).NewClient(ctx)
		got, err := c.GetLXCBridgeChoices(ctx)
		if err != nil {
			t.Fatalf("GetLXCBridgeChoices: %v", err)
		}
		if got["[AUTO]"] != "Automatic" || got["br0"] != "br0" {
			t.Errorf("choices = %v", got)
		}
	})
}

// The whole namespace is absent before 26.0. A bare -32601 names only the
// method, so every entry point translates it into something actionable.
func TestLXCConfig_methodNotFoundExplainsVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"config", func(c *Client) error { _, err := c.GetLXCConfig(ctx); return err }},
		{"update", func(c *Client) error {
			_, err := c.SetLXCConfig(ctx, &types.LXCConfigUpdateRequest{})
			return err
		}},
		{"bridge_choices", func(c *Client) error { _, err := c.GetLXCBridgeChoices(ctx); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
				return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
			})
			c, _ := ts.NewClient(ctx)
			err := tc.call(c)
			if err == nil {
				t.Fatal("a missing namespace was treated as success")
			}
			if !strings.Contains(err.Error(), "26.0") {
				t.Errorf("diagnostic should name the required version, got: %v", err)
			}
		})
	}
}

// Any other failure keeps its own error, or a permission problem would be
// reported as a version problem.
func TestLXCConfig_otherErrorsNotBlamedOnVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EPERM] nope"}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.GetLXCConfig(ctx); err == nil {
		t.Fatal("an error was swallowed")
	} else if strings.Contains(err.Error(), "26.0") {
		t.Errorf("a permission error was blamed on the TrueNAS version: %v", err)
	}
}

func TestLXCConfig_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := lxcServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}
	if _, err := bad("lxc.config").GetLXCConfig(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("config: %v", err)
	}
	if _, err := bad("lxc.update").SetLXCConfig(ctx, &types.LXCConfigUpdateRequest{}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
	if _, err := bad("lxc.bridge_choices").GetLXCBridgeChoices(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("bridge_choices: %v", err)
	}
}
