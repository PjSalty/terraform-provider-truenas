package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func containerEntity() map[string]interface{} {
	return map[string]interface{}{
		"id": 3, "uuid": "1b4e28ba-2fa1-11d2-883f-0016d3cca427", "name": "web",
		"description": "front end", "cpuset": nil, "autostart": true,
		"time": "LOCAL", "shutdown_timeout": 90, "dataset": "test/containers/web",
		"init": "/sbin/init", "initdir": nil, "initenv": map[string]interface{}{"TZ": "UTC"},
		"inituser": nil, "initgroup": nil,
		"idmap":               map[string]interface{}{"type": "DEFAULT"},
		"capabilities_policy": "DEFAULT",
		"capabilities_state":  map[string]interface{}{},
		"default_network":     "truenasbr0",
		"status":              map[string]interface{}{"state": "STOPPED", "pid": nil, "domain_state": nil},
	}
}

// containerServer answers the given method, and answers job bookkeeping so
// the job-backed calls (create, delete, stop) can complete.
func containerServer(t *testing.T, method string, result interface{}, got *[]interface{}) *TestServer {
	t.Helper()
	return NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		if m == method {
			if got != nil {
				*got = params
			}
			return result, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
	})
}

func TestContainerCRUD(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("get", func(t *testing.T) {
		c, _ := containerServer(t, "container.get_instance", containerEntity(), nil).NewClient(ctx)
		got, err := c.GetContainer(ctx, 3)
		if err != nil {
			t.Fatalf("GetContainer: %v", err)
		}
		if got.Name != "web" || got.Dataset != "test/containers/web" {
			t.Errorf("decoded %+v", got)
		}
		// A null upstream field must stay a nil pointer: "unset" is a real
		// state distinct from the empty string.
		if got.Cpuset != nil || got.InitDir != nil {
			t.Errorf("a null decoded as non-nil: %+v", got)
		}
		if got.Idmap == nil || got.Idmap.Type != "DEFAULT" || got.Idmap.Slice != nil {
			t.Errorf("idmap decoded %+v", got.Idmap)
		}
		if got.Status.State != "STOPPED" || got.Status.PID != nil {
			t.Errorf("status decoded %+v", got.Status)
		}
		if got.InitEnv["TZ"] != "UTC" {
			t.Errorf("initenv decoded %v", got.InitEnv)
		}
	})

	t.Run("list", func(t *testing.T) {
		c, _ := containerServer(t, "container.query", []interface{}{containerEntity()}, nil).NewClient(ctx)
		got, err := c.ListContainers(ctx)
		if err != nil {
			t.Fatalf("ListContainers: %v", err)
		}
		if len(got) != 1 {
			t.Errorf("got %d containers", len(got))
		}
	})

	t.Run("update sends only what was set", func(t *testing.T) {
		var params []interface{}
		c, _ := containerServer(t, "container.update", containerEntity(), &params).NewClient(ctx)
		desc := "renamed"
		if _, err := c.UpdateContainer(ctx, 3, &types.ContainerUpdateRequest{Description: &desc}); err != nil {
			t.Fatalf("UpdateContainer: %v", err)
		}
		if params[0] != float64(3) {
			t.Errorf("id param = %v, want 3", params[0])
		}
		body, ok := params[1].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[1])
		}
		if body["description"] != "renamed" {
			t.Errorf("description = %v", body["description"])
		}
		// container.update is ForUpdateMetaclass: an omitted key leaves the
		// stored value alone, so unset fields must not be sent at all.
		for _, k := range []string{"name", "autostart", "init", "initenv"} {
			if _, present := body[k]; present {
				t.Errorf("%s sent for an update that did not set it: %v", k, body)
			}
		}
		// pool, image and idmap are excluded from the upstream update
		// model; sending any of them is rejected server-side.
		for _, k := range []string{"pool", "image", "idmap"} {
			if _, present := body[k]; present {
				t.Errorf("%s is create-only and must never appear in an update body: %v", k, body)
			}
		}
	})

	t.Run("create waits for the job and decodes its result", func(t *testing.T) {
		var params []interface{}
		const jobID = int64(12)
		ts := NewTestServer(t, func(ctx context.Context, m string, p []interface{}) (interface{}, *RPCError) {
			switch m {
			case "container.create":
				params = p
				return jobID, nil
			case "core.get_jobs":
				return []interface{}{map[string]interface{}{
					"id": jobID, "state": "SUCCESS", "result": containerEntity(),
				}}, nil
			}
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		})
		c, _ := ts.NewClient(ctx)
		got, err := c.CreateContainer(ctx, &types.ContainerCreateRequest{
			Name:  "web",
			Image: types.ContainerImageRef{Name: "alpine:3.21:amd64:default", Version: "20260820_13:00"},
		})
		if err != nil {
			t.Fatalf("CreateContainer: %v", err)
		}
		if got.ID != 3 {
			t.Errorf("id = %d", got.ID)
		}
		body, ok := params[0].(map[string]interface{})
		if !ok {
			t.Fatalf("body = %#v", params[0])
		}
		img, ok := body["image"].(map[string]interface{})
		if !ok || img["name"] != "alpine:3.21:amd64:default" {
			t.Errorf("image = %v", body["image"])
		}
		// Unset optional fields must be omitted so the server default
		// applies, rather than sent as a zero value chosen here.
		for _, k := range []string{"uuid", "pool", "autostart", "idmap"} {
			if _, present := body[k]; present {
				t.Errorf("%s sent for a create that did not set it: %v", k, body)
			}
		}
	})

	t.Run("start is not a job", func(t *testing.T) {
		var calls []string
		ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
			calls = append(calls, m)
			return nil, nil
		})
		c, _ := ts.NewClient(ctx)
		if err := c.StartContainer(ctx, 3); err != nil {
			t.Fatalf("StartContainer: %v", err)
		}
		// A job-backed call would follow up with core.get_jobs polling.
		for _, m := range calls {
			if strings.HasPrefix(m, "core.") {
				t.Errorf("container.start polled a job (%s); it is not @job upstream", m)
			}
		}
	})

	t.Run("pool_choices", func(t *testing.T) {
		c, _ := containerServer(t, "container.pool_choices",
			map[string]interface{}{"tank": "tank"}, nil).NewClient(ctx)
		got, err := c.GetContainerPoolChoices(ctx)
		if err != nil {
			t.Fatalf("GetContainerPoolChoices: %v", err)
		}
		if got["tank"] != "tank" {
			t.Errorf("choices = %v", got)
		}
	})
}

// The delete and stop options carry destructive flags. A nil options
// argument must send the safe zero value, never omit the parameter and
// let middleware pick.
func TestContainer_nilOptionsSendSafeDefaults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, tc := range []struct {
		name, method string
		call         func(c *Client) error
	}{
		{"delete", "container.delete", func(c *Client) error { return c.DeleteContainer(ctx, 3, nil) }},
		{"stop", "container.stop", func(c *Client) error { return c.StopContainer(ctx, 3, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var params []interface{}
			const jobID = int64(11)
			ts := NewTestServer(t, func(ctx context.Context, m string, p []interface{}) (interface{}, *RPCError) {
				switch m {
				case tc.method:
					params = p
					return jobID, nil
				case "core.get_jobs":
					return []interface{}{map[string]interface{}{
						"id": jobID, "state": "SUCCESS", "result": nil,
					}}, nil
				}
				return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
			})
			c, _ := ts.NewClient(ctx)
			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			body, ok := params[1].(map[string]interface{})
			if !ok {
				t.Fatalf("options = %#v", params[1])
			}
			if body["force"] != false {
				t.Errorf("force = %v, want false", body["force"])
			}
			if tc.name == "delete" && body["recursive"] != false {
				t.Errorf("recursive = %v, want false: it destroys snapshots and clones", body["recursive"])
			}
		})
	}
}

// The whole namespace is absent before 26.0. A bare -32601 names only the
// method, so every entry point translates it into something actionable.
func TestContainer_methodNotFoundExplainsVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cases := []struct {
		name string
		call func(c *Client) error
	}{
		{"get", func(c *Client) error { _, err := c.GetContainer(ctx, 1); return err }},
		{"list", func(c *Client) error { _, err := c.ListContainers(ctx); return err }},
		{"create", func(c *Client) error {
			_, err := c.CreateContainer(ctx, &types.ContainerCreateRequest{Name: "x"})
			return err
		}},
		{"update", func(c *Client) error {
			_, err := c.UpdateContainer(ctx, 1, &types.ContainerUpdateRequest{})
			return err
		}},
		{"delete", func(c *Client) error { return c.DeleteContainer(ctx, 1, nil) }},
		{"start", func(c *Client) error { return c.StartContainer(ctx, 1) }},
		{"stop", func(c *Client) error { return c.StopContainer(ctx, 1, nil) }},
		{"pool_choices", func(c *Client) error { _, err := c.GetContainerPoolChoices(ctx); return err }},
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
func TestContainer_otherErrorsNotBlamedOnVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, m string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeMethodCallError, Message: "[EPERM] nope"}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.GetContainer(ctx, 1); err == nil {
		t.Fatal("an error was swallowed")
	} else if strings.Contains(err.Error(), "26.0") {
		t.Errorf("a permission error was blamed on the TrueNAS version: %v", err)
	}
}

func TestContainer_decodeErrors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bad := func(method string) *Client {
		c, _ := containerServer(t, method, "not-an-object", nil).NewClient(ctx)
		return c
	}
	if _, err := bad("container.get_instance").GetContainer(ctx, 1); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("get: %v", err)
	}
	if _, err := bad("container.query").ListContainers(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("list: %v", err)
	}
	if _, err := bad("container.update").UpdateContainer(ctx, 1, &types.ContainerUpdateRequest{}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("update: %v", err)
	}
	if _, err := bad("container.pool_choices").GetContainerPoolChoices(ctx); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("pool_choices: %v", err)
	}

	// create decodes the JOB result, not the immediate reply, so its
	// decode failure has to be provoked through a completed job.
	const jobID = int64(13)
	ts := NewTestServer(t, func(ctx context.Context, m string, p []interface{}) (interface{}, *RPCError) {
		switch m {
		case "container.create":
			return jobID, nil
		case "core.get_jobs":
			return []interface{}{map[string]interface{}{
				"id": jobID, "state": "SUCCESS", "result": "not-an-object",
			}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.CreateContainer(ctx, &types.ContainerCreateRequest{Name: "x"}); err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("create: %v", err)
	}
}

// container.delete changed shape during the 26.0 cycle: 26.0-BETA.1 takes
// the id alone and returns directly, later builds take an options argument
// and run as a job. Both have to work, and the fallback must be narrow
// enough that a real validation failure is never retried as an arity
// problem.
func TestDeleteContainer_argumentShapeFallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("modern server takes options and runs as a job", func(t *testing.T) {
		var argCounts []int
		const jobID = int64(41)
		ts := NewTestServer(t, func(ctx context.Context, m string, p []interface{}) (interface{}, *RPCError) {
			switch m {
			case "container.delete":
				argCounts = append(argCounts, len(p))
				return jobID, nil
			case "core.get_jobs":
				return []interface{}{map[string]interface{}{"id": jobID, "state": "SUCCESS"}}, nil
			}
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		})
		c, _ := ts.NewClient(ctx)
		if err := c.DeleteContainer(ctx, 3, &types.ContainerDeleteOptions{Force: true}); err != nil {
			t.Fatalf("DeleteContainer: %v", err)
		}
		if len(argCounts) != 1 || argCounts[0] != 2 {
			t.Errorf("call shapes = %v, want a single 2-argument call", argCounts)
		}
	})

	t.Run("beta server rejects the options argument and gets the older form", func(t *testing.T) {
		var argCounts []int
		var polled bool
		ts := NewTestServer(t, func(ctx context.Context, m string, p []interface{}) (interface{}, *RPCError) {
			switch m {
			case "container.delete":
				argCounts = append(argCounts, len(p))
				if len(p) > 1 {
					return nil, &RPCError{
						Code:    CodeInvalidParams,
						Message: "Invalid params",
						Data:    []byte(`{"errname":"EINVAL","reason":"[EINVAL] : Too many arguments (expected 1, found 2)"}`),
					}
				}
				return nil, nil
			case "core.get_jobs":
				polled = true
				return []interface{}{}, nil
			}
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		})
		c, _ := ts.NewClient(ctx)
		if err := c.DeleteContainer(ctx, 3, &types.ContainerDeleteOptions{Force: true}); err != nil {
			t.Fatalf("DeleteContainer: %v", err)
		}
		if len(argCounts) != 2 || argCounts[0] != 2 || argCounts[1] != 1 {
			t.Errorf("call shapes = %v, want a 2-argument call then a 1-argument retry", argCounts)
		}
		// The single-argument form is not a job upstream, so polling it
		// would hang on a job id that was never issued.
		if polled {
			t.Error("the fallback polled for a job; the single-argument form returns directly")
		}
	})

	// A genuine validation failure must NOT be retried as an arity problem:
	// the retry would send a different call and report whatever it returned.
	t.Run("a real validation failure is not retried", func(t *testing.T) {
		var calls int
		ts := NewTestServer(t, func(ctx context.Context, m string, p []interface{}) (interface{}, *RPCError) {
			if m == "container.delete" {
				calls++
				return nil, &RPCError{
					Code:    CodeInvalidParams,
					Message: "Invalid params",
					Data:    []byte(`{"errname":"EINVAL","reason":"[EINVAL] id: container is running"}`),
				}
			}
			return nil, &RPCError{Code: CodeMethodNotFound, Message: m}
		})
		c, _ := ts.NewClient(ctx)
		if err := c.DeleteContainer(ctx, 3, nil); err == nil {
			t.Fatal("a validation failure was swallowed")
		}
		if calls != 1 {
			t.Errorf("container.delete called %d times; a validation failure must not trigger the arity fallback", calls)
		}
	})
}

func TestIsTooManyArguments(t *testing.T) {
	arity := &RPCError{Code: CodeInvalidParams, Message: "Invalid params",
		Data: []byte(`{"errname":"EINVAL","reason":"[EINVAL] : Too many arguments (expected 1, found 2)"}`)}

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"not an RPC error", context.Canceled, false},
		{"arity in the reason", arity, true},
		{"arity in the message", &RPCError{Code: CodeMethodCallError, Message: "Too many arguments"}, true},
		{"wrong code", &RPCError{Code: CodeMethodNotFound, Message: "Too many arguments"}, false},
		{"a different validation failure", &RPCError{Code: CodeInvalidParams, Message: "Invalid params",
			Data: []byte(`{"errname":"EINVAL","reason":"[EINVAL] name: already exists"}`)}, false},
		{"no data at all", &RPCError{Code: CodeInvalidParams, Message: "Invalid params"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTooManyArguments(tc.err); got != tc.want {
				t.Errorf("isTooManyArguments = %v, want %v", got, tc.want)
			}
		})
	}
}
