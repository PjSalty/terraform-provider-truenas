package datasources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// newTestServer returns an httptest server and a Client pointed at it.
func newTestServer(t *testing.T, handler http.Handler) (*httptest.Server, *client.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := client.New(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("client.New: %v", err)
	}
	return srv, c
}

// writeJSON writes v as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// getDataSourceSchema fetches the Schema from a datasource.
func getDataSourceSchema(t *testing.T, ds datasource.DataSource) datasource.SchemaResponse {
	t.Helper()
	resp := datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema: %v", resp.Diagnostics)
	}
	return resp
}

// buildConfig constructs a tfsdk.Config from the datasource's schema using
// the provided attribute values (only the set fields; unset attributes become
// null). This lets us drive a real Read() call without the plugin protocol.
func buildConfig(t *testing.T, ds datasource.DataSource, values map[string]tftypes.Value) tfsdk.Config {
	t.Helper()
	schemaResp := getDataSourceSchema(t, ds)
	objType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)

	// Fill in any missing attributes with null values of the correct type.
	full := map[string]tftypes.Value{}
	for name, attrType := range objType.AttributeTypes {
		if v, ok := values[name]; ok {
			full[name] = v
			continue
		}
		full[name] = tftypes.NewValue(attrType, nil)
	}

	raw := tftypes.NewValue(objType, full)
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}
}

// callRead runs the datasource Read with the given config and returns the
// resulting ReadResponse.
func callRead(ctx context.Context, ds datasource.DataSource, cfg tfsdk.Config) *datasource.ReadResponse {
	resp := &datasource.ReadResponse{
		State: tfsdk.State{Schema: cfg.Schema},
	}
	ds.Read(ctx, datasource.ReadRequest{Config: cfg}, resp)
	return resp
}

// strVal is a helper for constructing a tftypes.Value of type String.
func strVal(s string) tftypes.Value {
	return tftypes.NewValue(tftypes.String, s)
}

// int64Val is a helper for constructing a tftypes.Value of type Number.
func int64Val(n int64) tftypes.Value {
	return tftypes.NewValue(tftypes.Number, n)
}
