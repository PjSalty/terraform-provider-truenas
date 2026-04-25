package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestProvider_Configure_DestroyProtectionEnvVar verifies that
// TRUENAS_DESTROY_PROTECTION=1 causes the provider to construct a
// client with DestroyProtection=true, so every DELETE request will
// fail with ErrDestroyProtected before reaching the network — while
// POST and PUT still flow through normally.
func TestProvider_Configure_DestroyProtectionEnvVar(t *testing.T) {
	cases := []struct {
		name     string
		envValue string
		want     bool
	}{
		{name: "unset", envValue: "", want: false},
		{name: "true lowercase", envValue: "true", want: true},
		{name: "one", envValue: "1", want: true},
		{name: "false-ish garbage", envValue: "yes", want: false},
		{name: "false literal", envValue: "false", want: false},
		{name: "zero", envValue: "0", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRUENAS_URL", "https://dp.example.com")
			t.Setenv("TRUENAS_API_KEY", "dp-key")
			t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
			t.Setenv("TRUENAS_READONLY", "")
			t.Setenv("TRUENAS_DESTROY_PROTECTION", tc.envValue)

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
			c, ok := resp.DataSourceData.(*client.Client)
			if !ok {
				t.Fatalf("DataSourceData is not *client.Client, got %T", resp.DataSourceData)
			}
			if c.DestroyProtection != tc.want {
				t.Errorf("DestroyProtection = %v, want %v (env %q)", c.DestroyProtection, tc.want, tc.envValue)
			}
		})
	}
}

// TestProvider_Configure_DestroyProtectionHCLAttribute verifies that
// the HCL attribute propagates to Client.DestroyProtection independent
// of the env var.
func TestProvider_Configure_DestroyProtectionHCLAttribute(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://dp.example.com")
	t.Setenv("TRUENAS_API_KEY", "dp-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_DESTROY_PROTECTION", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		destroyProtection:  tftypes.NewValue(tftypes.Bool, true),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c, ok := resp.DataSourceData.(*client.Client)
	if !ok {
		t.Fatalf("DataSourceData is not *client.Client, got %T", resp.DataSourceData)
	}
	if !c.DestroyProtection {
		t.Error("DestroyProtection = false, want true (set via HCL)")
	}
}

// TestProvider_Configure_DestroyProtectionHCLOverridesEnv verifies
// that the HCL attribute takes precedence over the env var — same
// precedence rule as ReadOnly. HCL is closer to the operator's intent.
func TestProvider_Configure_DestroyProtectionHCLOverridesEnv(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://dp.example.com")
	t.Setenv("TRUENAS_API_KEY", "dp-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_DESTROY_PROTECTION", "1") // env says ON

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		destroyProtection:  tftypes.NewValue(tftypes.Bool, false), // HCL says OFF
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c, ok := resp.DataSourceData.(*client.Client)
	if !ok {
		t.Fatalf("DataSourceData is not *client.Client, got %T", resp.DataSourceData)
	}
	if c.DestroyProtection {
		t.Error("DestroyProtection = true, want false (HCL false should override env var 1)")
	}
}

// TestProvider_Configure_SafeApplyProfile verifies the specific layer
// combo that the phased-rollout guide recommends for the first prod
// apply: read_only=false + destroy_protection=true. Creates and updates
// work; deletes are blocked. This is the production "safe apply" knob.
func TestProvider_Configure_SafeApplyProfile(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://dp.example.com")
	t.Setenv("TRUENAS_API_KEY", "dp-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "")
	t.Setenv("TRUENAS_DESTROY_PROTECTION", "")

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		readOnly:           tftypes.NewValue(tftypes.Bool, false),
		destroyProtection:  tftypes.NewValue(tftypes.Bool, true),
	})

	resp := &provider.ConfigureResponse{}
	p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	c, ok := resp.DataSourceData.(*client.Client)
	if !ok {
		t.Fatalf("DataSourceData is not *client.Client, got %T", resp.DataSourceData)
	}
	if c.ReadOnly {
		t.Error("ReadOnly = true, want false for safe-apply profile")
	}
	if !c.DestroyProtection {
		t.Error("DestroyProtection = false, want true for safe-apply profile")
	}
}
