package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// providerConfigValues is a convenience map of the provider attributes.
// Unset fields default to null via the zero value of tftypes.Value, which
// buildProviderConfig coerces to an explicit null for each attribute so
// the framework treats them as unset.
type providerConfigValues struct {
	url                tftypes.Value
	apiKey             tftypes.Value
	insecureSkipVerify tftypes.Value
	readOnly           tftypes.Value
	destroyProtection  tftypes.Value
	requestTimeout     tftypes.Value
}

// buildProviderConfig constructs a tfsdk.Config for the provider schema,
// populating attributes from the given values (use null values to leave
// attributes unset).
func buildProviderConfig(t *testing.T, p provider.Provider, v providerConfigValues) tfsdk.Config {
	t.Helper()
	schemaResp := provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("provider Schema returned diagnostics: %v", schemaResp.Diagnostics)
	}
	objType := schemaResp.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
	readOnlyVal := v.readOnly
	if readOnlyVal.Type() == nil {
		readOnlyVal = nullBool()
	}
	destroyProtectionVal := v.destroyProtection
	if destroyProtectionVal.Type() == nil {
		destroyProtectionVal = nullBool()
	}
	requestTimeoutVal := v.requestTimeout
	if requestTimeoutVal.Type() == nil {
		requestTimeoutVal = nullString()
	}
	raw := tftypes.NewValue(objType, map[string]tftypes.Value{
		"url":                  v.url,
		"api_key":              v.apiKey,
		"insecure_skip_verify": v.insecureSkipVerify,
		"read_only":            readOnlyVal,
		"destroy_protection":   destroyProtectionVal,
		"request_timeout":      requestTimeoutVal,
	})
	return tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}
}

func nullString() tftypes.Value { return tftypes.NewValue(tftypes.String, nil) }
func nullBool() tftypes.Value   { return tftypes.NewValue(tftypes.Bool, nil) }

func TestProvider_Metadata(t *testing.T) {
	p := New("1.2.3")()
	resp := provider.MetadataResponse{}
	p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)
	if resp.TypeName != "truenas" {
		t.Errorf("TypeName = %q, want truenas", resp.TypeName)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", resp.Version)
	}
}

func TestProvider_Schema(t *testing.T) {
	p := New("test")()
	resp := provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema diagnostics: %v", resp.Diagnostics)
	}
	if resp.Schema.Description == "" {
		t.Error("Schema description should not be empty")
	}
	for _, name := range []string{"url", "api_key", "insecure_skip_verify", "read_only", "destroy_protection", "request_timeout"} {
		if _, ok := resp.Schema.Attributes[name]; !ok {
			t.Errorf("Schema missing attribute %q", name)
		}
	}
}

func TestProvider_DataSources(t *testing.T) {
	p := New("test")()
	tp, ok := p.(*TrueNASProvider)
	if !ok {
		t.Fatalf("expected *TrueNASProvider, got %T", p)
	}
	ds := tp.DataSources(context.Background())
	if len(ds) == 0 {
		t.Error("DataSources returned empty slice")
	}
	// Each factory should return a non-nil datasource.
	for i, fn := range ds {
		if fn() == nil {
			t.Errorf("DataSources[%d] returned nil", i)
		}
	}
}

func TestProvider_Resources(t *testing.T) {
	p := New("test")()
	tp := p.(*TrueNASProvider)
	rs := tp.Resources(context.Background())
	if len(rs) == 0 {
		t.Error("Resources returned empty slice")
	}
}

// TestProvider_Configure_FromConfig supplies url/api_key/insecure_skip_verify
// via the provider config and expects a successful client creation.
func TestProvider_Configure_FromConfig(t *testing.T) {
	// Clear env so config values are authoritative.
	t.Setenv("TRUENAS_URL", "")
	t.Setenv("TRUENAS_API_KEY", "")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                tftypes.NewValue(tftypes.String, "https://truenas.example.com"),
		apiKey:             tftypes.NewValue(tftypes.String, "test-api-key"),
		insecureSkipVerify: tftypes.NewValue(tftypes.Bool, true),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure diagnostics: %v", resp.Diagnostics)
	}
	if resp.DataSourceData == nil || resp.ResourceData == nil {
		t.Error("Configure did not set DataSourceData/ResourceData")
	}
	if _, ok := resp.DataSourceData.(*client.Client); !ok {
		t.Errorf("DataSourceData is not *client.Client, got %T", resp.DataSourceData)
	}
}

// TestProvider_Configure_FromEnv leaves config null and supplies credentials
// via environment variables, also covering the TRUENAS_INSECURE_SKIP_VERIFY
// env parse branch.
func TestProvider_Configure_FromEnv(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://env.example.com")
	t.Setenv("TRUENAS_API_KEY", "env-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Configure diagnostics: %v", resp.Diagnostics)
	}
	if resp.DataSourceData == nil {
		t.Error("DataSourceData is nil")
	}
}

// TestProvider_Configure_MissingURL covers the "missing URL" diagnostic path.
func TestProvider_Configure_MissingURL(t *testing.T) {
	t.Setenv("TRUENAS_URL", "")
	t.Setenv("TRUENAS_API_KEY", "some-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected diagnostics error for missing URL")
	}
}

// TestProvider_Configure_MissingAPIKey covers the "missing API key" branch.
func TestProvider_Configure_MissingAPIKey(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://truenas.example.com")
	t.Setenv("TRUENAS_API_KEY", "")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected diagnostics error for missing API key")
	}
}

// TestProvider_Configure_BadConfig passes a config whose raw type does not
// match the provider schema, forcing req.Config.Get to return diagnostics and
// exercising the early-return branch in Configure.
func TestProvider_Configure_BadConfig(t *testing.T) {
	t.Setenv("TRUENAS_URL", "")
	t.Setenv("TRUENAS_API_KEY", "")

	p := New("test")()
	schemaResp := provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, &schemaResp)

	// Build a raw value with api_key as a Bool instead of String — this
	// produces a type-mismatch error from req.Config.Get.
	badType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"url":                  tftypes.String,
			"api_key":              tftypes.Bool, // wrong type
			"insecure_skip_verify": tftypes.Bool,
		},
	}
	raw := tftypes.NewValue(badType, map[string]tftypes.Value{
		"url":                  tftypes.NewValue(tftypes.String, "https://truenas.example.com"),
		"api_key":              tftypes.NewValue(tftypes.Bool, true),
		"insecure_skip_verify": tftypes.NewValue(tftypes.Bool, nil),
	})
	cfg := tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected diagnostics error from malformed config")
	}
}

// TestProvider_Configure_ClientError swaps newClientFn to force an error and
// exercises the client-construction error-handling branch in Configure.
func TestProvider_Configure_ClientError(t *testing.T) {
	t.Setenv("TRUENAS_URL", "")
	t.Setenv("TRUENAS_API_KEY", "")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")

	original := newClientFn
	t.Cleanup(func() { newClientFn = original })
	newClientFn = func(baseURL, apiKey string, insecure bool) (*client.Client, error) {
		return nil, fmt.Errorf("forced client failure")
	}

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                tftypes.NewValue(tftypes.String, "https://truenas.example.com"),
		apiKey:             tftypes.NewValue(tftypes.String, "key"),
		insecureSkipVerify: nullBool(),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("expected diagnostics error from forced client failure")
	}
	if resp.DataSourceData != nil {
		t.Error("DataSourceData should not be set when client creation fails")
	}
}
