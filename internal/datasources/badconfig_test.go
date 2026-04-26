package datasources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// badConfig returns a tfsdk.Config whose schema is the datasource's real
// schema but whose Raw value has an extra unknown attribute. Any attempt to
// unmarshal this Raw into a real *DataSourceModel struct fails, forcing the
// `if resp.Diagnostics.HasError() { return }` branch after the initial
// req.Config.Get() to execute.
func badConfig(t *testing.T, ds datasource.DataSource) tfsdk.Config {
	t.Helper()
	schemaResp := getDataSourceSchema(t, ds)

	// Build a throwaway schema that contains a single attribute of the wrong
	// type ("__bad"), so Config.Get reflects into a struct that doesn't have
	// it. We keep the ds's real schema on the Config so validation passes
	// inside buildConfig, but we use an unrelated Raw that is a different
	// object shape.
	badSchema := dsschema.Schema{
		Attributes: map[string]dsschema.Attribute{
			"__bad": dsschema.StringAttribute{Optional: true},
		},
	}
	badObjType := badSchema.Type().TerraformType(context.Background()).(tftypes.Object)
	raw := tftypes.NewValue(badObjType, map[string]tftypes.Value{
		"__bad": tftypes.NewValue(tftypes.String, "x"),
	})

	// The Config uses the REAL schema but a mismatched Raw. The framework's
	// Get() will fail with a type mismatch, hitting the early-return branch.
	_ = badObjType
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}
}

// dataSourcesWithConfigGet is the subset of data sources whose Read() begins
// with a `req.Config.Get(ctx, &config)` followed immediately by a
// `if resp.Diagnostics.HasError() { return }` check. Other data sources do
// not consult the config and instead query the API directly, so there is no
// early-return branch to exercise.
func dataSourcesWithConfigGet() []dataSourceCase {
	include := map[string]bool{
		"app":                  true,
		"certificate":          true,
		"cloudsync_credential": true,
		"cronjob":              true,
		"dataset":              true,
		"datasets":             true,
		"disk":                 true,
		"group":                true,
		"kerberos_realm":       true,
		"network_interface":    true,
		"pool":                 true,
		"privilege":            true,
		"service":              true,
		"share_nfs":            true,
		"share_smb":            true,
		"user":                 true,
		"vm":                   true,
	}
	var out []dataSourceCase
	for _, tc := range allDataSources() {
		if include[tc.name] {
			out = append(out, tc)
		}
	}
	return out
}

// TestAllDataSources_Read_BadConfig exercises the early-return branch that
// fires when req.Config.Get fails to unmarshal the Raw value.
func TestAllDataSources_Read_BadConfig(t *testing.T) {
	for _, tc := range dataSourcesWithConfigGet() {
		t.Run(tc.name, func(t *testing.T) {
			ds := tc.newFn()
			// No client needed — Read must return before any API call.
			cfg := badConfig(t, ds)
			resp := callRead(context.Background(), ds, cfg)
			if !resp.Diagnostics.HasError() {
				t.Errorf("expected error diagnostic from bad config")
			}
		})
	}
}
