package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// TestProvider_Configure_ReadOnlyEnvVar verifies that TRUENAS_READONLY=1
// causes the provider to construct a client with ReadOnly=true, so every
// mutating request will fail with ErrReadOnly before reaching the network.
func TestProvider_Configure_ReadOnlyEnvVar(t *testing.T) {
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
			t.Setenv("TRUENAS_URL", "https://ro.example.com")
			t.Setenv("TRUENAS_API_KEY", "ro-key")
			t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
			t.Setenv("TRUENAS_READONLY", tc.envValue)

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
			if c.ReadOnly != tc.want {
				t.Errorf("ReadOnly = %v, want %v (env %q)", c.ReadOnly, tc.want, tc.envValue)
			}
		})
	}
}

// TestProvider_Configure_ReadOnlyHCLAttribute verifies that setting the
// provider schema attribute `read_only = true` in HCL propagates to
// Client.ReadOnly, independent of the TRUENAS_READONLY env var.
func TestProvider_Configure_ReadOnlyHCLAttribute(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://hcl.example.com")
	t.Setenv("TRUENAS_API_KEY", "hcl-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "") // env var explicitly OFF

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		readOnly:           tftypes.NewValue(tftypes.Bool, true),
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
	if !c.ReadOnly {
		t.Error("ReadOnly = false, want true (set via HCL)")
	}
}

// TestProvider_Configure_ReadOnlyHCLOverridesEnv verifies that the HCL
// attribute takes precedence over the env var when both are set. HCL wins
// because it is closer to the operator's intent — the env var is a
// shell-level default, the HCL value is the explicit module choice.
func TestProvider_Configure_ReadOnlyHCLOverridesEnv(t *testing.T) {
	t.Setenv("TRUENAS_URL", "https://hcl.example.com")
	t.Setenv("TRUENAS_API_KEY", "hcl-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "")
	t.Setenv("TRUENAS_READONLY", "1") // env says ON

	p := New("test")()
	cfg := buildProviderConfig(t, p, providerConfigValues{
		url:                nullString(),
		apiKey:             nullString(),
		insecureSkipVerify: nullBool(),
		readOnly:           tftypes.NewValue(tftypes.Bool, false), // HCL says OFF
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
		t.Error("ReadOnly = true, want false (HCL false should override env var 1)")
	}
}
