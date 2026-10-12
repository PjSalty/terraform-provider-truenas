package resources

import (
	"context"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// Updating an existing SMB config on a server older than 26.0.
//
// Each half of this was already tested and each half passed. Reading a server
// without search_protocols yields a known empty list, so plans show no phantom
// diff. The client refuses search_protocols against such a server. Put
// together they failed every update on 25.10: the empty list the read
// produced came back in the plan of the next change, and the client took it
// for a request for a feature the server does not have. Released that way in
// 3.0.0 and 3.1.0.
//
// The existing Update unit test could not see it, because it updates from an
// all-null plan, and a null search_protocols is never forwarded. So this
// drives the sequence Terraform actually runs: Read, change one unrelated
// attribute, Update.
func TestSMBConfigResource_UpdateAfterReadOnPre26Server(t *testing.T) {
	ctx := context.Background()

	var mu sync.Mutex
	var sent map[string]interface{}
	server := map[string]interface{}{
		"id": 1, "netbiosname": "TRUENAS", "workgroup": "WORKGROUP",
		"description": "initial", "enable_smb1": false, "unixcharset": "UTF-8",
		"aapl_extensions": false, "guest": "nobody", "filemask": "0775", "dirmask": "0775",
	}
	c := newWSTestClient(ctx, t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		mu.Lock()
		defer mu.Unlock()
		switch method {
		case "smb.config":
			return server, nil
		case "smb.update":
			body, _ := params[0].(map[string]interface{})
			// The pre-26.0 model forbids keys it does not know, which is how
			// the real server answers either of these.
			for _, key := range []string{"search_protocols", "minimum_protocol"} {
				if _, ok := body[key]; ok {
					return nil, &wsclient.RPCError{
						Code:    wsclient.CodeInvalidParams,
						Message: "smb_update." + key + ": Extra inputs are not permitted",
					}
				}
			}
			sent = body
			for k, v := range body {
				server[k] = v
			}
			return server, nil
		}
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})

	r := &SMBConfigResource{client: c}
	sch := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, sch)

	state := primedState(t, ctx, *sch)
	readResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}

	// The precondition the whole test rests on: state holds a known empty
	// list for a server that has no such field.
	var searchProtocols types.List
	if diags := readResp.State.GetAttribute(ctx, path.Root("search_protocols"), &searchProtocols); diags.HasError() {
		t.Fatalf("reading search_protocols from state: %v", diags)
	}
	if searchProtocols.IsNull() || searchProtocols.IsUnknown() || len(searchProtocols.Elements()) != 0 {
		t.Fatalf("precondition: want a known empty search_protocols in state, got %v", searchProtocols)
	}

	// The plan for "change the description": everything else is what Read
	// left in state.
	plan := tfsdk.Plan{Schema: sch.Schema, Raw: readResp.State.Raw}
	if diags := plan.SetAttribute(ctx, path.Root("description"), "updated"); diags.HasError() {
		t.Fatalf("building the plan: %v", diags)
	}

	updateResp := &resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: plan, State: readResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update of an unrelated attribute on a pre-26.0 server: %v", updateResp.Diagnostics)
	}

	mu.Lock()
	defer mu.Unlock()
	if sent == nil {
		t.Fatal("smb.update was never called")
	}
	if got := sent["description"]; got != "updated" {
		t.Errorf("description on the wire = %v, want %q", got, "updated")
	}
	var description types.String
	if diags := updateResp.State.GetAttribute(ctx, path.Root("description"), &description); diags.HasError() {
		t.Fatalf("reading description from state: %v", diags)
	}
	if description.ValueString() != "updated" {
		t.Errorf("description in state = %q, want %q", description.ValueString(), "updated")
	}
}
