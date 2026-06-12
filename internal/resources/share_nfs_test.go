package resources_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/PjSalty/terraform-provider-truenas/internal/acctest"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func TestAccNFSShare_basic(t *testing.T) {
	pool := testAccDatasetPool()
	// Dataset path used as the NFS export path.
	datasetName := "tf-acc-nfs-basic"
	resourceName := "truenas_share_nfs.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNFSShareDestroy(resourceName),
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccNFSShareConfigBasic(pool, datasetName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "readonly", "false"),
				),
			},
			// Import by numeric ID
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccNFSShare_withHosts(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-nfs-hosts"
	resourceName := "truenas_share_nfs.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNFSShareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNFSShareConfigWithHosts(pool, datasetName, []string{"10.0.0.1", "10.0.0.2"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hosts.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "hosts.0", "10.0.0.1"),
					resource.TestCheckResourceAttr(resourceName, "hosts.1", "10.0.0.2"),
				),
			},
			// Update hosts list
			{
				Config: testAccNFSShareConfigWithHosts(pool, datasetName, []string{"10.0.0.3"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "hosts.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "hosts.0", "10.0.0.3"),
				),
			},
		},
	})
}

func TestAccNFSShare_withNetworks(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-nfs-networks"
	resourceName := "truenas_share_nfs.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNFSShareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNFSShareConfigWithNetworks(pool, datasetName, "192.168.1.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "networks.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "networks.0", "192.168.1.0/24"),
				),
			},
		},
	})
}

func testAccCheckNFSShareDestroy(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return nil
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("NFS share ID not set")
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return fmt.Errorf("NFS share ID %q is not numeric: %w", rs.Primary.ID, err)
		}
		c, err := acctest.Client()
		if err != nil {
			return fmt.Errorf("building API client: %w", err)
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		_, err = c.GetNFSShare(ctx, id)
		if err == nil {
			return fmt.Errorf("NFS share %d still exists upstream after Terraform removed it", id)
		}
		if !wsclient.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking removal of NFS share %d: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNFSShareExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if _, err := c.GetNFSShare(ctx, id); err != nil {
			return fmt.Errorf("NFS share %d should exist but lookup failed: %w", id, err)
		}
		return nil
	}
}

func testAccCheckNFSShareDisappears(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("not found in state: %s", resourceName)
		}
		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}
		c, err := acctest.Client()
		if err != nil {
			return err
		}
		ctx, cancel := acctest.Ctx()
		defer cancel()
		if err := c.DeleteNFSShare(ctx, id); err != nil {
			return fmt.Errorf("out-of-band delete of NFS share %d failed: %w", id, err)
		}
		return nil
	}
}

func TestAccNFSShare_disappears(t *testing.T) {
	pool := testAccDatasetPool()
	datasetName := "tf-acc-nfsshare-disappears"
	resourceName := "truenas_share_nfs.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNFSShareDestroy(resourceName),
		Steps: []resource.TestStep{
			{
				Config: testAccNFSShareConfigBasic(pool, datasetName),
				Check:  testAccCheckNFSShareExists(resourceName),
			},
			{
				Config:             testAccNFSShareConfigBasic(pool, datasetName),
				Check:              testAccCheckNFSShareDisappears(resourceName),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccNFSShareConfigBasic(pool, datasetName string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "nfs_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_nfs" "test" {
  path    = truenas_dataset.nfs_parent.mount_point
  comment = "Terraform acceptance test"
  enabled = true
}
`, pool, datasetName)
}

func testAccNFSShareConfigWithHosts(pool, datasetName string, hosts []string) string {
	hostsStr := "["
	for i, h := range hosts {
		if i > 0 {
			hostsStr += ", "
		}
		hostsStr += fmt.Sprintf("%q", h)
	}
	hostsStr += "]"

	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "nfs_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_nfs" "test" {
  path    = truenas_dataset.nfs_parent.mount_point
  hosts   = %s
  enabled = true
}
`, pool, datasetName, hostsStr)
}

func testAccNFSShareConfigWithNetworks(pool, datasetName, network string) string {
	return fmt.Sprintf(`
provider "truenas" {}

resource "truenas_dataset" "nfs_parent" {
  pool = %q
  name = %q
}

resource "truenas_share_nfs" "test" {
  path     = truenas_dataset.nfs_parent.mount_point
  networks = [%q]
  enabled  = true
}
`, pool, datasetName, network)
}
