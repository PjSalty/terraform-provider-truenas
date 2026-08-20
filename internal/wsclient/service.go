package wsclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// JSON-RPC method namespace for system services:
// service.{query, get_instance, update, control}.
//
// service.start and service.stop were removed in TrueNAS 26.0. In 25.10.5
// they carried removed_in="v26.04"; on 26.0 they are @private, and a
// @private method is registered on no endpoint at all, so pinning an
// older API version does not bring them back. service.control is the
// replacement and exists in 25.10.5, 26.0 and 27.0 alike, so the single
// call path below is correct on every train we support.

// ListServices retrieves all services.
func (c *Client) ListServices(ctx context.Context) ([]types.Service, error) {
	tflog.Trace(ctx, "ListServices (ws) start")

	result, err := c.Call(ctx, "service.query", nil,
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	var services []types.Service
	if err := json.Unmarshal(result, &services); err != nil {
		return nil, fmt.Errorf("parsing services list: %w", err)
	}

	tflog.Trace(ctx, "ListServices (ws) success")
	return services, nil
}

// GetService retrieves a service by ID.
func (c *Client) GetService(ctx context.Context, id int) (*types.Service, error) {
	tflog.Trace(ctx, "GetService (ws) start")

	result, err := c.Call(ctx, "service.get_instance",
		[]interface{}{id},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("getting service %d: %w", id, err)
	}

	var svc types.Service
	if err := json.Unmarshal(result, &svc); err != nil {
		return nil, fmt.Errorf("parsing service response: %w", err)
	}

	tflog.Trace(ctx, "GetService (ws) success")
	return &svc, nil
}

// GetServiceByName uses server-side filtering on service.query for an
// O(1) lookup.
func (c *Client) GetServiceByName(ctx context.Context, name string) (*types.Service, error) {
	tflog.Trace(ctx, "GetServiceByName (ws) start")

	filters := []interface{}{[]interface{}{"service", "=", name}}
	result, err := c.Call(ctx, "service.query",
		[]interface{}{filters},
		CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	var services []types.Service
	if err := json.Unmarshal(result, &services); err != nil {
		return nil, fmt.Errorf("parsing services list: %w", err)
	}

	if len(services) == 0 {
		return nil, &RPCError{
			Code:    CodeMethodNotFound,
			Message: fmt.Sprintf("service %q not found", name),
		}
	}

	tflog.Trace(ctx, "GetServiceByName (ws) success")
	return &services[0], nil
}

// UpdateService updates a service's enable flag.
func (c *Client) UpdateService(ctx context.Context, id int, req *types.ServiceUpdateRequest) error {
	tflog.Trace(ctx, "UpdateService (ws) start")

	if _, err := c.Call(ctx, "service.update",
		[]interface{}{id, req},
		CallOptions{Idempotent: false}); err != nil {
		return fmt.Errorf("updating service %d: %w", id, err)
	}

	tflog.Trace(ctx, "UpdateService (ws) success")
	return nil
}

// Verbs accepted by service.control. RESTART and RELOAD are part of the
// same middleware enum and are declared here so a future caller does not
// reintroduce a bare string.
const (
	serviceVerbStart = "START"
	serviceVerbStop  = "STOP"
)

// serviceControlOptions is middleware's ServiceOptions. Only silent is
// set; ha_propagate and timeout keep their server-side defaults.
//
// silent defaults to TRUE on the server, which makes service.control
// answer a failed operation with a SUCCESSFUL job whose result is false,
// rather than with an error. The old service.start path ignored that
// bool, so a service that never came up still produced a clean apply.
// Asking for silent=false makes the failure arrive as a real error
// carrying middleware's own diagnostic text.
type serviceControlOptions struct {
	Silent bool `json:"silent"`
}

// legacyServiceMethod maps a control verb onto the pre-25.10 method it
// replaced. Only START and STOP are mapped because they are the only
// verbs this client emits.
func legacyServiceMethod(verb string) string {
	if verb == serviceVerbStop {
		return "service.stop"
	}
	return "service.start"
}

// controlService issues service.control and reports the resulting state,
// falling back to the pre-25.10 methods when the server is too old.
//
// service.control is decorated @job in middleware, so it answers with a
// job id rather than the bool. It has to go through CallJob: a plain Call
// would hand back the job id, which decodes into a non-zero value that
// reads as "it worked" no matter what the service actually did.
//
// Version coverage, which is why the fallback exists:
//
//	25.04 and older : service.start/stop only, NO service.control
//	25.10.0 - 25.10.x: both (start/stop marked removed_in="v26.04")
//	26.0 and newer  : service.control only, start/stop are @private
//
// So neither method alone covers the range the provider supports. Trying
// control first means every currently supported release takes the modern
// path, and only a 25.04 host ever falls back.
func (c *Client) controlService(ctx context.Context, verb, name string) (bool, error) {
	result, err := c.CallJob(ctx, "service.control",
		[]interface{}{verb, name, serviceControlOptions{Silent: false}},
		CallOptions{Idempotent: false}, 0)
	if err != nil {
		// Fall back ONLY when middleware says the method itself is
		// unknown. Any other failure is a real failure and must
		// surface: retrying a genuine error against a second method
		// would report whatever that one happened to return.
		if isMethodUnknown(err) {
			tflog.Debug(ctx, "service.control not available, falling back to the pre-25.10 method",
				map[string]interface{}{"verb": verb, "service": name})
			return c.controlServiceLegacy(ctx, verb, name)
		}
		return false, err
	}
	return decodeServiceControlResult(result, "service.control", verb, name)
}

// controlServiceLegacy drives service.start / service.stop, for TrueNAS
// 25.04 and older where service.control does not exist.
//
// These are NOT jobs, so they answer with the bool directly. They carry
// the same silent option and the same default, so silent=false is set
// here too: 25.04's docstring is explicit that a true silent turns a
// startup failure into a false return rather than an exception.
func (c *Client) controlServiceLegacy(ctx context.Context, verb, name string) (bool, error) {
	method := legacyServiceMethod(verb)
	result, err := c.Call(ctx, method,
		[]interface{}{name, serviceControlOptions{Silent: false}},
		CallOptions{Idempotent: false})
	if err != nil {
		return false, err
	}
	return decodeServiceControlResult(result, method, verb, name)
}

// decodeServiceControlResult reads the documented bool. With silent=false
// a failure normally raises, but a server that still answers false must
// not be mistaken for success.
func decodeServiceControlResult(result json.RawMessage, method, verb, name string) (bool, error) {
	var ok bool
	if err := json.Unmarshal(result, &ok); err != nil {
		return false, fmt.Errorf("%s %s %q: parsing result %s: %w", method, verb, name, string(result), err)
	}
	return ok, nil
}

// StartService starts a service by name.
func (c *Client) StartService(ctx context.Context, name string) error {
	tflog.Trace(ctx, "StartService (ws) start")

	running, err := c.controlService(ctx, serviceVerbStart, name)
	if err != nil {
		return fmt.Errorf("starting service %q: %w", name, err)
	}
	if !running {
		return fmt.Errorf("starting service %q: TrueNAS reports it is not running after START", name)
	}

	tflog.Trace(ctx, "StartService (ws) success")
	return nil
}

// StopService stops a service by name.
func (c *Client) StopService(ctx context.Context, name string) error {
	tflog.Trace(ctx, "StopService (ws) start")

	stopped, err := c.controlService(ctx, serviceVerbStop, name)
	if err != nil {
		return fmt.Errorf("stopping service %q: %w", name, err)
	}
	if !stopped {
		return fmt.Errorf("stopping service %q: TrueNAS reports it was not stopped", name)
	}

	tflog.Trace(ctx, "StopService (ws) success")
	return nil
}
