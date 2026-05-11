package provider

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestProvider_Configure_RequestTimeoutEnvVar verifies that
// TRUENAS_REQUEST_TIMEOUT=<duration> propagates to the httpClient.
func TestProvider_Configure_RequestTimeoutEnvVar(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://example.com")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_REQUEST_TIMEOUT", "3m")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
	})
	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c := resp.DataSourceData.(*client.Client)
	if got := c.RequestTimeout(); got != 3*time.Minute {
		t.Errorf("RequestTimeout = %s, want 3m", got)
	}
}

// TestProvider_Configure_RequestTimeoutHCL verifies the HCL attribute path.
func TestProvider_Configure_RequestTimeoutHCL(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://example.com")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_REQUEST_TIMEOUT", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		requestTimeout:     tftypes.NewValue(tftypes.String, "7m"),
	})
	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c := resp.DataSourceData.(*client.Client)
	if got := c.RequestTimeout(); got != 7*time.Minute {
		t.Errorf("RequestTimeout = %s, want 7m", got)
	}
}

// TestProvider_Configure_RequestTimeoutHCLOverridesEnv asserts precedence.
func TestProvider_Configure_RequestTimeoutHCLOverridesEnv(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://example.com")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_REQUEST_TIMEOUT", "90s")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		requestTimeout:     tftypes.NewValue(tftypes.String, "10m"),
	})
	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c := resp.DataSourceData.(*client.Client)
	if got := c.RequestTimeout(); got != 10*time.Minute {
		t.Errorf("RequestTimeout = %s, want 10m (HCL must beat env)", got)
	}
}

// TestProvider_Configure_RequestTimeoutInvalidDuration asserts that a
// malformed duration surfaces as a provider diagnostic rather than
// silently being ignored.
func TestProvider_Configure_RequestTimeoutInvalidDuration(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://example.com")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_REQUEST_TIMEOUT", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		requestTimeout:     tftypes.NewValue(tftypes.String, "nonsense-value"),
	})
	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostic error for invalid duration, got none")
	}
}

// TestProvider_Configure_RequestTimeoutDefault verifies that with neither
// env var nor HCL set, the client keeps the default 60s timeout.
func TestProvider_Configure_RequestTimeoutDefault(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://example.com")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_REQUEST_TIMEOUT", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
	})
	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c := resp.DataSourceData.(*client.Client)
	if got := c.RequestTimeout(); got != 60*time.Second {
		t.Errorf("default RequestTimeout = %s, want 60s", got)
	}
}
