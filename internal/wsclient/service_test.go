package wsclient

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func TestListServices(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.query" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return []interface{}{
			map[string]interface{}{"id": 1, "service": "smb", "enable": true, "state": "RUNNING"},
		}, nil
	})
	c, _ := ts.NewClient(ctx)
	svcs, err := c.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices: %v", err)
	}
	if len(svcs) != 1 || svcs[0].Service != "smb" {
		t.Errorf("got %+v", svcs)
	}
}

func TestListServices_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListServices(ctx)
	if err == nil || !strings.Contains(err.Error(), "listing services") {
		t.Errorf("got %v", err)
	}
}

func TestListServices_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.ListServices(ctx)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.get_instance" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"id": 5, "service": "ssh", "enable": true}, nil
	})
	c, _ := ts.NewClient(ctx)
	svc, err := c.GetService(ctx, 5)
	if err != nil {
		t.Fatalf("GetService: %v", err)
	}
	if svc.Service != "ssh" {
		t.Errorf("got %+v", svc)
	}
}

func TestGetService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetService(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "getting service") {
		t.Errorf("got %v", err)
	}
}

func TestGetService_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetService(ctx, 1)
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestGetServiceByName_found(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{map[string]interface{}{"id": 7, "service": "smb"}}, nil
	})
	c, _ := ts.NewClient(ctx)
	svc, err := c.GetServiceByName(ctx, "smb")
	if err != nil {
		t.Fatalf("GetServiceByName: %v", err)
	}
	if svc.ID != 7 {
		t.Errorf("got %+v", svc)
	}
}

func TestGetServiceByName_notFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return []interface{}{}, nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetServiceByName(ctx, "missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("got %v", err)
	}
}

func TestGetServiceByName_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetServiceByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "listing services") {
		t.Errorf("got %v", err)
	}
}

func TestGetServiceByName_decodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return "not-an-array", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.GetServiceByName(ctx, "x")
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "service.update" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.UpdateService(ctx, 5, &types.ServiceUpdateRequest{Enable: true}); err != nil {
		t.Errorf("UpdateService: %v", err)
	}
}

func TestUpdateService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.UpdateService(ctx, 5, &types.ServiceUpdateRequest{})
	if err == nil || !strings.Contains(err.Error(), "updating service") {
		t.Errorf("got %v", err)
	}
}

// serviceControlServer fakes the job-bound service.control.
//
// It records every method it was asked for, so a test can assert that
// service.start / service.stop are never emitted again: those are
// @private on TrueNAS 26 and calling them returns method-not-found.
// jobResult is what core.get_jobs reports as the job's result.
type serviceControlRecorder struct {
	methods []string
	params  []interface{}
}

func serviceControlServer(t *testing.T, rec *serviceControlRecorder, jobResult interface{}, jobError string) *TestServer {
	t.Helper()
	const jobID = int64(77)
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		rec.methods = append(rec.methods, method)
		switch method {
		case "service.control":
			rec.params = params
			return jobID, nil
		case "core.get_jobs":
			state := "SUCCESS"
			if jobError != "" {
				state = "FAILED"
			}
			return []interface{}{map[string]interface{}{
				"id": jobID, "state": state, "error": jobError, "result": jobResult,
			}}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
}

func TestStartService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := serviceControlServer(t, rec, true, "")
	c, _ := ts.NewClient(ctx)
	if err := c.StartService(ctx, "smb"); err != nil {
		t.Fatalf("StartService: %v", err)
	}

	// The removed methods must never be emitted.
	for _, m := range rec.methods {
		if m == "service.start" || m == "service.stop" {
			t.Fatalf("emitted %q, which is @private on TrueNAS 26", m)
		}
	}
	// It must have gone through the job path, not a plain Call. Without
	// CallJob the job id decodes as the result and every failure reads
	// as success.
	if !containsMethod(rec.methods, "core.get_jobs") {
		t.Error("service.control was not polled as a job; CallJob is required")
	}
	// verb first, service second -- middleware's control(verb, service, options).
	if len(rec.params) < 2 {
		t.Fatalf("params = %v, want at least [verb, service]", rec.params)
	}
	if rec.params[0] != "START" {
		t.Errorf("verb = %v, want START", rec.params[0])
	}
	if rec.params[1] != "smb" {
		t.Errorf("service = %v, want smb", rec.params[1])
	}
	// silent=false, or a failure comes back as a successful job holding false.
	opts, ok := rec.params[2].(map[string]interface{})
	if !ok {
		t.Fatalf("options = %#v, want an object", rec.params[2])
	}
	if opts["silent"] != false {
		t.Errorf("silent = %v, want false", opts["silent"])
	}
}

func TestStopService(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := serviceControlServer(t, rec, true, "")
	c, _ := ts.NewClient(ctx)
	if err := c.StopService(ctx, "smb"); err != nil {
		t.Fatalf("StopService: %v", err)
	}
	if len(rec.params) < 2 || rec.params[0] != "STOP" {
		t.Errorf("verb = %v, want STOP", rec.params)
	}
}

// A false result must be an error, not a silent success. This is the
// regression that the old service.start path shipped: middleware
// defaults options.silent to true, answers a failed start with a
// SUCCESSFUL job whose result is false, and the bool was discarded --
// so a service that never came up still produced a clean apply.
func TestStartService_falseResultIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := serviceControlServer(t, rec, false, "")
	c, _ := ts.NewClient(ctx)
	err := c.StartService(ctx, "smb")
	if err == nil {
		t.Fatal("a false result reported success; a service that did not start must fail the apply")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("got %v, want it to say the service is not running", err)
	}
}

func TestStopService_falseResultIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := serviceControlServer(t, rec, false, "")
	c, _ := ts.NewClient(ctx)
	if err := c.StopService(ctx, "smb"); err == nil {
		t.Fatal("a false result reported success")
	}
}

// A failed job keeps the caller-facing prefix, so existing error
// handling and docs stay accurate.
func TestStartService_jobFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := serviceControlServer(t, rec, nil, "smb failed configuration check")
	c, _ := ts.NewClient(ctx)
	err := c.StartService(ctx, "smb")
	if err == nil || !strings.Contains(err.Error(), "starting service") {
		t.Errorf("got %v, want it prefixed with 'starting service'", err)
	}
}

func TestStartService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StartService(ctx, "smb")
	if err == nil || !strings.Contains(err.Error(), "starting service") {
		t.Errorf("got %v", err)
	}
}

func TestStopService_serverError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StopService(ctx, "smb")
	if err == nil || !strings.Contains(err.Error(), "stopping service") {
		t.Errorf("got %v", err)
	}
}

// --- pre-25.10 fallback ---
//
// service.control does not exist before 25.10.0, and service.start/stop
// do not exist from 26.0. Neither method alone covers the supported
// range, so control is tried first and only a method-not-found falls
// back.

func TestStartService_fallsBackWhenControlUnknown(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	var legacyParams []interface{}
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		rec.methods = append(rec.methods, method)
		switch method {
		case "service.control":
			// What a 25.04 host answers.
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "service.control"}
		case "service.start":
			legacyParams = params
			return true, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StartService(ctx, "smb"); err != nil {
		t.Fatalf("StartService on a pre-25.10 host: %v", err)
	}
	if !containsMethod(rec.methods, "service.start") {
		t.Fatalf("did not fall back; methods were %v", rec.methods)
	}
	// Legacy takes (service, options) -- no verb. Passing the control
	// argument order here would set the service name to "START".
	if len(legacyParams) < 1 || legacyParams[0] != "smb" {
		t.Errorf("legacy params = %v, want service name first", legacyParams)
	}
	opts, ok := legacyParams[1].(map[string]interface{})
	if !ok || opts["silent"] != false {
		t.Errorf("legacy options = %v, want silent=false", legacyParams[1])
	}
}

func TestStopService_fallbackUsesStopNotStart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		rec.methods = append(rec.methods, method)
		switch method {
		case "service.control":
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "service.control"}
		case "service.stop":
			return true, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StopService(ctx, "smb"); err != nil {
		t.Fatalf("StopService on a pre-25.10 host: %v", err)
	}
	if containsMethod(rec.methods, "service.start") {
		t.Error("STOP fell back to service.start; that would leave the service running")
	}
	if !containsMethod(rec.methods, "service.stop") {
		t.Errorf("did not call service.stop; methods were %v", rec.methods)
	}
}

// A real failure must NOT be retried against the legacy method. Doing so
// would report whatever the second call returned and mask the original
// error -- e.g. a permission denial coming back as a clean apply.
func TestStartService_realErrorDoesNotFallBack(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		rec.methods = append(rec.methods, method)
		if method == "service.control" {
			return nil, &RPCError{Code: CodeMethodCallError, Message: "[EPERM] lacks privilege"}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StartService(ctx, "smb"); err == nil {
		t.Fatal("a permission error was swallowed")
	}
	if containsMethod(rec.methods, "service.start") {
		t.Error("fell back on a non-method-not-found error; the real failure would be masked")
	}
}

// A MISSING SERVICE is not a missing METHOD. IsNotFound treats an ENOENT
// as not-found, which is right for a Read path and wrong here: keying the
// fallback on it would retry a nonexistent service against the legacy
// method and report that call's answer instead of the real error.
func TestStartService_enoentDoesNotFallBack(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		rec.methods = append(rec.methods, method)
		if method == "service.control" {
			return nil, &RPCError{Code: CodeMethodCallError, Message: "[ENOENT] nosuchservice does not exist"}
		}
		return true, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StartService(ctx, "nosuchservice"); err == nil {
		t.Fatal("a missing service reported success")
	}
	if containsMethod(rec.methods, "service.start") {
		t.Error("fell back on ENOENT; a missing service is not a missing method")
	}
}

// The legacy path carries the same silent-default trap as the modern one.
func TestStartService_legacyFalseResultIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "service.control" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "service.control"}
		}
		return false, nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StartService(ctx, "smb"); err == nil {
		t.Fatal("legacy false result reported success")
	}
}

// A server that answers with something other than the documented bool
// must be an error. Silently treating an unexpected shape as success is
// the same failure mode as discarding the bool.
func TestStartService_nonBooleanResultIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rec := &serviceControlRecorder{}
	ts := serviceControlServer(t, rec, map[string]interface{}{"unexpected": "shape"}, "")
	c, _ := ts.NewClient(ctx)
	err := c.StartService(ctx, "smb")
	if err == nil {
		t.Fatal("a non-boolean result reported success")
	}
	if !strings.Contains(err.Error(), "parsing result") {
		t.Errorf("got %v, want a parse diagnostic naming the payload", err)
	}
}

// The legacy path's own transport failure must surface, not be swallowed
// or mistaken for a false result.
func TestStartService_legacyCallErrorSurfaces(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "service.control" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "service.control"}
		}
		return nil, &RPCError{Code: CodeInternalError, Message: "legacy boom"}
	})
	c, _ := ts.NewClient(ctx)
	err := c.StartService(ctx, "smb")
	if err == nil || !strings.Contains(err.Error(), "starting service") {
		t.Errorf("got %v, want the legacy failure surfaced", err)
	}
}

func TestStopService_legacyNonBooleanResultIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "service.control" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: "service.control"}
		}
		return "not a bool", nil
	})
	c, _ := ts.NewClient(ctx)
	if err := c.StopService(ctx, "smb"); err == nil {
		t.Fatal("a non-boolean legacy result reported success")
	}
}

// isMethodUnknown decides whether the version fallback fires, so its
// non-RPC inputs matter: a nil or a plain error must never be read as
// "this TrueNAS is too old".
func TestIsMethodUnknown(t *testing.T) {
	if isMethodUnknown(nil) {
		t.Error("nil was treated as method-unknown")
	}
	if isMethodUnknown(errors.New("dial tcp: connection refused")) {
		t.Error("a transport error was treated as method-unknown; the fallback would fire on a network blip")
	}
	if isMethodUnknown(&RPCError{Code: CodeInternalError, Message: "boom"}) {
		t.Error("an internal error was treated as method-unknown")
	}
	if !isMethodUnknown(&RPCError{Code: CodeMethodNotFound, Message: "service.control"}) {
		t.Error("a genuine method-not-found was not recognised")
	}
}

func containsMethod(methods []string, want string) bool {
	for _, m := range methods {
		if m == want {
			return true
		}
	}
	return false
}
