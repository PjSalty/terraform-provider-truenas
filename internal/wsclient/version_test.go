package wsclient

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseServerVersion(t *testing.T) {
	cases := []struct {
		raw   string
		major int
		minor int
	}{
		{"25.10.4", 25, 10},
		{"25.04.2.6", 25, 4},
		{"26.0.0-BETA.2", 26, 0},
		{"27.0.0-BETA.1", 27, 0},
		{"TrueNAS-SCALE-24.10.2", 24, 10},
		{"  25.10.6  ", 25, 10},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			v, err := parseServerVersion(tc.raw)
			if err != nil {
				t.Fatalf("parseServerVersion(%q): %v", tc.raw, err)
			}
			if v.Major != tc.major || v.Minor != tc.minor {
				t.Errorf("got %d.%d, want %d.%d", v.Major, v.Minor, tc.major, tc.minor)
			}
			if !v.Known() {
				t.Error("Known() is false for a parsed version")
			}
		})
	}
}

// An unparsable version must be an error. Defaulting would silently pick a
// wire format, which is how an apply "succeeds" while writing the wrong thing.
func TestParseServerVersion_rejectsGarbage(t *testing.T) {
	for _, raw := range []string{"", "   ", "unknown", "v", "26", "TrueNAS"} {
		if v, err := parseServerVersion(raw); err == nil {
			t.Errorf("parseServerVersion(%q) returned %v, want an error", raw, v)
		}
	}
}

func TestServerVersion_AtLeast(t *testing.T) {
	v := ServerVersion{Major: 25, Minor: 10}
	cases := []struct {
		major, minor int
		want         bool
	}{
		{25, 10, true},  // equal
		{25, 4, true},   // same major, older minor
		{24, 10, true},  // older major
		{26, 0, false},  // newer major
		{25, 11, false}, // same major, newer minor
	}
	for _, tc := range cases {
		if got := v.AtLeast(tc.major, tc.minor); got != tc.want {
			t.Errorf("25.10.AtLeast(%d,%d) = %v, want %v", tc.major, tc.minor, got, tc.want)
		}
	}
	// The minor must not be compared as a bare number across majors:
	// 26.0 is newer than 25.10 even though 0 < 10.
	if !(ServerVersion{Major: 26, Minor: 0}).AtLeast(25, 10) {
		t.Error("26.0 was not treated as newer than 25.10")
	}
}

// The zero value must not read as an old version, or a caller would take the
// legacy path against a server it never identified.
func TestServerVersion_zeroValueIsNotKnown(t *testing.T) {
	var v ServerVersion
	if v.Known() {
		t.Error("the zero value reports Known()")
	}
}

func TestClient_ServerVersion_probesAndCaches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	calls := 0
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		if method != "system.info" {
			return nil, &RPCError{Code: CodeMethodNotFound, Message: method}
		}
		calls++
		return map[string]interface{}{"version": "26.0.0-BETA.2", "hostname": "truenas"}, nil
	})
	c, _ := ts.NewClient(ctx)

	for i := 0; i < 3; i++ {
		v, err := c.ServerVersion(ctx)
		if err != nil {
			t.Fatalf("ServerVersion: %v", err)
		}
		if !v.AtLeast(26, 0) {
			t.Errorf("got %v, want >= 26.0", v)
		}
	}
	if calls != 1 {
		t.Errorf("system.info called %d times, want 1; the version must be cached", calls)
	}
}

// A failed probe must surface. Returning a zero version would let callers take
// whichever branch the zero value happens to select.
func TestClient_ServerVersion_probeFailureSurfaces(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		return nil, &RPCError{Code: CodeInternalError, Message: "boom"}
	})
	c, _ := ts.NewClient(ctx)
	v, err := c.ServerVersion(ctx)
	if err == nil {
		t.Fatal("a failed probe returned no error")
	}
	if v.Known() {
		t.Error("a failed probe returned a usable version")
	}
	if !strings.Contains(err.Error(), "determining TrueNAS version") {
		t.Errorf("got %v, want it to name what failed", err)
	}
}

// An unparsable version is cached as an error so a broken server is not
// re-probed once per resource.
func TestClient_ServerVersion_unparsableIsStickyError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	calls := 0
	ts := NewTestServer(t, func(ctx context.Context, method string, params []interface{}) (interface{}, *RPCError) {
		calls++
		return map[string]interface{}{"version": "not-a-version"}, nil
	})
	c, _ := ts.NewClient(ctx)
	for i := 0; i < 3; i++ {
		if _, err := c.ServerVersion(ctx); err == nil {
			t.Fatal("an unparsable version was accepted")
		}
	}
	if calls != 1 {
		t.Errorf("system.info called %d times, want 1; the failure must be cached", calls)
	}
}

// A version whose minor is not numeric must be rejected, not coerced.
func TestParseServerVersion_nonNumericComponents(t *testing.T) {
	for _, raw := range []string{"25.xx.4", "25.10beta.1"} {
		if v, err := parseServerVersion(raw); err == nil && v.Minor != 10 {
			t.Errorf("parseServerVersion(%q) = %v, want an error or a correct minor", raw, v)
		}
	}
	if _, err := parseServerVersion("25.xx.4"); err == nil {
		t.Error("a non-numeric minor was accepted")
	}
	// A non-numeric MAJOR is a separate branch from a non-numeric minor.
	// "9x.1" starts with a digit so it clears the prefix skip, then fails
	// on the major itself.
	if _, err := parseServerVersion("9x.1"); err == nil {
		t.Error("a non-numeric major was accepted")
	}
}
