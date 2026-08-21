package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func webshareEntity() map[string]interface{} {
	return map[string]interface{}{
		"id": 7, "name": "docs", "path": "/mnt/tank/docs",
		"enabled": true, "is_home_base": false,
		"dataset": "tank/docs", "relative_path": "", "locked": false,
	}
}

func webshareServer(t *testing.T, method string, result interface{}, rpcErr *RPCError, got *[]interface{}) *TestServer {
	t.Helper()
	return NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		if m != method {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		}
		if got != nil {
			*got = params
		}
		if rpcErr != nil {
			return nil, rpcErr
		}
		return result, nil
	})
}

func TestWebshareCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("get", func(t *testing.T) {
		c, _ := webshareServer(t, "sharing.webshare.get_instance", webshareEntity(), nil, nil).NewClient(ctx)
		got, err := c.GetWebshare(ctx, 7)
		if err != nil {
			t.Fatalf("GetWebshare: %v", err)
		}
		if got.Name != "docs" || got.Dataset == nil || *got.Dataset != "tank/docs" {
			t.Errorf("decoded %+v", got)
		}
	})

	t.Run("list", func(t *testing.T) {
		c, _ := webshareServer(t, "sharing.webshare.query", []interface{}{webshareEntity()}, nil, nil).NewClient(ctx)
		got, err := c.ListWebshares(ctx)
		if err != nil {
			t.Fatalf("ListWebshares: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d shares", len(got))
		}
	})

	t.Run("create", func(t *testing.T) {
		var params []interface{}
		c, _ := webshareServer(t, "sharing.webshare.create", webshareEntity(), nil, &params).NewClient(ctx)
		enabled := true
		got, err := c.CreateWebshare(ctx, &types.WebshareCreateRequest{
			Name: "docs", Path: "/mnt/tank/docs", Enabled: &enabled,
		})
		if err != nil {
			t.Fatalf("CreateWebshare: %v", err)
		}
		if got.ID != 7 {
			t.Errorf("id = %d", got.ID)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		// is_home_base was not set, so it must not appear: sending false
		// would override a server default rather than leaving it alone.
		if _, present := body["is_home_base"]; present {
			t.Errorf("is_home_base sent for a request that did not set it: %v", body)
		}
	})

	t.Run("update", func(t *testing.T) {
		var params []interface{}
		c, _ := webshareServer(t, "sharing.webshare.update", webshareEntity(), nil, &params).NewClient(ctx)
		name := "renamed"
		if _, err := c.UpdateWebshare(ctx, 7, &types.WebshareUpdateRequest{Name: &name}); err != nil {
			t.Fatalf("UpdateWebshare: %v", err)
		}
		if params[0] != float64(7) {
			t.Errorf("id param = %v, want 7", params[0])
		}
		body := params[1].(map[string]interface{})
		if body["name"] != "renamed" {
			t.Errorf("name = %v", body["name"])
		}
		if _, present := body["path"]; present {
			t.Errorf("path sent for an update that did not set it: %v", body)
		}
	})

	t.Run("delete", func(t *testing.T) {
		c, _ := webshareServer(t, "sharing.webshare.delete", true, nil, nil).NewClient(ctx)
		if err := c.DeleteWebshare(ctx, 7); err != nil {
			t.Errorf("DeleteWebshare: %v", err)
		}
	})
}

// The whole namespace is absent before 26.0. A bare -32601 names only the
// method, so every entry point translates it into something actionable.
func TestWebshare_methodNotFoundExplainsVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	notFound := &RPCError{Code: CodeMethodNotFound, Message: "sharing.webshare.query"}

	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"get", func(c *Client) error { _, err := c.GetWebshare(ctx, 1); return err }},
		{"list", func(c *Client) error { _, err := c.ListWebshares(ctx); return err }},
		{"create", func(c *Client) error {
			_, err := c.CreateWebshare(ctx, &types.WebshareCreateRequest{Name: "x", Path: "/y"})
			return err
		}},
		{"update", func(c *Client) error {
			_, err := c.UpdateWebshare(ctx, 1, &types.WebshareUpdateRequest{})
			return err
		}},
		{"delete", func(c *Client) error { return c.DeleteWebshare(ctx, 1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
				return nil, notFound
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
func TestWebshare_otherErrorsNotBlamedOnVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EPERM] nope"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.DeleteWebshare(ctx, 1)
	if err == nil {
		t.Fatal("an error was swallowed")
	}
	if strings.Contains(err.Error(), "26.0") {
		t.Errorf("a permission error was blamed on the TrueNAS version: %v", err)
	}
}

func TestWebshare_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := webshareServer(t, method, "not-an-object", nil, nil).NewClient(ctx)
		return c
	}
	if _, err := bad("sharing.webshare.get_instance").GetWebshare(ctx, 1); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("get: %v", err)
	}
	if _, err := bad("sharing.webshare.query").ListWebshares(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("list: %v", err)
	}
	if _, err := bad("sharing.webshare.create").CreateWebshare(ctx, &types.WebshareCreateRequest{}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("create: %v", err)
	}
	if _, err := bad("sharing.webshare.update").UpdateWebshare(ctx, 1, &types.WebshareUpdateRequest{}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
}
