package sweep_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/sweep"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestCtx(t *testing.T) {
	t.Parallel()
	ctx, cancel := sweep.Ctx()
	defer cancel()
	if ctx == nil {
		t.Fatal("Ctx returned nil context")
	}
	if dl, ok := ctx.Deadline(); !ok || dl.IsZero() {
		t.Error("Ctx must carry a deadline so a hung TrueNAS can't hang the sweeper run")
	}
}

func TestHasAcctestPrefix(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"tf-acc-foo":    true,
		"tf-acc-":       true,
		"tf-acc":        false,
		"":              false,
		"prod-dataset":  false,
		"acc-tf-prefix": false, // wrong order
	}
	for in, want := range cases {
		if got := sweep.HasAcctestPrefix(in); got != want {
			t.Errorf("HasAcctestPrefix(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDatasetIsAcctest(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"pool/tf-acc-foo":   true,
		"tank/sub/tf-acc-x": true,
		"pool/prod":         false,
		"tf-acc-bare":       true, // no slash at all is the leaf
		"":                  false,
	}
	for in, want := range cases {
		if got := sweep.DatasetIsAcctest(in); got != want {
			t.Errorf("DatasetIsAcctest(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLog_OK(t *testing.T) {
	// Smoke that Log doesn't panic; capture is tricky in a parallel
	// test so we just exercise both branches.
	sweep.Log("cronjob", "destroy", "tf-acc-foo", nil)
	sweep.Log("cronjob", "destroy", "tf-acc-bar", context.Canceled)
}

// QueryList calls "<namespace>.query" over the JSON-RPC WebSocket. It
// replaced an HTTP GET against /api/v2.0, which TrueNAS 26.0 removed:
// those GETs answer 404, the sweeper returned an error, and the
// plugin-testing framework aborts the whole run on the first sweeper
// error, so one dead call stopped every later sweeper.
func TestQueryList_OK(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var gotMethod string
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, _ []interface{}) (interface{}, *wsclient.RPCError) {
		gotMethod = method
		return []interface{}{
			map[string]interface{}{"id": 1, "name": "tf-acc-one"},
			map[string]interface{}{"id": 2, "name": "keep-me"},
		}, nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	var out []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := sweep.QueryList(ctx, c, "iscsi.portal.query", &out); err != nil {
		t.Fatalf("QueryList: %v", err)
	}
	// The namespace must become "<ns>.query", or the sweeper silently
	// lists nothing and reports a clean run having reclaimed nothing.
	if gotMethod != "iscsi.portal.query" {
		t.Errorf("called %q, want %q", gotMethod, "iscsi.portal.query")
	}
	if len(out) != 2 || out[0].Name != "tf-acc-one" {
		t.Errorf("decoded %+v", out)
	}
}

func TestQueryList_NilClient(t *testing.T) {
	t.Parallel()
	var out []struct{}
	err := sweep.QueryList(context.Background(), nil, "iscsi.portal.query", &out)
	if err == nil {
		t.Fatal("a nil client was accepted")
	}
	// Assert the specific message: without the guard the call still fails,
	// but with something that does not say what actually went wrong.
	if !strings.Contains(err.Error(), "nil client") {
		t.Errorf("error should name the nil client, got: %v", err)
	}
}

// A failing query must surface. Swallowing it would report a clean sweep
// that reclaimed nothing, and the litter only shows up as an EEXIST
// collision on some later run.
func TestQueryList_CallError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts := wsclient.NewTestServer(t, func(_ context.Context, method string, _ []interface{}) (interface{}, *wsclient.RPCError) {
		return nil, &wsclient.RPCError{Code: wsclient.CodeMethodNotFound, Message: method}
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var out []struct{}
	err = sweep.QueryList(ctx, c, "nope.gone.query", &out)
	if err == nil {
		t.Fatal("a failed query was treated as success")
	}
	if !strings.Contains(err.Error(), "nope.gone.query") {
		t.Errorf("error should name the method, got: %v", err)
	}
	// It must be reported as the QUERY failing. Falling through to the
	// decode step turns a dead endpoint into "malformed response", which
	// sends the next person looking at the wrong layer entirely.
	if !strings.HasPrefix(err.Error(), "query ") {
		t.Errorf("a failed call must be reported as a query failure, got: %v", err)
	}
	if strings.Contains(err.Error(), "decode") {
		t.Errorf("a failed call was misreported as a decode failure: %v", err)
	}
}

func TestQueryList_BadJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ts := wsclient.NewTestServer(t, func(_ context.Context, _ string, _ []interface{}) (interface{}, *wsclient.RPCError) {
		return "not-a-list", nil
	})
	c, err := ts.NewClient(ctx)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	var out []struct{}
	err = sweep.QueryList(ctx, c, "iscsi.portal.query", &out)
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("err = %v", err)
	}
}

// LiveAPIKeyID tells the api_key sweeper which key it is authenticated
// with, so it does not revoke its own credential mid-run and leave every
// later sweeper unauthenticated.
func TestLiveAPIKeyID(t *testing.T) {
	cases := []struct {
		name   string
		value  string
		set    bool
		wantID int
		wantOK bool
	}{
		{"well-formed key", "17-abcdefghijklmnop", true, 17, true},
		{"unset", "", false, 0, false},
		{"empty", "", true, 0, false},
		{"no separator", "abcdefghijklmnop", true, 0, false},
		{"non-numeric id", "notanid-secret", true, 0, false},
		{"leading separator", "-secret", true, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv("TRUENAS_API_KEY", tc.value)
			} else {
				t.Setenv("TRUENAS_API_KEY", "")
				os.Unsetenv("TRUENAS_API_KEY")
			}
			id, ok := sweep.LiveAPIKeyID()
			if ok != tc.wantOK || id != tc.wantID {
				t.Errorf("LiveAPIKeyID() = (%d, %v), want (%d, %v)", id, ok, tc.wantID, tc.wantOK)
			}
		})
	}
}

// The secret half must never be returned, whatever it contains.
func TestLiveAPIKeyID_ReturnsOnlyTheID(t *testing.T) {
	t.Setenv("TRUENAS_API_KEY", "42-thisisthesecretpart-with-dashes")
	id, ok := sweep.LiveAPIKeyID()
	if !ok || id != 42 {
		t.Fatalf("LiveAPIKeyID() = (%d, %v), want (42, true)", id, ok)
	}
}
