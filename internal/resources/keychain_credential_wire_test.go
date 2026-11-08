package resources

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func kcAttrs(vals map[string]tftypes.Value) tftypes.Value {
	return tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, vals)
}

func kcSSHCredsAttrs(port, privateKey string) tftypes.Value {
	return kcAttrs(map[string]tftypes.Value{
		"host":            str("192.0.2.10"),
		"port":            str(port),
		"username":        str("root"),
		"private_key":     str(privateKey),
		"remote_host_key": str("ssh-ed25519 AAAA"),
		"connect_timeout": str("10"),
	})
}

// kcCaptureClient records the attributes sent to keychaincredential.create
// and keychaincredential.update.
func kcCaptureClient(t *testing.T) (*wsclient.Client, func(method string) map[string]interface{}) {
	t.Helper()
	var mu sync.Mutex
	sent := map[string]map[string]interface{}{}
	body := map[string]interface{}{
		"id": 2, "name": "creds", "type": "SSH_CREDENTIALS",
		"attributes": map[string]interface{}{
			"host": "192.0.2.10", "port": 22, "username": "root", "private_key": 1,
			"remote_host_key": "ssh-ed25519 AAAA", "connect_timeout": 10,
		},
	}
	c := newWSTestClient(context.Background(), t, func(_ context.Context, method string, params []interface{}) (interface{}, *wsclient.RPCError) {
		var req interface{}
		switch method {
		case "keychaincredential.create":
			req = params[0]
		case "keychaincredential.update":
			req = params[1]
		}
		if m, ok := req.(map[string]interface{}); ok {
			if a, ok := m["attributes"].(map[string]interface{}); ok {
				mu.Lock()
				sent[method] = a
				mu.Unlock()
			}
		}
		return body, nil
	})
	return c, func(method string) map[string]interface{} {
		mu.Lock()
		defer mu.Unlock()
		return sent[method]
	}
}

// Middleware validates strictly: the string "22" is rejected for port with
// "Input should be a valid integer". The integer attributes of an
// SSH_CREDENTIALS credential must go on the wire as numbers.
func TestKeychainCredential_SSHCredentialsIntegersOnWire(t *testing.T) {
	ctx := context.Background()
	c, sent := kcCaptureClient(t)
	r := &KeychainCredentialResource{client: c}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"name":       str("creds"),
		"type":       str("SSH_CREDENTIALS"),
		"attributes": kcSSHCredsAttrs("22", "1"),
	}

	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, vals)}, cResp)
	if cResp.Diagnostics.HasError() {
		t.Fatalf("create: %v", cResp.Diagnostics)
	}

	vals["id"] = str("2")
	st := stateFromValues(t, ctx, sch, vals)
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, uResp)
	if uResp.Diagnostics.HasError() {
		t.Fatalf("update: %v", uResp.Diagnostics)
	}

	for _, method := range []string{"keychaincredential.create", "keychaincredential.update"} {
		a := sent(method)
		if a == nil {
			t.Fatalf("%s: no attributes captured", method)
		}
		for _, k := range []string{"port", "private_key", "connect_timeout"} {
			if _, ok := a[k].(float64); !ok {
				t.Errorf("%s: attributes.%s = %#v (%T), want a JSON number", method, k, a[k], a[k])
			}
		}
		for _, k := range []string{"host", "username", "remote_host_key"} {
			if _, ok := a[k].(string); !ok {
				t.Errorf("%s: attributes.%s = %#v, want a string", method, k, a[k])
			}
		}
	}
}

// An SSH_KEY_PAIR private_key is the PEM text, not an ID: it stays a string.
func TestKeychainWireAttributes_KeyPairUnchanged(t *testing.T) {
	got, err := keychainWireAttributes("SSH_KEY_PAIR", map[string]string{"private_key": "PEM", "public_key": "PUB"})
	if err != nil {
		t.Fatal(err)
	}
	if got["private_key"] != "PEM" || got["public_key"] != "PUB" {
		t.Errorf("got %#v", got)
	}
}

// A non-number fails without calling the API, naming the key and never the
// value: the attributes are sensitive.
func TestKeychainCredential_SSHCredentialsNotANumber(t *testing.T) {
	ctx := context.Background()
	c, sent := kcCaptureClient(t)
	r := &KeychainCredentialResource{client: c}
	sch := schemaOf(t, ctx, r)
	vals := map[string]tftypes.Value{
		"name":       str("creds"),
		"type":       str("SSH_CREDENTIALS"),
		"attributes": kcSSHCredsAttrs("ssh-port-secret", "1"),
	}
	check := func(op string, d interface {
		HasError() bool
	}, detail string) {
		t.Helper()
		if !d.HasError() {
			t.Fatalf("%s: a non-numeric port was accepted", op)
		}
		if !strings.Contains(detail, "attributes.port") || strings.Contains(detail, "ssh-port-secret") {
			t.Errorf("%s: detail %q should name attributes.port and not echo the value", op, detail)
		}
	}

	cResp := &resource.CreateResponse{State: primedStateV2(t, ctx, sch)}
	r.Create(ctx, resource.CreateRequest{Plan: planFromValues(t, ctx, sch, vals)}, cResp)
	check("create", cResp.Diagnostics, cResp.Diagnostics.Errors()[0].Detail())

	vals["id"] = str("2")
	st := stateFromValues(t, ctx, sch, vals)
	uResp := &resource.UpdateResponse{State: st}
	r.Update(ctx, resource.UpdateRequest{State: st, Plan: planFromValues(t, ctx, sch, vals)}, uResp)
	check("update", uResp.Diagnostics, uResp.Diagnostics.Errors()[0].Detail())

	if sent("keychaincredential.create") != nil || sent("keychaincredential.update") != nil {
		t.Error("the API was called with an invalid port")
	}
}

func TestKeychainCredential_ValidateConfig(t *testing.T) {
	ctx := context.Background()
	r := &KeychainCredentialResource{}
	sch := schemaOf(t, ctx, r)
	unknownStr := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	cases := []struct {
		name    string
		typ     tftypes.Value
		attrs   tftypes.Value
		wantKey string
	}{
		{"valid", str("SSH_CREDENTIALS"), kcSSHCredsAttrs("22", "1"), ""},
		{"port not a number", str("SSH_CREDENTIALS"), kcSSHCredsAttrs("twenty-two", "1"), "port"},
		{"private_key not a number", str("SSH_CREDENTIALS"), kcSSHCredsAttrs("22", "key"), "private_key"},
		// private_key = truenas_keychain_credential.kp.id is unknown at plan time.
		{"private_key unknown", str("SSH_CREDENTIALS"), kcAttrs(map[string]tftypes.Value{
			"host": str("192.0.2.10"), "private_key": unknownStr,
		}), ""},
		{"key pair PEM is not checked", str("SSH_KEY_PAIR"), kcAttrs(map[string]tftypes.Value{
			"private_key": str("-----BEGIN OPENSSH PRIVATE KEY-----"),
		}), ""},
		{"type unknown", unknownStr, kcSSHCredsAttrs("twenty-two", "1"), ""},
		{"attributes unknown", str("SSH_CREDENTIALS"),
			tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, tftypes.UnknownValue), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := stateFromValues(t, ctx, sch, map[string]tftypes.Value{
				"name": str("creds"), "type": tc.typ, "attributes": tc.attrs,
			}).Raw
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: sch.Schema, Raw: raw}}, resp)
			if tc.wantKey == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected: %v", resp.Diagnostics)
				}
				return
			}
			errs := resp.Diagnostics.Errors()
			if len(errs) != 1 {
				t.Fatalf("want one error, got: %v", resp.Diagnostics)
			}
			want := path.Root("attributes").AtMapKey(tc.wantKey)
			if wp, ok := errs[0].(interface{ Path() path.Path }); !ok || !wp.Path().Equal(want) {
				t.Errorf("want error on %s, got: %v", want, errs[0])
			}
		})
	}
}

// A config that does not decode is reported, not validated further.
func TestKeychainCredential_ValidateConfigDecodeError(t *testing.T) {
	ctx := context.Background()
	r := &KeychainCredentialResource{}
	sch := schemaOf(t, ctx, r)
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{
		Schema: sch.Schema, Raw: tftypes.NewValue(tftypes.String, "not an object"),
	}}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("an undecodable config was accepted")
	}
}
