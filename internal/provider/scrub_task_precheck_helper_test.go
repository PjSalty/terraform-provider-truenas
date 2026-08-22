package provider

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// testAccRequireFreeScrubTask skips when the target pool already has a
// scrub task. Duplicated from the resources package: the two test
// packages share no helper file. Deliberately not named acc_*_test.go,
// which TestIdempotencyCheckCoverage scans for real acceptance tests.
//
// Middleware permits exactly one scrub task per pool
// ("A scrub with this pool already exists"), so a box that was set up
// with a scrub schedule, or that kept one from an aborted run, fails
// every scrub-task test identically. That is an environment collision,
// not a provider defect, and it should read as a skip.
func testAccRequireFreeScrubTask(t *testing.T) {
	t.Helper()
	url, key := os.Getenv("TRUENAS_URL"), os.Getenv("TRUENAS_API_KEY")
	if url == "" || key == "" {
		return // testAccPreCheck reports missing credentials
	}
	insecure := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "true" ||
		os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY") == "1"

	ctx := context.Background()
	c, err := wsclient.New(ctx, url, key, insecure)
	if err != nil {
		t.Skipf("skipping: cannot reach TrueNAS to check for an existing scrub task: %v", err)
	}
	defer func() { _ = c.Close() }()

	raw, err := c.Call(ctx, "pool.scrub.query", nil,
		wsclient.CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Skipf("skipping: pool.scrub.query failed: %v", err)
	}
	var tasks []struct {
		ID          int    `json:"id"`
		PoolName    string `json:"pool_name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(raw, &tasks); err != nil {
		return // shape changed; let the test run and fail honestly
	}
	if len(tasks) > 0 {
		t.Skipf("skipping: scrub task %d already exists for pool %q, and middleware allows one per pool. "+
			"Remove it or run the scrub-task tests on a box without one.",
			tasks[0].ID, tasks[0].PoolName)
	}
}
