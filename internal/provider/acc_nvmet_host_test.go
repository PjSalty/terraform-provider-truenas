package provider

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// randomUUID returns a random UUID v4 string. NVMe-oF hostnqn values
// that start with `nqn.2014-08.org.nvmexpress:uuid:` require a
// canonical 8-4-4-4-12 hex UUID tail, so `shortSuffix()` was
// insufficient — TrueNAS rejects arbitrary suffixes.
func randomUUID(t *testing.T) string {
	t.Helper()
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("randomUUID: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // v4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	h := hex.EncodeToString(b)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}

func TestAccNVMetHostResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	hostnqn := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_host" "test" {
  hostnqn = %q
}
`, hostnqn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "hostnqn", hostnqn),
					resource.TestCheckResourceAttrSet("truenas_nvmet_host.test", "id"),
				),
			},
			{
				ResourceName:            "truenas_nvmet_host.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"dhchap_key", "dhchap_ctrl_key"},
			},
		},
	})
}

// dhchapKey1 and dhchapKey2 are real DHCHAP-1 keys with HMAC=00 (none),
// generated as: 32 random secret bytes + 4-byte CRC32 → base64. SCALE
// 25.10 validates the key structure ("Unexpected key termination") so
// arbitrary base64 payloads are rejected. These are static fixtures —
// the secret material is dummy and the keys are never used for real
// NVMe-oF authentication during acceptance testing.
const (
	dhchapKey1 = "DHHC-1:00:Fc+SyPGdJz1nxym2IWjVAQjiGGEMCPfWCwRJSzL2LbcAUjY9:"
	dhchapKey2 = "DHHC-1:00:my1eIfdl2vl0Snh6dKcJLxlI9x56L6d5PU0ULgTwjDjbR9B7:"
)

// TestAccNVMetHostResource_update exercises the updatable dhchap_key
// attribute round-trip (create with one value, then change it and
// confirm the provider issues an Update in place rather than replace).
func TestAccNVMetHostResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	hostnqn := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_host" "test" {
  hostnqn     = %q
  dhchap_key  = %q
  dhchap_hash = "SHA-256"
}
`, hostnqn, dhchapKey1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "hostnqn", hostnqn),
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "dhchap_key", dhchapKey1),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_host" "test" {
  hostnqn     = %q
  dhchap_key  = %q
  dhchap_hash = "SHA-512"
}
`, hostnqn, dhchapKey2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "hostnqn", hostnqn),
					resource.TestCheckResourceAttr("truenas_nvmet_host.test", "dhchap_key", dhchapKey2),
				),
			},
		},
	})
}

// TestAccNVMetHostResource_disappears deletes the resource out-of-band
// via a direct client call and verifies Terraform detects the drift by
// producing a non-empty plan (the resource must be re-created on next
// apply rather than leaving stale state).
func TestAccNVMetHostResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	hostnqn := fmt.Sprintf("nqn.2014-08.org.nvmexpress:uuid:%s", randomUUID(t))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_host" "test" {
  hostnqn = %q
}
`, hostnqn),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_host.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_nvmet_host.test"]
						if !ok {
							return fmt.Errorf("resource not found in state")
						}
						id, err := strconv.Atoi(rs.Primary.ID)
						if err != nil {
							return fmt.Errorf("parsing id: %w", err)
						}
						c, err := testAccClient()
						if err != nil {
							return err
						}
						ctx, cancel := testAccCtx()
						defer cancel()
						return c.DeleteNVMetHost(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
