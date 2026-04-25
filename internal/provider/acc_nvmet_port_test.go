package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// nvmetPortPort returns a pseudo-random TCP port in the 20000-29999
// range so parallel test runs don't collide on the same addr_trsvcid.
func nvmetPortPort() int {
	return 20000 + int(shortSuffix()%10000)
}

// TestAccNVMetPortResource_basic creates a TCP loopback port.
func TestAccNVMetPortResource_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	port := nvmetPortPort()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = %d
}
`, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_trtype", "TCP"),
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_traddr", "127.0.0.1"),
					resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_trsvcid", strconv.Itoa(port)),
					resource.TestCheckResourceAttrSet("truenas_nvmet_port.test", "id"),
				),
			},
			{
				ResourceName:      "truenas_nvmet_port.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// TestAccNVMetPortResource_update rolls the service port forward to
// exercise the PUT /nvmet/port/id/{id} path.
func TestAccNVMetPortResource_update(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	p1 := nvmetPortPort()
	p2 := p1 + 1
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = %d
}
`, p1),
				Check: resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_trsvcid", strconv.Itoa(p1)),
			},
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = %d
}
`, p2),
				Check: resource.TestCheckResourceAttr("truenas_nvmet_port.test", "addr_trsvcid", strconv.Itoa(p2)),
			},
		},
	})
}

// TestAccNVMetPortResource_disappears deletes the port out-of-band.
func TestAccNVMetPortResource_disappears(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	port := nvmetPortPort()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "truenas_nvmet_port" "test" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = %d
}
`, port),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("truenas_nvmet_port.test", "id"),
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["truenas_nvmet_port.test"]
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
						return c.DeleteNVMetPort(ctx, id)
					},
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
