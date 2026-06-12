// Package sweep provides shared infrastructure used by the resource
// sweepers in internal/provider/sweeper_test.go. Sweepers run as
// test-cleanup glue when the framework invokes them via TF_ACC=1
// `go test -sweep`. They list dangling test fixtures via direct
// HTTP GETs against the TrueNAS REST API and delete by name when
// the name carries the canonical acceptance-test prefix.
//
// v2.0 cutover note: production resource I/O is JSON-RPC over
// WebSocket via internal/wsclient. The sweepers continue to use
// REST GETs because the collection-list endpoints have no direct
// typed equivalents on the wsclient call surface — instead of
// dragging the full REST client into sweep just for a handful of
// GETs, we issue them inline against /api/v2.0/<path> here. The
// transient http.Client is owned by GetList; production code never
// touches it.
package sweep

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AcctestPrefix is the canonical name prefix every acceptance test
// resource carries. Sweepers compare fixture names against this
// prefix before destroying — anything not starting with it is left
// alone, protecting any non-test resources on the target TrueNAS.
const AcctestPrefix = "tf-acc-"

// Ctx returns a fresh context + cancel function with a generous but
// bounded deadline. Sweepers run unattended at the end of an acc
// session; the deadline guards against a hung TrueNAS hanging the
// whole CI job.
func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Minute)
}

// HasAcctestPrefix reports whether name begins with AcctestPrefix.
// Used by every sweeper to gate destruction.
func HasAcctestPrefix(name string) bool {
	return strings.HasPrefix(name, AcctestPrefix)
}

// DatasetIsAcctest reports whether a dataset id (path-shaped like
// "pool/tf-acc-foo") belongs to the acctest suite. The check looks
// at the final path component because TrueNAS' dataset ids carry
// the full pool prefix and the sweeper only cares about the leaf.
func DatasetIsAcctest(id string) bool {
	idx := strings.LastIndex(id, "/")
	if idx < 0 {
		return HasAcctestPrefix(id)
	}
	return HasAcctestPrefix(id[idx+1:])
}

// GetList issues an HTTP GET against the TrueNAS REST API at the
// given path (e.g. "/iscsi/portal") and unmarshals the response into
// the provided target slice. Builds the http.Client and base URL
// from TRUENAS_URL / TRUENAS_API_KEY / TRUENAS_INSECURE_SKIP_VERIFY
// env vars — the same vars the production wsclient binds to. Used
// only by sweepers; production resource I/O flows over WebSocket.
func GetList(ctx context.Context, _ interface{}, path string, out interface{}) error {
	baseURL := os.Getenv("TRUENAS_URL")
	apiKey := os.Getenv("TRUENAS_API_KEY")
	insecure := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "true" ||
		os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "1"
	if baseURL == "" || apiKey == "" {
		return fmt.Errorf("sweep.GetList: TRUENAS_URL + TRUENAS_API_KEY must be set")
	}

	hc := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // test cleanup against TRUENAS_TEST_POOL only; insecure is the documented test posture
		},
	}
	url := strings.TrimRight(baseURL, "/") + "/api/v2.0" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody) //nolint:gosec // G704: dev-time sweeper hits the operator-configured TRUENAS_URL
	if err != nil {
		return fmt.Errorf("sweep.GetList: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := hc.Do(req) //nolint:gosec // G704: same operator-configured target as above
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body %s: %w", path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// Log emits a structured one-line message for a sweeper action. Used
// by the per-resource sweepers in internal/provider/sweeper_test.go
// to surface what was destroyed during test cleanup.
func Log(resourceType, action, name string, err error) {
	if err != nil {
		fmt.Printf("sweep[%s] %s %s: ERROR %v\n", resourceType, action, name, err)
		return
	}
	fmt.Printf("sweep[%s] %s %s: ok\n", resourceType, action, name)
}
