package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccReplicationResource_encryptionInherit turns destination encryption
// on with the settings inherited from the parent dataset, the one form that
// needs no key, and off again, in place.
func TestAccReplicationResource_encryptionInherit(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	name := fmt.Sprintf("acctestrep%d", shortSuffix())
	src := randomName("acctestrepsrc")
	dst := randomName("acctestrepdst")
	prereqs := replicationPrereqs(src, dst)
	const res = "truenas_replication.test"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: prereqs + replicationOptionsConfig(name, "NONE", `
  also_include_naming_schema = ["auto-%Y-%m-%d_%H-%M"]
  encryption                 = true
  encryption_inherit         = true
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "encryption", "true"),
					resource.TestCheckResourceAttr(res, "encryption_inherit", "true"),
					// Left out and null in TrueNAS: the empty values.
					resource.TestCheckResourceAttr(res, "encryption_key", ""),
					resource.TestCheckResourceAttr(res, "encryption_key_format", ""),
					resource.TestCheckResourceAttr(res, "encryption_key_location", ""),
					resource.TestCheckResourceAttr(res, "netcat_active_side", ""),
					resource.TestCheckResourceAttr(res, "netcat_active_side_port_min", "0"),
				),
			},
			{
				Config: prereqs + replicationOptionsConfig(name, "NONE", `
  also_include_naming_schema = ["auto-%Y-%m-%d_%H-%M"]
  encryption                 = false
  encryption_inherit         = false
`),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "encryption", "false"),
					resource.TestCheckResourceAttr(res, "encryption_inherit", "false"),
				),
			},
		},
	})
}

// TestAccReplicationResource_sshNetcat creates an SSH+NETCAT task with its
// netcat attributes. As in TestAccReplicationResource_sshOneApply, TrueNAS
// only checks that the credential exists and does not connect, so the
// addresses are documentation addresses (RFC 5737).
func TestAccReplicationResource_sshNetcat(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	kpName := randomName("acctest-kp")
	credName := randomName("acctest-sshc")
	repName := fmt.Sprintf("acctestrepnc%d", shortSuffix())
	src := randomName("acctestrepsrc")
	priv, pub := kcGenTestKeyPair(t)
	hostKey := strings.TrimSpace(pub)
	const res = "truenas_replication.test"

	config := replicationPrereqs(src, src+"dst") + fmt.Sprintf(`
resource "truenas_keychain_credential" "kp" {
  name = %q
  type = "SSH_KEY_PAIR"
  attributes = {
    private_key = %q
    public_key  = %q
  }
}

resource "truenas_keychain_credential" "creds" {
  name = %q
  type = "SSH_CREDENTIALS"
  attributes = {
    host            = "192.0.2.10"
    port            = "22"
    username        = "root"
    private_key     = truenas_keychain_credential.kp.id
    remote_host_key = %q
    connect_timeout = "10"
  }
}

resource "truenas_replication" "test" {
  name                                = %q
  direction                           = "PUSH"
  transport                           = "SSH+NETCAT"
  ssh_credentials                     = tonumber(truenas_keychain_credential.creds.id)
  source_datasets                     = [truenas_dataset.src.id]
  target_dataset                      = "backup/%s"
  recursive                           = false
  auto                                = false
  enabled                             = false
  retention_policy                    = "NONE"
  also_include_naming_schema          = ["auto-%%Y-%%m-%%d_%%H-%%M"]
  netcat_active_side                  = "LOCAL"
  netcat_active_side_listen_address   = "192.0.2.20"
  netcat_active_side_port_min         = 2000
  netcat_active_side_port_max         = 2100
  netcat_passive_side_connect_address = "192.0.2.20"
}
`, kpName, priv, pub, credName, hostKey, repName, src)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "transport", "SSH+NETCAT"),
					resource.TestCheckResourceAttr(res, "netcat_active_side", "LOCAL"),
					resource.TestCheckResourceAttr(res, "netcat_active_side_listen_address", "192.0.2.20"),
					resource.TestCheckResourceAttr(res, "netcat_active_side_port_min", "2000"),
					resource.TestCheckResourceAttr(res, "netcat_active_side_port_max", "2100"),
					resource.TestCheckResourceAttr(res, "netcat_passive_side_connect_address", "192.0.2.20"),
				),
			},
		},
	})
}
