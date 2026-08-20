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
	newClientFn = func(ctx context.Context, baseURL, apiKey, username, apiVersion string, insecure bool) (*wsclient.Client, error) {
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

// api_version is opt-in: unset means /api/current, which is what every
// previous release dialled. These pin the resolution order and the
// unknown-value refusal.
func TestProvider_Configure_apiVersionResolution(t *testing.T) {
	cases := []struct {
		name    string
		config  tftypes.Value
		env     string
		want    string
		wantErr bool
	}{
		{"unset means current", nullString(), "", "", false},
		{"config wins", tftypes.NewValue(tftypes.String, "v26.0"), "v25.10.5", "v26.0", false},
		{"env used when config is null", nullString(), "v25.10.5", "v25.10.5", false},
		// Falling through as empty would discard TRUENAS_API_VERSION and
		// quietly dial a different endpoint than the operator asked for.
		{"unknown is refused", tftypes.NewValue(tftypes.String, tftypes.UnknownValue), "v25.10.5", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRUENAS_API_VERSION", tc.env)

			var got string
			orig := newClientFn
			newClientFn = func(ctx context.Context, baseURL, apiKey, username, apiVersion string, insecure bool) (*wsclient.Client, error) {
				got = apiVersion
				return &wsclient.Client{}, nil
			}
			t.Cleanup(func() { newClientFn = orig })

			p := New("test")()
			resp := &provider.ConfigureResponse{}
			p.Configure(context.Background(), provider.ConfigureRequest{
				Config: buildProviderConfig(t, p, providerConfigValues{
					url:                tftypes.NewValue(tftypes.String, "https://tn.example.com"),
					apiKey:             tftypes.NewValue(tftypes.String, "k"),
					insecureSkipVerify: nullBool(),
					apiVersion:         tc.config,
				}),
			}, resp)

			if tc.wantErr {
				if !resp.Diagnostics.HasError() {
					t.Error("an unknown api_version was accepted")
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("Configure: %v", resp.Diagnostics)
			}
			if got != tc.want {
				t.Errorf("api_version passed to the client = %q, want %q", got, tc.want)
			}
		})
	}
}
