package provider

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccReplicationResource_sshOneApply creates an SSH key pair, an
// SSH_CREDENTIALS credential that uses it, and an SSH replication that uses
// the credential, all in one apply. Both references are unknown at plan time,
// which is the case GitHub #43 reported for ssh_credentials.
//
// replication.create validates only that the credential exists and is of
// type SSH_CREDENTIALS; it does not connect. The host is therefore a
// documentation address (RFC 5737) and the host key is the generated public
// key: nothing here reaches a real SSH server.
//
// The SSH_CREDENTIALS attributes port, private_key and connect_timeout are
// integers upstream, and middleware validates strictly. All six are set so
// the read-back matches the configuration.
func TestAccReplicationResource_sshOneApply(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	kpName := randomName("acctest-kp")
	credName := randomName("acctest-sshc")
	repName := fmt.Sprintf("acctestrepssh%d", shortSuffix())
	src := randomName("acctestrepsrc")
	priv, pub := kcGenTestKeyPair(t)
	hostKey := strings.TrimSpace(pub)

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
  name                       = %q
  direction                  = "PUSH"
  transport                  = "SSH"
  ssh_credentials            = tonumber(truenas_keychain_credential.creds.id)
  source_datasets            = [truenas_dataset.src.id]
  target_dataset             = "backup/%s"
  recursive                  = false
  auto                       = false
  enabled                    = false
  retention_policy           = "NONE"
  also_include_naming_schema = ["auto-%%Y-%%m-%%d_%%H-%%M"]
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
					resource.TestCheckResourceAttr("truenas_keychain_credential.creds", "type", "SSH_CREDENTIALS"),
					resource.TestCheckResourceAttr("truenas_replication.test", "transport", "SSH"),
					resource.TestCheckResourceAttrPair(
						"truenas_replication.test", "ssh_credentials",
						"truenas_keychain_credential.creds", "id",
					),
				),
			},
		},
	})
}
