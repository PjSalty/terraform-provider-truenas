// Package sweep provides shared infrastructure used by the resource
// sweepers in internal/provider/sweeper_test.go. Sweepers run as
// test-cleanup glue when the framework invokes them via TF_ACC=1
// `go test -sweep`. They list dangling test fixtures and delete the
// ones whose name carries the canonical acceptance-test prefix.
//
// Listing goes over the same JSON-RPC WebSocket as production resource
// I/O, via internal/wsclient. It used to issue HTTP GETs against
// /api/v2.0 instead, on the reasoning that the collection-list
// endpoints had no typed wsclient equivalent and it was not worth
// dragging a REST client in for a handful of GETs. TrueNAS 26.0
// removed the REST API, so those GETs answer 404, and because the
// plugin-testing framework aborts a sweep run on the first sweeper
// error, one dead call stopped every later sweeper from running.
// QueryList calls "<namespace>.query" instead, which needs no typed
// wrapper per resource and no second transport.
package sweep

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// AcctestPrefix is the canonical name prefix every acceptance test
// resource carries. Sweepers compare fixture names against this
// prefix before destroying, anything not starting with it is left
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

// QueryList calls a JSON-RPC "<namespace>.query" method and unmarshals
// the result into out.
//
// This replaced an HTTP GET against /api/v2.0. TrueNAS 26.0 removed the
// REST API outright: every such request answers 404, which made the
// sweeper that hit it return an error, and the plugin-testing framework
// aborts the whole sweep run on the first sweeper error. One dead REST
// call therefore stopped every later sweeper from running at all, so
// acceptance-test litter accumulated silently until a rerun collided
// with it.
//
// Callers pass the WHOLE method ("iscsi.portal.query"), not just the
// namespace. That is deliberate: scripts/api-drift.sh scans the source for
// method-name literals and checks each against the newest upstream API
// models, and it exempts the auto-generated CRUD verbs by looking at the
// last dotted segment. A bare namespace, with no verb on the end, reads
// to that scan as a method upstream does not have, and gets reported as a
// removal that is not real. Spelling the whole call out keeps the gate
// honest and makes the calls greppable.
func QueryList(ctx context.Context, c *wsclient.Client, method string, out interface{}) error {
	if c == nil {
		return fmt.Errorf("sweep.QueryList: nil client")
	}
	result, err := c.Call(ctx, method, nil,
		wsclient.CallOptions{Read: true, Idempotent: true})
	if err != nil {
		return fmt.Errorf("query %s: %w", method, err)
	}
	if err := json.Unmarshal(result, out); err != nil {
		return fmt.Errorf("decode %s: %w", method, err)
	}
	return nil
}

// LiveAPIKeyID returns the row id embedded in TRUENAS_API_KEY, the
// credential the sweepers themselves authenticate with.
//
// TrueNAS API keys are shaped "<id>-<secret>", and middleware resolves
// the id exactly this way: api_key.authenticate does
// int(key.split('-', 1)[0]). Deriving it the same way means we agree
// with the box about which key is in use. Only the id is read, never
// the secret half, and the secret is never logged or returned.
//
// ok is false when the var is unset or is not id-shaped. A caller that
// gets false does not know which key it is holding and so must not
// delete any of them.
//
// This exists because the api_key sweeper destroys every key carrying
// AcctestPrefix, and a test key is conventionally named "tf-acc-...".
// It therefore matched its own credential, revoked it mid-run, and left
// every sweeper scheduled after it panicking unauthenticated.
func LiveAPIKeyID() (int, bool) {
	idPart, _, found := strings.Cut(os.Getenv("TRUENAS_API_KEY"), "-")
	if !found {
		return 0, false
	}
	id, err := strconv.Atoi(idPart)
	if err != nil {
		return 0, false
	}
	return id, true
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
