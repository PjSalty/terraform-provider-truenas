package resources_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

// testAccRequireFreeWildcardPortal skips when the target TrueNAS already
// has an iSCSI portal bound to 0.0.0.0.
//
// The portal acceptance tests bind the wildcard address, and middleware
// permits exactly one portal on it, so a pre-existing wildcard portal
// (a CSI driver's, or litter from an aborted run) makes every one of
// them fail with "'0.0.0.0' IP is already in use". That is an
// environment collision, not a provider defect, and it should read as a
// skip with the offending portal named rather than as a wall of
// identical apply failures.
func testAccRequireFreeWildcardPortal(t *testing.T) {
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
		t.Skipf("skipping: cannot reach TrueNAS to check for a free wildcard portal: %v", err)
	}
	defer func() { _ = c.Close() }()

	raw, err := c.Call(ctx, "iscsi.portal.query", nil,
		wsclient.CallOptions{Read: true, Idempotent: true})
	if err != nil {
		t.Skipf("skipping: iscsi.portal.query failed: %v", err)
	}
	var portals []struct {
		ID      int    `json:"id"`
		Comment string `json:"comment"`
		Listen  []struct {
			IP string `json:"ip"`
		} `json:"listen"`
	}
	if err := json.Unmarshal(raw, &portals); err != nil {
		return // shape changed; let the test run and fail honestly
	}
	for _, p := range portals {
		for _, l := range p.Listen {
			if l.IP == "0.0.0.0" {
				t.Skipf("skipping: portal %d (%q) already binds 0.0.0.0, and middleware allows only one. "+
					"Remove it or run the iSCSI portal tests on a box without one.", p.ID, p.Comment)
			}
		}
	}
}
