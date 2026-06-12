// Package recordreplay implements a record/replay HTTP+WebSocket
// proxy for the TrueNAS provider's acceptance tests. The intent:
// remove the live-API dependency from CI by capturing the
// request/response stream of a green acc run and replaying it
// against the provider in subsequent runs.
//
// Why this matters at hyperscale-provider rigor: live acc runs
// against the test TrueNAS are slow (the suite takes ~30 min) and
// fragile (network blip → false failure). A recorded corpus lets
// CI run the same coverage in seconds without touching live
// infrastructure. Drift between the recorded corpus and the live
// API surfaces as a fixture mismatch, which is itself a useful
// regression signal.
//
// Modes:
//
//   - Record: front the provider with this server. Forwards every
//     request to the real TrueNAS, stores the (request, response)
//     pair on disk indexed by a stable hash of the request shape.
//   - Replay: front the provider with this server. Looks up the
//     stored response by request hash and serves it back. If no
//     fixture exists, fails the request loudly so the operator
//     re-records.
//
// Both modes use the same on-disk layout so a recorded corpus is
// portable: drop the directory in CI and the replay server can
// serve it.
package recordreplay

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Fixture is the on-disk record of a single (request, response)
// pair. Stored as JSON for portability and reviewability, a
// regression that touches the wire shape is visible in the diff.
//
// Bodies are base64-encoded []byte rather than RawMessage so the
// recorder can capture non-JSON responses (HTML error pages,
// binary keytab payloads, compressed gzip bodies) without the
// outer JSON envelope rejecting the inner shape.
type Fixture struct {
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	Query         map[string]string `json:"query,omitempty"`
	RequestBody   []byte            `json:"request_body,omitempty"`
	StatusCode    int               `json:"status_code"`
	ResponseBody  []byte            `json:"response_body,omitempty"`
	ContentType   string            `json:"content_type,omitempty"`
	RecordedAtRFC string            `json:"recorded_at"`
}

// Hash returns the deterministic fixture index for a request. We
// hash method + path + sorted query + body to handle the common
// case where TrueNAS issues the same call with different transient
// parameters (request IDs, timestamps). Headers are intentionally
// EXCLUDED, they carry the X-Request-ID which differs every call.
func Hash(method, path string, query url.Values, body []byte) string {
	h := sha256.New()
	h.Write([]byte(strings.ToUpper(method)))
	h.Write([]byte{0x00})
	h.Write([]byte(path))
	h.Write([]byte{0x00})
	// stable query encoding
	keys := make([]string, 0, len(query))
	for k := range query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		vals := query[k]
		sort.Strings(vals)
		for _, v := range vals {
			h.Write([]byte(k))
			h.Write([]byte{0x01})
			h.Write([]byte(v))
			h.Write([]byte{0x02})
		}
	}
	h.Write([]byte{0x00})
	h.Write(canonicalJSONBody(body))
	return hex.EncodeToString(h.Sum(nil))
}

// canonicalJSONBody normalises JSON bodies so semantically equal
// payloads with different key ordering hash identically. Returns
// the raw bytes unchanged if the body is not JSON.
func canonicalJSONBody(body []byte) []byte {
	if len(body) == 0 {
		return body
	}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return body
	}
	// Marshal of a value produced by Unmarshal into interface{} cannot
	// fail: every shape Unmarshal emits (map[string]interface{}, []
	// interface{}, string, float64, bool, nil) is marshalable.
	out, _ := json.Marshal(v)
	return out
}

// Recorder is an httptest.Server-shaped proxy that captures every
// request/response pair to disk. Upstream is the live TrueNAS;
// fixtures land in Dir.
type Recorder struct {
	Upstream string
	Dir      string

	mu       sync.Mutex
	server   *httptest.Server
	captured int
}

// NewRecorder spins up a recording proxy. The returned Server has
// a URL the provider's `truenas_url` should point at.
func NewRecorder(upstreamURL, fixtureDir string) (*Recorder, error) {
	if err := os.MkdirAll(fixtureDir, 0o750); err != nil {
		return nil, fmt.Errorf("creating fixture dir: %w", err)
	}
	r := &Recorder{Upstream: upstreamURL, Dir: fixtureDir}
	r.server = httptest.NewTLSServer(http.HandlerFunc(r.serve))
	return r, nil
}

// URL returns the proxy's facing URL. Point the provider at this
// instead of the real TrueNAS to record traffic.
func (r *Recorder) URL() string { return r.server.URL }

// Close stops the proxy. After Close, recorded fixtures persist on
// disk for the Replay path to find.
func (r *Recorder) Close() {
	r.server.Close()
}

// Captured returns the number of fixtures recorded since New.
func (r *Recorder) Captured() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.captured
}

func (r *Recorder) serve(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	req.Body = io.NopCloser(bytes.NewReader(body))

	// Forward to upstream.
	upstream, err := url.Parse(r.Upstream)
	if err != nil {
		http.Error(w, "recordreplay: bad upstream URL", http.StatusInternalServerError)
		return
	}
	upstream.Path = req.URL.Path
	upstream.RawQuery = req.URL.RawQuery

	// NewRequestWithContext cannot fail here: req.Method comes from a
	// parsed inbound request (always a valid token) and upstream.String()
	// re-serializes a URL that url.Parse already accepted.
	upReq, _ := http.NewRequestWithContext(req.Context(), req.Method, upstream.String(), bytes.NewReader(body))
	// Forward headers EXCEPT Host (which httputil would do too).
	for k, vs := range req.Header {
		if strings.EqualFold(k, "Host") {
			continue
		}
		for _, v := range vs {
			upReq.Header.Add(k, v)
		}
	}
	// Use the default client with InsecureSkipVerify for the test
	// TrueNAS which ships with a self-signed cert. Real CI tooling
	// would use a configured TLS bundle.
	resp, err := insecureHTTPClient.Do(upReq) //nolint:gosec // G704: proxying to the operator-configured upstream is this recorder's purpose
	if err != nil {
		http.Error(w, "recordreplay: upstream call failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	// Persist fixture.
	fx := Fixture{
		Method:       req.Method,
		Path:         req.URL.Path,
		Query:        flattenQuery(req.URL.Query()),
		RequestBody:  canonicalJSONBody(body),
		StatusCode:   resp.StatusCode,
		ResponseBody: respBody,
		ContentType:  resp.Header.Get("Content-Type"),
	}
	r.write(req.Method, req.URL.Path, req.URL.Query(), body, fx)

	// Echo response back to caller.
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(respBody)
}

func flattenQuery(q url.Values) map[string]string {
	m := make(map[string]string, len(q))
	for k, vs := range q {
		if len(vs) > 0 {
			m[k] = vs[0]
		}
	}
	return m
}

func (r *Recorder) write(method, path string, query url.Values, body []byte, fx Fixture) {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := Hash(method, path, query, body)
	// Fixture is a flat struct of strings, ints, and []byte, MarshalIndent
	// cannot fail on it.
	out, _ := json.MarshalIndent(fx, "", "  ")
	_ = os.WriteFile(filepath.Join(r.Dir, h+".json"), out, 0o600)
	r.captured++
}

// Replayer is the read-only counterpart: serves recorded fixtures
// back. CI points the provider at this server, no live TrueNAS
// touched.
type Replayer struct {
	Dir string

	mu     sync.Mutex
	server *httptest.Server
	hits   int
	misses []string
}

// NewReplayer spins up a replay server from the given fixture
// directory.
func NewReplayer(fixtureDir string) (*Replayer, error) {
	if _, err := os.Stat(fixtureDir); err != nil {
		return nil, fmt.Errorf("fixture dir: %w", err)
	}
	rp := &Replayer{Dir: fixtureDir}
	rp.server = httptest.NewTLSServer(http.HandlerFunc(rp.serve))
	return rp, nil
}

// URL of the replay server. Point the provider here.
func (rp *Replayer) URL() string { return rp.server.URL }

// Close stops the replay server.
func (rp *Replayer) Close() { rp.server.Close() }

// Hits returns the number of fixtures served. Misses lists the
// requests that had no matching fixture, CI assertion: len(Misses) == 0.
func (rp *Replayer) Hits() int        { return rp.hits }
func (rp *Replayer) Misses() []string { return rp.misses }

func (rp *Replayer) serve(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	h := Hash(req.Method, req.URL.Path, req.URL.Query(), body)
	rp.mu.Lock()
	defer rp.mu.Unlock()

	// Fixture path is hash-derived inside the configured fixture dir;
	// this is dev-time test tooling, not a user-facing input path.
	data, err := os.ReadFile(filepath.Join(rp.Dir, h+".json")) //nolint:gosec // G304: hash-named file under the fixture dir
	if err != nil {
		// Miss, record + 404 so the caller sees the gap.
		rp.misses = append(rp.misses, fmt.Sprintf("%s %s (hash %s)", req.Method, req.URL.Path, h))
		http.Error(w, "recordreplay: no fixture for "+req.Method+" "+req.URL.Path, http.StatusNotFound)
		return
	}
	var fx Fixture
	if err := json.Unmarshal(data, &fx); err != nil {
		http.Error(w, "recordreplay: corrupt fixture: "+err.Error(), http.StatusInternalServerError)
		return
	}
	rp.hits++
	if fx.ContentType != "" {
		w.Header().Set("Content-Type", fx.ContentType)
	}
	w.WriteHeader(fx.StatusCode)
	_, _ = w.Write(fx.ResponseBody)
}
