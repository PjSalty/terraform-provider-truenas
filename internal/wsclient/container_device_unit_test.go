package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func containerDeviceEntity() map[string]interface{} {
	return map[string]interface{}{
		"id": 9, "container": 3,
		"attributes": map[string]interface{}{
			"dtype": "FILESYSTEM", "source": "/mnt/tank/media", "target": "/srv/media",
		},
	}
}

func deviceServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
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

func TestContainerDeviceCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("get", func(t *testing.T) {
		c, _ := deviceServer(t, "container.device.get_instance", containerDeviceEntity(), nil).NewClient(ctx)
		got, err := c.GetContainerDevice(ctx, 9)
		if err != nil {
			t.Fatalf("GetContainerDevice: %v", err)
		}
		if got.ID != 9 || got.Container != 3 {
			t.Errorf("decoded %+v", got)
		}
		if got.Attributes["dtype"] != "FILESYSTEM" || got.Attributes["source"] != "/mnt/tank/media" {
			t.Errorf("attributes = %v", got.Attributes)
		}
	})

	t.Run("list", func(t *testing.T) {
		c, _ := deviceServer(t, "container.device.query", []interface{}{containerDeviceEntity()}, nil).NewClient(ctx)
		got, err := c.ListContainerDevices(ctx)
		if err != nil {
			t.Fatalf("ListContainerDevices: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d devices", len(got))
		}
	})

	t.Run("create", func(t *testing.T) {
		var params []interface{}
		c, _ := deviceServer(t, "container.device.create", containerDeviceEntity(), &params).NewClient(ctx)
		got, err := c.CreateContainerDevice(ctx, &types.ContainerDeviceCreateRequest{
			Container:  3,
			Attributes: map[string]interface{}{"dtype": "FILESYSTEM", "source": "/mnt/tank/media", "target": "/srv/media"},
		})
		if err != nil {
			t.Fatalf("CreateContainerDevice: %v", err)
		}
		if got.ID != 9 {
			t.Errorf("id = %d", got.ID)
		}
		// container.device.create takes ONE positional argument upstream.
		if len(params) != 1 {
			t.Fatalf("sent %d arguments, want 1", len(params))
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		attrs, ok := body["attributes"].(map[string]interface{})
		if !ok || attrs["dtype"] != "FILESYSTEM" {
			t.Errorf("attributes = %v", body["attributes"])
		}
	})

	t.Run("update", func(t *testing.T) {
		var params []interface{}
		c, _ := deviceServer(t, "container.device.update", containerDeviceEntity(), &params).NewClient(ctx)
		if _, err := c.UpdateContainerDevice(ctx, 9, &types.ContainerDeviceUpdateRequest{
			Attributes: map[string]interface{}{"dtype": "FILESYSTEM", "source": "/mnt/tank/x", "target": "/srv/x"},
		}); err != nil {
			t.Fatalf("UpdateContainerDevice: %v", err)
		}
		if params[0] != float64(9) {
			t.Errorf("id param = %v, want 9", params[0])
		}
	})

	t.Run("nic_attach_choices", func(t *testing.T) {
		c, _ := deviceServer(t, "container.device.nic_attach_choices", map[string]interface{}{
			"BRIDGE": []interface{}{"truenasbr0"}, "MACVLAN": []interface{}{"ens18"},
		}, nil).NewClient(ctx)
		got, err := c.GetContainerNICAttachChoices(ctx)
		if err != nil {
			t.Fatalf("GetContainerNICAttachChoices: %v", err)
		}
		if len(got.Bridge) != 1 || got.Bridge[0] != "truenasbr0" {
			t.Errorf("bridges = %v", got.Bridge)
		}
		if len(got.Macvlan) != 1 || got.Macvlan[0] != "ens18" {
			t.Errorf("macvlan = %v", got.Macvlan)
		}
	})
}

// raw_file and zvol destroy the storage behind a device. A nil options
// argument must send the safe zero value, never omit the parameter and let
// middleware pick.
func TestDeleteContainerDevice_nilOptionsSendSafeDefaults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var params []interface{}
	c, _ := deviceServer(t, "container.device.delete", true, &params).NewClient(ctx)
	if err := c.DeleteContainerDevice(ctx, 9, nil); err != nil {
		t.Fatalf("DeleteContainerDevice: %v", err)
	}
	body, ok := params[1].(map[string]interface{})
	if !ok {
		t.Fatalf("options = %#v", params[1])
	}
	for _, k := range []string{"force", "raw_file", "zvol"} {
		if body[k] != false {
			t.Errorf("%s = %v, want false", k, body[k])
		}
	}
}

// The whole namespace is absent before 26.0. A bare -32601 names only the
// method, so every entry point translates it into something actionable.
func TestContainerDevice_methodNotFoundExplainsVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"get", func(c *Client) error { _, err := c.GetContainerDevice(ctx, 1); return err }},
		{"list", func(c *Client) error { _, err := c.ListContainerDevices(ctx); return err }},
		{"create", func(c *Client) error {
			_, err := c.CreateContainerDevice(ctx, &types.ContainerDeviceCreateRequest{Container: 1})
			return err
		}},
		{"update", func(c *Client) error {
			_, err := c.UpdateContainerDevice(ctx, 1, &types.ContainerDeviceUpdateRequest{})
			return err
		}},
		{"delete", func(c *Client) error { return c.DeleteContainerDevice(ctx, 1, nil) }},
		{"nic_attach_choices", func(c *Client) error { _, err := c.GetContainerNICAttachChoices(ctx); return err }},
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
func TestContainerDevice_otherErrorsNotBlamedOnVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EPERM] nope"}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.DeleteContainerDevice(ctx, 1, nil); err == nil {
		t.Fatal("an error was swallowed")
	} else if strings.Contains(err.Error(), "26.0") {
		t.Errorf("a permission error was blamed on the TrueNAS version: %v", err)
	}
}

func TestContainerDevice_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := deviceServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}
	if _, err := bad("container.device.get_instance").GetContainerDevice(ctx, 1); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("get: %v", err)
	}
	if _, err := bad("container.device.query").ListContainerDevices(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("list: %v", err)
	}
	if _, err := bad("container.device.create").CreateContainerDevice(ctx, &types.ContainerDeviceCreateRequest{}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("create: %v", err)
	}
	if _, err := bad("container.device.update").UpdateContainerDevice(ctx, 1, &types.ContainerDeviceUpdateRequest{}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
	if _, err := bad("container.device.nic_attach_choices").GetContainerNICAttachChoices(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("nic_attach_choices: %v", err)
	}
}
