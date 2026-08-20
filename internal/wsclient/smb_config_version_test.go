package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/types"
)

func strp(s string) *string { return &s }
func boolp(b bool) *bool    { return &b }

// --- normalizeSMBConfig ---

func TestNormalizeSMBConfig(t *testing.T) {
	cases := []struct {
		name         string
		in           types.SMBConfig
		wantProtocol string
		wantSMB1     bool
		wantDialect  types.SMBDialect
	}{
		{
			name:         "26.0 minimum_protocol SMB2",
			in:           types.SMBConfig{MinimumProtocol: strp("SMB2")},
			wantProtocol: "SMB2", wantSMB1: false, wantDialect: types.SMBDialectMinimumProtocol,
		},
		{
			name:         "26.0 minimum_protocol SMB1",
			in:           types.SMBConfig{MinimumProtocol: strp("SMB1")},
			wantProtocol: "SMB1", wantSMB1: true, wantDialect: types.SMBDialectMinimumProtocol,
		},
		{
			// SMB3 has no legacy representation at all, so it must survive
			// the round trip untouched rather than collapsing to SMB2.
			name:         "26.0 minimum_protocol SMB3",
			in:           types.SMBConfig{MinimumProtocol: strp("SMB3")},
			wantProtocol: "SMB3", wantSMB1: false, wantDialect: types.SMBDialectMinimumProtocol,
		},
		{
			// Mapping fixed by middleware's own alembic migration:
			// CASE WHEN enable_smb1 = 1 THEN 'SMB1' ELSE 'SMB2' END.
			name:         "25.10 enable_smb1 true",
			in:           types.SMBConfig{EnableSMB1: boolp(true)},
			wantProtocol: "SMB1", wantSMB1: true, wantDialect: types.SMBDialectEnableSMB1,
		},
		{
			name:         "25.10 enable_smb1 false",
			in:           types.SMBConfig{EnableSMB1: boolp(false)},
			wantProtocol: "SMB2", wantSMB1: false, wantDialect: types.SMBDialectEnableSMB1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.in
			if err := normalizeSMBConfig(&cfg); err != nil {
				t.Fatalf("normalizeSMBConfig: %v", err)
			}
			if cfg.Protocol != tc.wantProtocol {
				t.Errorf("Protocol = %q, want %q", cfg.Protocol, tc.wantProtocol)
			}
			if cfg.SMB1Enabled != tc.wantSMB1 {
				t.Errorf("SMB1Enabled = %v, want %v", cfg.SMB1Enabled, tc.wantSMB1)
			}
			if cfg.Dialect != tc.wantDialect {
				t.Errorf("Dialect = %v, want %v", cfg.Dialect, tc.wantDialect)
			}
		})
	}
}

// A response carrying neither key must fail loudly. Defaulting to SMB2 would
// report a setting the server never stated, and on a box that really did have
// SMB1 enabled the next apply would silently turn it off.
func TestNormalizeSMBConfig_neitherKeyIsAnError(t *testing.T) {
	cfg := types.SMBConfig{NetbiosName: "truenas"}
	err := normalizeSMBConfig(&cfg)
	if err == nil {
		t.Fatal("a response with no protocol key was accepted; it would be reported as SMB2")
	}
	for _, want := range []string{"enable_smb1", "minimum_protocol"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %q, got: %v", want, err)
		}
	}
}

// --- buildSMBUpdateBody ---

func TestBuildSMBUpdateBody_modernDialect(t *testing.T) {
	body, err := buildSMBUpdateBody(
		&types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB3")},
		types.SMBDialectMinimumProtocol)
	if err != nil {
		t.Fatalf("buildSMBUpdateBody: %v", err)
	}
	if body.MinimumProtocol == nil || *body.MinimumProtocol != "SMB3" {
		t.Errorf("minimum_protocol = %v, want SMB3", body.MinimumProtocol)
	}
	if body.EnableSMB1 != nil {
		t.Error("enable_smb1 was set on a 26.0 body; the model forbids the key and would hard-fail")
	}
}

func TestBuildSMBUpdateBody_legacyDialect(t *testing.T) {
	cases := []struct {
		proto string
		want  bool
	}{{"SMB1", true}, {"SMB2", false}}
	for _, tc := range cases {
		t.Run(tc.proto, func(t *testing.T) {
			body, err := buildSMBUpdateBody(
				&types.SMBConfigUpdateRequest{MinimumProtocol: strp(tc.proto)},
				types.SMBDialectEnableSMB1)
			if err != nil {
				t.Fatalf("buildSMBUpdateBody: %v", err)
			}
			if body.EnableSMB1 == nil || *body.EnableSMB1 != tc.want {
				t.Errorf("enable_smb1 = %v, want %v", body.EnableSMB1, tc.want)
			}
			if body.MinimumProtocol != nil {
				t.Error("minimum_protocol was set on a 25.10 body; that key is a hard ValidationError there")
			}
		})
	}
}

// SMB3 cannot be expressed as a boolean. Degrading it to false would apply
// SMB2 and report success, which is the silent-failure shape this whole
// change exists to remove.
func TestBuildSMBUpdateBody_smb3OnLegacyIsAnError(t *testing.T) {
	_, err := buildSMBUpdateBody(
		&types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB3")},
		types.SMBDialectEnableSMB1)
	if err == nil {
		t.Fatal("SMB3 was silently accepted on a server that cannot express it")
	}
	if !strings.Contains(err.Error(), "26.0") {
		t.Errorf("error should say which version is required, got: %v", err)
	}
}

// The reset path is the reason enable_smb1 shipped on requests from users who
// never set it: the old body emitted the key unconditionally. No protocol
// requested must mean no protocol key at all.
func TestBuildSMBUpdateBody_noProtocolEmitsNeitherKey(t *testing.T) {
	for _, d := range []types.SMBDialect{types.SMBDialectEnableSMB1, types.SMBDialectMinimumProtocol} {
		body, err := buildSMBUpdateBody(
			&types.SMBConfigUpdateRequest{NetbiosName: strp("truenas")}, d)
		if err != nil {
			t.Fatalf("buildSMBUpdateBody: %v", err)
		}
		if body.EnableSMB1 != nil || body.MinimumProtocol != nil {
			t.Errorf("dialect %v: a protocol key was emitted for a request that set none", d)
		}
	}
}

func TestBuildSMBUpdateBody_unknownDialectIsAnError(t *testing.T) {
	_, err := buildSMBUpdateBody(
		&types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB2")},
		types.SMBDialectUnknown)
	if err == nil {
		t.Fatal("an undetermined dialect was allowed to guess a wire key")
	}
}

// --- end-to-end over the fake server ---

// smbVersionServer models a TrueNAS of one version and records the keys that
// actually reached the wire.
func smbVersionServer(t *testing.T, modern bool, got *map[string]interface{}) *TestServer {
	t.Helper()
	entry := func() map[string]interface{} {
		if modern {
			return map[string]interface{}{"id": 1, "netbiosname": "truenas", "minimum_protocol": "SMB2"}
		}
		return map[string]interface{}{"id": 1, "netbiosname": "truenas", "enable_smb1": false}
	}
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "smb.config":
			return entry(), nil
		case "smb.update":
			if len(params) > 0 {
				if m, ok := params[0].(map[string]interface{}); ok {
					*got = m
				}
			}
			return entry(), nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
}

func TestUpdateSMBConfig_sendsTheKeyTheServerSpeaks(t *testing.T) {
	cases := []struct {
		name        string
		modern      bool
		wantKey     string
		forbidKey   string
		wantValue   interface{}
		reqProtocol string
	}{
		{"26.0 server gets minimum_protocol", true, "minimum_protocol", "enable_smb1", "SMB1", "SMB1"},
		{"25.10 server gets enable_smb1", false, "enable_smb1", "minimum_protocol", true, "SMB1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var got map[string]interface{}
			ts := smbVersionServer(t, tc.modern, &got)
			c, _ := ts.NewClient(ctx)

			if _, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{
				MinimumProtocol: strp(tc.reqProtocol),
			}); err != nil {
				t.Fatalf("UpdateSMBConfig: %v", err)
			}
			if _, present := got[tc.forbidKey]; present {
				t.Errorf("body carried %q, which this server rejects outright: %v", tc.forbidKey, got)
			}
			v, present := got[tc.wantKey]
			if !present {
				t.Fatalf("body did not carry %q: %v", tc.wantKey, got)
			}
			if v != tc.wantValue {
				t.Errorf("%s = %v, want %v", tc.wantKey, v, tc.wantValue)
			}
		})
	}
}

// The dialect probe must not cost a round trip when a Read already happened,
// which is the ordinary Terraform Read-then-Update sequence.
func TestUpdateSMBConfig_reusesDialectFromPriorRead(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	configCalls := 0
	var got map[string]interface{}
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "smb.config":
			configCalls++
			return map[string]interface{}{"id": 1, "minimum_protocol": "SMB2"}, nil
		case "smb.update":
			if m, ok := params[0].(map[string]interface{}); ok {
				got = m
			}
			return map[string]interface{}{"id": 1, "minimum_protocol": "SMB3"}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)

	if _, err := c.GetSMBConfig(ctx); err != nil {
		t.Fatalf("GetSMBConfig: %v", err)
	}
	if _, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB3")}); err != nil {
		t.Fatalf("UpdateSMBConfig: %v", err)
	}
	if configCalls != 1 {
		t.Errorf("smb.config called %d times, want 1 (the Read); Update should reuse the cached dialect", configCalls)
	}
	if _, present := got["enable_smb1"]; present {
		t.Error("legacy key leaked onto a 26.0 body")
	}
}

// GetSMBConfig must surface an unrecognized shape rather than returning a
// config whose protocol fields are silently zero.
func TestGetSMBConfig_noProtocolKeyIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return map[string]interface{}{"id": 1, "netbiosname": "truenas"}, nil
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.GetSMBConfig(ctx); err == nil {
		t.Fatal("a config with no protocol key was accepted")
	}
}

// A failed dialect probe must abort the update, not fall through to a guess.
func TestUpdateSMBConfig_dialectProbeFailureAborts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updates := 0
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "smb.update" {
			updates++
		}
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB2")})
	if err == nil || !strings.Contains(err.Error(), "updating SMB config") {
		t.Errorf("got %v", err)
	}
	if updates != 0 {
		t.Error("smb.update was issued despite an unknown dialect")
	}
}

// A build failure must abort BEFORE smb.update goes out.
func TestUpdateSMBConfig_buildFailureSkipsTheCall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	updates := 0
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "smb.config":
			return map[string]interface{}{"id": 1, "enable_smb1": false}, nil
		case "smb.update":
			updates++
			return map[string]interface{}{"id": 1, "enable_smb1": false}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	// SMB3 against a legacy server cannot be expressed.
	_, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB3")})
	if err == nil {
		t.Fatal("SMB3 was sent to a server that cannot express it")
	}
	if updates != 0 {
		t.Error("smb.update was issued after the body failed to build")
	}
}

// The write path decodes the returned entry too, so an unrecognized response
// there must error rather than land a zero protocol in state.
func TestUpdateSMBConfig_unrecognizedResponseIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		switch method {
		case "smb.config":
			return map[string]interface{}{"id": 1, "minimum_protocol": "SMB2"}, nil
		case "smb.update":
			return map[string]interface{}{"id": 1, "netbiosname": "x"}, nil
		}
		return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
	})
	c, _ := ts.NewClient(ctx)
	if _, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB2")}); err == nil {
		t.Fatal("an update response with no protocol key was accepted")
	}
}

// smb.update failing AFTER a successful dialect probe is a distinct path
// from the probe itself failing.
func TestUpdateSMBConfig_updateCallFailureAfterProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "smb.config" {
			return map[string]interface{}{"id": 1, "minimum_protocol": "SMB2"}, nil
		}
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB2")})
	if err == nil || !strings.Contains(err.Error(), "updating SMB config") {
		t.Errorf("got %v", err)
	}
}

func TestUpdateSMBConfig_responseDecodeError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method == "smb.config" {
			return map[string]interface{}{"id": 1, "minimum_protocol": "SMB2"}, nil
		}
		return "not-an-object", nil
	})
	c, _ := ts.NewClient(ctx)
	_, err := c.UpdateSMBConfig(ctx, &types.SMBConfigUpdateRequest{MinimumProtocol: strp("SMB2")})
	if err == nil || !strings.Contains(err.Error(), "parsing") {
		t.Errorf("got %v", err)
	}
}
