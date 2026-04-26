package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestACMEDNSAuthenticatorResource_CRUD(t *testing.T) {
	ctx := context.Background()
	body := map[string]interface{}{
		"id":         7,
		"name":       "example",
		"attributes": map[string]interface{}{"authenticator": "cloudflare", "api_token": "tok"},
	}
	handler := func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			_, _ = w.Write([]byte("true"))
			return
		}
		_ = json.NewEncoder(w).Encode(body)
	}
	c, srv := newTestServerClient(t, handler)
	defer srv.Close()
	r := &ACMEDNSAuthenticatorResource{client: c}
	sch := schemaOf(t, ctx, r)

	vals := map[string]tftypes.Value{
		"name":          str("example"),
		"authenticator": str("cloudflare"),
		"attributes":    strMapNull(),
	}
	plan := planFromValues(t, ctx, sch, vals)

	// Create
	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", cResp.Diagnostics)
	}

	// Read
	state := stateFromValues(t, ctx, sch, map[string]tftypes.Value{"id": str("7")})
	rResp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, rResp)
	if rResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", rResp.Diagnostics)
	}

	// Update
	vals["id"] = str("7")
	stFull := stateFromValues(t, ctx, sch, vals)
	planU := planFromValues(t, ctx, sch, vals)
	uResp := &resource.UpdateResponse{State: stFull}
	r.Update(ctx, resource.UpdateRequest{State: stFull, Plan: planU}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", uResp.Diagnostics)
	}

	// Delete
	dResp := &resource.DeleteResponse{State: stFull}
	r.Delete(ctx, resource.DeleteRequest{State: stFull}, dResp)
	if dResp.Diagnostics.HasError() {
		t.Fatalf("Delete: %v", dResp.Diagnostics)
	}
}
