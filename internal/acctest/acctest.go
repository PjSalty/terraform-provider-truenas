// Package acctest provides shared acceptance-test helpers so every
// internal/provider/acc_*_test.go file can lean on a single copy of the
// boilerplate (env-check preflight, out-of-band client construction,
// random name generation, skip messages). The layout follows the standard
// Terraform provider acceptance-test helper pattern.
//
// These helpers are NOT inside a _test.go file because they need to be
// importable from test files in sibling packages. Nothing here should
// ever be called from production code — runtime refers only to the
// TF_ACC / TRUENAS_URL / TRUENAS_API_KEY environment variables already
// required by the plugin-testing framework itself.
package acctest

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/PjSalty/terraform-provider-truenas/internal/client"
)

// SkipMsg is the shared skip message used by every acceptance test when
// TF_ACC is not set. Keeping it in one place makes grepping the SKIPs in
// CI logs trivial.
const SkipMsg = "TF_ACC not set — skipping acceptance test"

// PreCheck verifies the environment is configured for acceptance testing
// against a real TrueNAS SCALE instance. Tests are expected to call this
// from their TestCase.PreCheck hook. TF_ACC gating happens at the top of
// each test via SkipIfNoAcc so PreCheck only runs for tests that actually
// intend to execute.
func PreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TRUENAS_URL") == "" {
		t.Fatal("TRUENAS_URL must be set for acceptance tests")
	}
	if os.Getenv("TRUENAS_API_KEY") == "" {
		t.Fatal("TRUENAS_API_KEY must be set for acceptance tests")
	}
}

// SkipIfNoAcc skips the current test when TF_ACC is not set. Returns
// true when the test was skipped so callers can short-circuit any setup
// that would otherwise run even after t.Skip.
func SkipIfNoAcc(t *testing.T) bool {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip(SkipMsg)
		return true
	}
	return false
}

// Client builds a live client.Client from the same environment variables
// the provider uses. This is exclusively for `_disappears` test checks
// that need to delete a resource out-of-band to verify the provider
// detects and recovers from the drift. Never use this for anything else —
// all other infrastructure mutations must flow through Terraform.
func Client() (*client.Client, error) {
	url := os.Getenv("TRUENAS_URL")
	apiKey := os.Getenv("TRUENAS_API_KEY")
	if url == "" || apiKey == "" {
		return nil, fmt.Errorf("TRUENAS_URL and TRUENAS_API_KEY must be set")
	}
	insecure := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "true"
	return client.NewWithOptions(url, apiKey, insecure)
}

// Ctx returns a short-lived context suitable for out-of-band API calls
// made from within `_disappears` test check functions.
func Ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

// RandomName returns a unique resource name with the given prefix so that
// concurrent or back-to-back acceptance test runs never collide on the
// TrueNAS side. UnixNano has enough resolution that adjacent calls within
// a single test still produce distinct names.
func RandomName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// ShortSuffix returns a short numeric suffix (last 6 digits of UnixNano)
// for resources with tight length limits like POSIX usernames (32 chars)
// or group names. Collisions within a single second are possible in
// theory but astronomically unlikely in practice.
func ShortSuffix() int64 {
	return time.Now().UnixNano() % 1000000
}

// PIDSuffix is an alias of ShortSuffix used by tests that need a short
// numeric token for fields with length constraints.
func PIDSuffix() int64 {
	return ShortSuffix()
}
