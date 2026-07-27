package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// username selects the auth mechanism, so its resolution carries the
// same precedence contract as url and api_key plus one guard the
// others do not need: an unknown value must never fall through as
// empty, which would discard TRUENAS_USERNAME and silently pick the
// handshake TrueNAS removes in 27.
func TestProvider_Configure_UsernameResolution(t *testing.T) {
	original := newClientFn
	t.Cleanup(func() { newClientFn = original })
	var seen string
	newClientFn = func(ctx context.Context, baseURL, apiKey, username string, insecure bool) (*wsclient.Client, error) {
		seen = username
		return &wsclient.Client{}, nil
	}

	cases := []struct {
		name      string
		env       string
		config    tftypes.Value
		want      string
		wantError bool
	}{
		{name: "env only", env: "envuser", config: nullString(), want: "envuser"},
		{name: "config beats env", env: "envuser", config: tftypes.NewValue(tftypes.String, "cfguser"), want: "cfguser"},
		{name: "unset everywhere selects legacy", env: "", config: nullString(), want: ""},
		{name: "explicit empty config selects legacy", env: "envuser", config: tftypes.NewValue(tftypes.String, ""), want: ""},
		{
			name:      "unknown errors instead of downgrading",
			env:       "envuser",
			config:    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
			wantError: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seen = "<unset>"
			t.Setenv("TRUENAS_URL", "https://u.example.com")
			t.Setenv("TRUENAS_API_KEY", "u-key")
			t.Setenv("TRUENAS_USERNAME", tc.env)

			p := New("test")()
			cfg := buildProviderConfig(t, p, providerConfigValues{
				url:                nullString(),
				apiKey:             nullString(),
				username:           tc.config,
				insecureSkipVerify: nullBool(),
			})
			resp := &provider.ConfigureResponse{}
			p.(*TrueNASProvider).Configure(context.Background(), provider.ConfigureRequest{Config: cfg}, resp)

			if tc.wantError {
				if !resp.Diagnostics.HasError() {
					t.Fatal("unknown username must error, not silently downgrade to the legacy handshake")
				}
				if seen != "<unset>" {
					t.Fatalf("client was constructed despite the unknown username (got %q)", seen)
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
			}
			if seen != tc.want {
				t.Fatalf("username passed to the client = %q, want %q", seen, tc.want)
			}
		})
	}
}
