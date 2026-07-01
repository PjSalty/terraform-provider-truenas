package resources

// Repro for issue #15 / PR #23: configuring an existing PHYSICAL network
// interface. On released main, NetworkInterfaceResource.Create sends
// interface.create for type=PHYSICAL, which TrueNAS rejects (physical
// NICs are host-discovered, not creatable). PR #23 routes PHYSICAL
// Create through interface.update instead, and makes Delete a
// state-only removal that never touches the hardware interface.
//
// The mock server RECORDS every JSON-RPC method it receives so the
// assertions are about which methods were invoked, not just about
// diagnostics being empty.

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

type repro15Recorder struct {
	mu      sync.Mutex
	methods []string
}

func (r *repro15Recorder) record(m string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.methods = append(r.methods, m)
}

func (r *repro15Recorder) saw(m string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, got := range r.methods {
		if got == m {
			return true
		}
	}
	return false
}

func repro15Body() map[string]interface{} {
	return map[string]interface{}{
		"id":                       "eth0",
		"name":                     "eth0",
		"type":                     "PHYSICAL",
		"description":              "mgmt",
		"ipv4_dhcp":                true,
		"ipv6_auto":                false,
		"mtu":                      1500,
		"state":                    map[string]interface{}{"name": "eth0"},
		"aliases":                  []interface{}{},
		"failover_aliases":         []interface{}{},
		"failover_virtual_aliases": []interface{}{},
		"bridge_members":           []interface{}{},
		"lag_protocol":             "",
		"lag_ports":                []interface{}{},
		"vlan_parent_interface":    "",
		"vlan_tag":                 nil,
		"vlan_pcp":                 nil,
	}
}

// newRepro15Client builds a client against a recording mock. When
// updateErr is non-nil, interface.update fails with it (simulates a
// typo'd NIC name). interface.create ALWAYS fails the way a real
// TrueNAS does for PHYSICAL: EINVAL invalid choice.
func newRepro15Client(t *testing.T, updateErr *wsclient.RPCError) (*wsclient.Client, *repro15Recorder) {
	t.Helper()
	rec := &repro15Recorder{}
	body := repro15Body()
	ts := wsclient.NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		rec.record(method)
		switch method {
		case "interface.update":
			if updateErr != nil {
				return nil, updateErr
			}
			return body, nil
		case "interface.create":
			return nil, &wsclient.RPCError{
				Code:    wsclient.CodeMethodCallError,
				Message: "Method call error",
				Data:    json.RawMessage(`{"errname":"EINVAL","reason":"[EINVAL] interface_create.type: Invalid choice: PHYSICAL"}`),
			}
		case "interface.commit", "interface.checkin":
			return nil, nil
		case "interface.get_instance", "interface.query":
			return body, nil
		case "interface.delete":
			return true, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(context.Background())
	if err != nil {
		t.Fatalf("testserver NewClient: %v", err)
	}
	return c, rec
}

// TestRepro15_PhysicalCreateUsesUpdate is the core repro: Create of a
// type=PHYSICAL interface must succeed by invoking interface.update
// and must never invoke interface.create.
//
// On released main this fails twice over: interface.create IS invoked
// and its EINVAL error becomes an error diagnostic.
func TestRepro15_PhysicalCreateUsesUpdate(t *testing.T) {
	c, rec := newRepro15Client(t, nil)
	r := &NetworkInterfaceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name":      str("eth0"),
		"type":      str("PHYSICAL"),
		"ipv4_dhcp": flag(true),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)

	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create(PHYSICAL) returned error diagnostics: %v", cResp.Diagnostics)
	}
	if rec.saw("interface.create") {
		t.Error("Create(PHYSICAL) invoked interface.create; must configure the existing NIC via interface.update")
	}
	if !rec.saw("interface.update") {
		t.Error("Create(PHYSICAL) never invoked interface.update; plan settings were not applied")
	}
}

// TestRepro15_PhysicalDeleteLeavesHardwareAlone: Delete of a PHYSICAL
// interface must be a state-only removal. It must not call
// interface.delete (you cannot delete hardware) and, per the PR's
// documented behavior, does not touch the interface at all.
func TestRepro15_PhysicalDeleteLeavesHardwareAlone(t *testing.T) {
	c, rec := newRepro15Client(t, nil)
	r := &NetworkInterfaceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
		"id":   str("eth0"),
		"name": str("eth0"),
		"type": str("PHYSICAL"),
	})
	dResp := &resource.DeleteResponse{State: state}
	r.Delete(ctx, resource.DeleteRequest{State: state}, dResp)

	if dResp.Diagnostics.HasError() {
		t.Fatalf("Delete(PHYSICAL) returned error diagnostics: %v", dResp.Diagnostics)
	}
	if rec.saw("interface.delete") {
		t.Error("Delete(PHYSICAL) invoked interface.delete against a hardware interface")
	}
	if rec.saw("interface.update") {
		t.Error("Delete(PHYSICAL) invoked interface.update; documented behavior is state-only removal")
	}
	if rec.saw("interface.commit") {
		t.Error("Delete(PHYSICAL) invoked interface.commit; nothing should be staged")
	}
}

// TestRepro15_PhysicalCreateUnknownNameSurfacesError: a typo'd NIC name
// must surface as an error diagnostic (the patched code maps the
// server's ENOENT to a dedicated "Could not find PHYSICAL Interface"
// diagnostic), never silently succeed.
func TestRepro15_PhysicalCreateUnknownNameSurfacesError(t *testing.T) {
	notFound := &wsclient.RPCError{
		Code:    wsclient.CodeMethodCallError,
		Message: "Method call error",
		Data:    json.RawMessage(`{"errname":"ENOENT","reason":"[ENOENT] Interface eth9 does not exist"}`),
	}
	c, rec := newRepro15Client(t, notFound)
	r := &NetworkInterfaceResource{client: c}
	ctx := context.Background()
	sch := schemaOf(t, ctx, r)

	plan := planFromValues(t, ctx, sch, map[string]tftypes.Value{
		"name": str("eth9"),
		"type": str("PHYSICAL"),
	})
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)

	if !cResp.Diagnostics.HasError() {
		t.Fatal("Create(PHYSICAL, unknown name) produced no error diagnostics; server ENOENT was swallowed")
	}
	sawNotFoundDiag := false
	for _, d := range cResp.Diagnostics.Errors() {
		if strings.Contains(d.Summary(), "Could not find PHYSICAL Interface") {
			sawNotFoundDiag = true
		}
	}
	if !sawNotFoundDiag {
		t.Errorf("expected dedicated not-found diagnostic, got: %v", cResp.Diagnostics)
	}
	if rec.saw("interface.commit") {
		t.Error("commit staged after failed update of unknown interface")
	}
}
