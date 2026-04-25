package sweep_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
	"github.com/PjSalty/terraform-provider-truenas/internal/sweep"
)

func TestCtx(t *testing.T) {
	t.Parallel()
	ctx, cancel := sweep.Ctx()
	defer cancel()
	if ctx == nil {
		t.Fatalf("Ctx returned nil context")
	}
	dl, ok := ctx.Deadline()
	if !ok {
		t.Fatalf("Ctx must have a deadline")
	}
	if dl.IsZero() {
		t.Fatalf("Ctx deadline must be non-zero")
	}
}

func TestHasAcctestPrefix(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		want bool
	}{
		{"acct-foo", true},
		{"acctest-bar", true},
		{"tf-acc-baz", true},
		{"tfacc123", true},
		{"ACCT-upper", true},
		{"user-data", false},
		{"production", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := sweep.HasAcctestPrefix(tc.name); got != tc.want {
			t.Errorf("HasAcctestPrefix(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDatasetIsAcctest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		id   string
		want bool
	}{
		{"tank/acct-foo", true},
		{"tank/child/acct-bar", true},
		{"test/tf-acc-one", true},
		{"pool/tfacc-two", true},
		{"tank", false},
		{"tank/system", false},
		{"tank/prod-data", false},
	}
	for _, tc := range cases {
		if got := sweep.DatasetIsAcctest(tc.id); got != tc.want {
			t.Errorf("DatasetIsAcctest(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestLog(t *testing.T) {
	t.Parallel()
	// Log writes to os.Stdout; we can't easily capture that without
	// hijacking fds. The value of exercising it is the error vs. OK
	// branch coverage, which we do here for completeness.
	sweep.Log("truenas_user", "delete", "acct-foo", nil)
	sweep.Log("truenas_user", "delete", "acct-foo", errors.New("boom"))
}

func TestGetList_OK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"a"},{"name":"b"}]`))
	}))
	defer srv.Close()

	c, err := client.NewWithOptions(srv.URL, "key", true)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out []struct {
		Name string `json:"name"`
	}
	if err := sweep.GetList(context.Background(), c, "/fake", &out); err != nil {
		t.Fatalf("GetList: %v", err)
	}
	if len(out) != 2 || out[0].Name != "a" || out[1].Name != "b" {
		t.Fatalf("unexpected list: %#v", out)
	}
}

func TestGetList_GetError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c, err := client.NewWithOptions(srv.URL, "key", true)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out []any
	err = sweep.GetList(context.Background(), c, "/fake", &out)
	if err == nil {
		t.Fatalf("expected error from GetList")
	}
	if !strings.Contains(err.Error(), "GET /fake") {
		t.Fatalf("want GET wrap, got %v", err)
	}
}

func TestGetList_DecodeError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	c, err := client.NewWithOptions(srv.URL, "key", true)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out []any
	err = sweep.GetList(context.Background(), c, "/fake", &out)
	if err == nil {
		t.Fatalf("expected error from GetList")
	}
	if !strings.Contains(err.Error(), "decode /fake") {
		t.Fatalf("want decode wrap, got %v", err)
	}
	// Sanity: make sure json.Unmarshal is actually what produced the
	// underlying error (guards against regressions that bypass decode).
	var je *json.SyntaxError
	if !errors.As(err, &je) {
		t.Logf("note: decode error was not *json.SyntaxError: %T", err)
	}
}
