// Package sweep provides the shared sweeper base infrastructure used by
// internal/provider/sweeper_test.go. The per-resource sweeper functions
// themselves live alongside the acceptance tests so they can reach
// unexported helpers; this package holds only the cross-cutting pieces:
// the acctest prefix filter, the collection-GET helper, and the log
// formatter. Layout mirrors hashicorp/terraform-provider-aws:internal/sweep.
package sweep

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// DefaultContextTimeout is the moderately long-lived deadline used by
// Ctx. TrueNAS list endpoints can be slow on a busy system so sweepers
// get a generous 5 minute budget.
const DefaultContextTimeout = 5 * time.Minute

// AcctestPrefixes lists every string that acceptance test fixtures use
// when generating random resource names. Any name that starts with one
// of these is considered a candidate for sweeping. Keep this list in
// sync with RandomName() callsites in acc_*_test.go.
var AcctestPrefixes = []string{
	"acct",
	"acctest",
	"acctest-",
	"tf-acc",
	"tf-acc-",
	"tfacc",
}

// Ctx returns a moderately long-lived context for a single sweeper run.
func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), DefaultContextTimeout)
}

// HasAcctestPrefix reports whether a resource name looks like an
// abandoned acceptance test fixture. This MUST be strict: every sweeper
// uses it to avoid wiping user data. A resource is considered a fixture
// only when its name starts with one of the well-known acctest prefixes.
func HasAcctestPrefix(name string) bool {
	n := strings.ToLower(name)
	for _, p := range AcctestPrefixes {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	return false
}

// DatasetIsAcctest reports whether a dataset/zvol ID belongs to an
// acceptance test fixture. Dataset IDs are full paths like
// "tank/acct-foo" or "test/acct/bar", so we look for "/acct" segments
// anywhere rather than requiring a prefix match. We never sweep
// datasets whose ID is just "tank" or "tank/system" etc. — only those
// containing an acctest-style segment.
func DatasetIsAcctest(id string) bool {
	lid := strings.ToLower(id)
	return strings.Contains(lid, "/acct") ||
		strings.Contains(lid, "/tf-acc") ||
		strings.Contains(lid, "/tfacc")
}

// Log is the standard log format used by every sweeper so the output
// of `go test -sweep=all -v` is easy to grep.
func Log(resourceType, action, name string, err error) {
	if err != nil {
		fmt.Printf("[sweep] %s %s %q: FAILED: %v\n", resourceType, action, name, err)
		return
	}
	fmt.Printf("[sweep] %s %s %q: OK\n", resourceType, action, name)
}

// GetList performs a GET against a list endpoint and unmarshals the
// response into the provided target slice. It exists because many
// TrueNAS resources don't have dedicated ListXxx methods on the client
// but are still listable via a simple GET on the collection URL.
func GetList(ctx context.Context, c *client.Client, path string, out interface{}) error {
	resp, err := c.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	if err := json.Unmarshal(resp, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
