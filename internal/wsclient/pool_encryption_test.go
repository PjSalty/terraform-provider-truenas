package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"
)

// versionServer answers system.info with the given release string.
func versionServer(t *testing.T, version string) *TestServer {
	t.Helper()
	return NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "system.info" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		return map[string]interface{}{"version": version, "hostname": "truenas"}, nil
	})
}

func TestValidatePoolEncryptionOptions_emptyIsFine(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "26.0.0-BETA.2").NewClient(ctx)
	if err := c.ValidatePoolEncryptionOptions(ctx, nil); err != nil {
		t.Errorf("nil options rejected: %v", err)
	}
	if err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{}); err != nil {
		t.Errorf("empty options rejected: %v", err)
	}
}

// algorithm is VALID on 25.10 and must not be rejected there. A blanket
// allowlist would have broken every current user of the attribute.
func TestValidatePoolEncryptionOptions_algorithmAllowedPre26(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "25.10.4").NewClient(ctx)
	err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{
		"algorithm": "AES-256-GCM",
	})
	if err != nil {
		t.Errorf("algorithm rejected on 25.10, where it is valid: %v", err)
	}
}

func TestValidatePoolEncryptionOptions_algorithmRejectedOn26(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "26.0.0-BETA.2").NewClient(ctx)
	err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{
		"algorithm": "AES-256-GCM",
	})
	if err == nil {
		t.Fatal("algorithm accepted on 26.0; middleware would reject it mid-create")
	}
	for _, want := range []string{"algorithm", "26.0", "26.0.0-BETA.2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("diagnostic should mention %q, got: %v", want, err)
		}
	}
}

func TestValidatePoolEncryptionOptions_algorithmRejectedOn27(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "27.0.0-BETA.1").NewClient(ctx)
	if err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{"algorithm": "AES-256-GCM"}); err == nil {
		t.Fatal("algorithm accepted on 27.0")
	}
}

// The floor moved from 100000 to 1300000 in 26.0.
func TestValidatePoolEncryptionOptions_pbkdf2iters(t *testing.T) {
	cases := []struct {
		name    string
		version string
		iters   interface{}
		wantErr bool
	}{
		{"old default fine on 25.10", "25.10.4", float64(350000), false},
		{"old default rejected on 26.0", "26.0.0-BETA.2", float64(350000), true},
		{"new floor accepted on 26.0", "26.0.0-BETA.2", float64(1300000), false},
		{"above floor accepted on 26.0", "26.0.0-BETA.2", float64(2000000), false},
		// 0 means "server default" and the 26 default already clears the
		// floor, so it must not be flagged.
		{"zero means server default", "26.0.0-BETA.2", float64(0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			c, _ := versionServer(t, tc.version).NewClient(ctx)
			err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{"pbkdf2iters": tc.iters})
			if tc.wantErr && err == nil {
				t.Error("expected a rejection")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected rejection: %v", err)
			}
		})
	}
}

func TestValidatePoolEncryptionOptions_pbkdf2itersMustBeNumeric(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "26.0.0-BETA.2").NewClient(ctx)
	if err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{"pbkdf2iters": "350000"}); err == nil {
		t.Fatal("a string pbkdf2iters was accepted")
	}
}

// A typo is wrong on every version, so it must be caught without a probe.
func TestValidatePoolEncryptionOptions_unknownKeyRejectedWithoutProbe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	probes := 0
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		probes++
		return map[string]interface{}{"version": "26.0.0-BETA.2"}, nil
	})
	c, _ := ts.NewClient(ctx)
	err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{"generate_kye": true})
	if err == nil {
		t.Fatal("a misspelled key was forwarded to middleware")
	}
	if !strings.Contains(err.Error(), "generate_kye") {
		t.Errorf("diagnostic should name the bad key, got: %v", err)
	}
	if probes != 0 {
		t.Error("probed the server for a key that is invalid on every version")
	}
}

func TestValidatePoolEncryptionOptions_validKeysPass(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "26.0.0-BETA.2").NewClient(ctx)
	err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{
		"generate_key": true,
		"pbkdf2iters":  float64(1300000),
	})
	if err != nil {
		t.Errorf("valid 26.0 options rejected: %v", err)
	}
}

// Without a version there is no safe branch, so the probe failure must
// surface rather than defaulting to "allow".
func TestValidatePoolEncryptionOptions_probeFailureIsAnError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	if err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{"algorithm": "AES-256-GCM"}); err == nil {
		t.Fatal("options were accepted without knowing the server version")
	}
}

// JSON decodes numbers as float64, but the helper also accepts the integer
// shapes a Go caller might pass, and rejects everything else.
func TestToInt(t *testing.T) {
	cases := []struct {
		in   interface{}
		want int64
		ok   bool
	}{
		{float64(1300000), 1300000, true},
		{int(42), 42, true},
		{int64(7), 7, true},
		{"1300000", 0, false},
		{true, 0, false},
		{nil, 0, false},
	}
	for _, tc := range cases {
		got, ok := toInt(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("toInt(%#v) = (%d,%v), want (%d,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

// An int-typed pbkdf2iters must be enforced the same as a float64 one.
func TestValidatePoolEncryptionOptions_intIters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _ := versionServer(t, "26.0.0-BETA.2").NewClient(ctx)
	if err := c.ValidatePoolEncryptionOptions(ctx, map[string]interface{}{"pbkdf2iters": 350000}); err == nil {
		t.Fatal("an int below the floor was accepted")
	}
}
