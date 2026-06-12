package sweep_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/sweep"
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

func TestGetList_OK(t *testing.T) {
	type item struct {
		ID int `json:"id"`
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/iscsi/portal") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong auth header: %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode([]item{{ID: 1}, {ID: 2}})
	}))
	defer srv.Close()

	t.Setenv("TRUENAS_URL", srv.URL)
	t.Setenv("TRUENAS_API_KEY", "test-key")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	var out []item
	if err := sweep.GetList(context.Background(), nil, "/iscsi/portal", &out); err != nil {
		t.Fatalf("GetList: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("got %d items, want 2", len(out))
	}
}

func TestGetList_MissingCreds(t *testing.T) {
	t.Setenv("TRUENAS_URL", "")
	t.Setenv("TRUENAS_API_KEY", "")
	err := sweep.GetList(context.Background(), nil, "/anything", &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "TRUENAS_URL") {
		t.Errorf("want error about TRUENAS_URL, got: %v", err)
	}
}

func TestGetList_HTTPError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()
	t.Setenv("TRUENAS_URL", srv.URL)
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	err := sweep.GetList(context.Background(), nil, "/missing", &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("want HTTP 404 error, got: %v", err)
	}
}

func TestLog_OK(t *testing.T) {
	// Smoke that Log doesn't panic; capture is tricky in a parallel
	// test so we just exercise both branches.
	sweep.Log("cronjob", "destroy", "tf-acc-foo", nil)
	sweep.Log("cronjob", "destroy", "tf-acc-bar", context.Canceled)
}

func TestGetList_BadURL(t *testing.T) {
	// A control character forces http.NewRequest to error before
	// any HTTP call. Exercises the build-request error path.
	t.Setenv("TRUENAS_URL", "http://example.com")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	// Embedded newline character in the path triggers
	// "net/http: invalid path" from http.NewRequestWithContext.
	err := sweep.GetList(context.Background(), nil, "/bad\npath", &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "build request") {
		t.Errorf("want build-request error, got: %v", err)
	}
}

func TestGetList_BadJSON(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()
	t.Setenv("TRUENAS_URL", srv.URL)
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	err := sweep.GetList(context.Background(), nil, "/anything", &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Errorf("want decode error, got: %v", err)
	}
}

func TestGetList_DialFail(t *testing.T) {
	// Point at an unreachable address, the http.Client's Do call
	// will return a dial error which exercises the GET-failure branch.
	t.Setenv("TRUENAS_URL", "https://127.0.0.1:1")
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	err := sweep.GetList(context.Background(), nil, "/anything", &struct{}{})
	if err == nil || !strings.Contains(err.Error(), "GET /anything") {
		t.Errorf("want GET path error, got: %v", err)
	}
}

func TestGetList_BodyReadFails(t *testing.T) {
	// Lie about Content-Length then close the connection mid-body so
	// io.ReadAll fails after a successful header exchange. Covers the
	// read-body error branch.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("partial"))
		// Hijack + close to cut the stream short.
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, _ := hj.Hijack()
			_ = conn.Close()
		}
	}))
	defer srv.Close()
	t.Setenv("TRUENAS_URL", srv.URL)
	t.Setenv("TRUENAS_API_KEY", "k")
	t.Setenv("TRUENAS_INSECURE_SKIP_VERIFY", "true")

	err := sweep.GetList(context.Background(), nil, "/anything", &struct{}{})
	if err == nil {
		t.Error("expected error from truncated body")
	}
}
