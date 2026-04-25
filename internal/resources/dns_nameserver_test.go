package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDNSNameserver_basic(t *testing.T) {
	resourceName := "truenas_dns_nameserver.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read
			{
				Config: testAccDNSNameserverConfigBasic("8.8.8.8", "8.8.4.4"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "nameserver1", "8.8.8.8"),
					resource.TestCheckResourceAttr(resourceName, "nameserver2", "8.8.4.4"),
					resource.TestCheckResourceAttr(resourceName, "id", "network_config"),
				),
			},
			// Update nameservers
			{
				Config: testAccDNSNameserverConfigBasic("1.1.1.1", "1.0.0.1"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "nameserver1", "1.1.1.1"),
					resource.TestCheckResourceAttr(resourceName, "nameserver2", "1.0.0.1"),
				),
			},
			// Import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDNSNameserverConfigBasic(ns1, ns2 string) string {
	return `
provider "truenas" {}

resource "truenas_dns_nameserver" "test" {
  nameserver1 = "` + ns1 + `"
  nameserver2 = "` + ns2 + `"
}
`
}
