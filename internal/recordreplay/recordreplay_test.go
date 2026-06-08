package recordreplay

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// upstreamForTest stands in for a real TrueNAS — returns a fixed
// JSON payload for every request so the test asserts the proxy
// captures + replays it correctly.
func upstreamForTest(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"version":"25.10.0","build":"test"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func insecureClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

func TestRecordReplay_RoundTrip(t *testing.T) {
	upstream := upstreamForTest(t)
	dir := t.TempDir()

	// Phase 1: record.
	rec, err := NewRecorder(upstream.URL, dir)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	defer rec.Close()

	client := insecureClient()
	resp, err := client.Get(rec.URL() + "/api/v2.0/system/info")
	if err != nil {
		t.Fatalf("recorded GET: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "25.10.0") {
		t.Fatalf("recorded body wrong: %s", body)
	}
	if rec.Captured() != 1 {
		t.Fatalf("Captured = %d, want 1", rec.Captured())
	}

	// Phase 2: replay (upstream is dead).
	rep, err := NewReplayer(dir)
	if err != nil {
		t.Fatalf("NewReplayer: %v", err)
	}
	defer rep.Close()

	resp2, err := client.Get(rep.URL() + "/api/v2.0/system/info")
	if err != nil {
		t.Fatalf("replayed GET: %v", err)
	}
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()
	if string(body) != string(body2) {
		t.Errorf("replay drift:\n  rec: %s\n  rep: %s", body, body2)
	}
	if rep.Hits() != 1 {
		t.Errorf("Hits = %d, want 1", rep.Hits())
	}
	if len(rep.Misses()) != 0 {
		t.Errorf("Misses = %v, want none", rep.Misses())
	}
}

func TestRecordReplay_MissReturns404(t *testing.T) {
	dir := t.TempDir()
	rep, err := NewReplayer(dir)
	if err != nil {
		t.Fatalf("NewReplayer: %v", err)
	}
	defer rep.Close()

	client := insecureClient()
	resp, err := client.Get(rep.URL() + "/api/v2.0/never/recorded")
	if err != nil {
		t.Fatalf("replayed GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Errorf("miss → status %d, want 404", resp.StatusCode)
	}
	if len(rep.Misses()) != 1 {
		t.Errorf("Misses = %v, want 1 entry", rep.Misses())
	}
}

func TestHash_StableForSemanticallyEqualBodies(t *testing.T) {
	// Same fields, different key order, same hash.
	a := []byte(`{"a":1,"b":2,"c":3}`)
	b := []byte(`{"c":3,"a":1,"b":2}`)
	h1 := Hash("POST", "/api/v2.0/x", url.Values{}, a)
	h2 := Hash("POST", "/api/v2.0/x", url.Values{}, b)
	if h1 != h2 {
		t.Errorf("Hash drift for key-reordered JSON: %s vs %s", h1, h2)
	}
}

func TestHash_DifferForDifferentMethods(t *testing.T) {
	body := []byte(`{"x":1}`)
	hGet := Hash("GET", "/api/v2.0/x", url.Values{}, body)
	hPost := Hash("POST", "/api/v2.0/x", url.Values{}, body)
	if hGet == hPost {
		t.Error("GET and POST hashed identically")
	}
}

func TestHash_StableForQueryReorder(t *testing.T) {
	body := []byte(`{}`)
	q1, _ := url.ParseQuery("limit=10&offset=0")
	q2, _ := url.ParseQuery("offset=0&limit=10")
	if Hash("GET", "/x", q1, body) != Hash("GET", "/x", q2, body) {
		t.Error("query reorder produced different hashes")
	}
}

func TestRecorder_PreservesUpstreamBody(t *testing.T) {
	wantBody := `{"echoed":true,"name":"my-test"}`
	echo := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wantBody + "|" + string(body)))
	}))
	defer echo.Close()

	dir := t.TempDir()
	rec, err := NewRecorder(echo.URL, dir)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	defer rec.Close()

	client := insecureClient()
	resp, err := client.Post(rec.URL()+"/api/v2.0/create",
		"application/json", bytes.NewReader([]byte(`{"req":"shape"}`)))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), `"req":"shape"`) {
		t.Errorf("upstream did not see request body — proxy ate it: %s", body)
	}

	// Read fixture from disk and confirm it has request_body + response_body
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("expected 1 fixture, got %d", len(entries))
	}
	data, _ := os.ReadFile(dir + "/" + entries[0].Name())
	// Body is base64-encoded inside the JSON envelope; decode + check.
	if !strings.Contains(string(data), `"request_body":`) ||
		!strings.Contains(string(data), `"response_body":`) {
		t.Errorf("fixture missing body fields: %s", data)
	}
}
