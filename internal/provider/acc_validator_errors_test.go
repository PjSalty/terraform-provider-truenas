package provider

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// These tests assert that custom validators reject malformed input at
// plan time, before any API call is issued. The configured fixtures
// never reach the upstream, so the tests are safe to run against any
// TrueNAS instance (including the test VM). They still require TF_ACC=1
// because the plugin-testing framework's validator path runs inside the
// resource.Test harness.
//
// Every test below targets one validator and one failure mode. Adding a
// new validator? Add a corresponding _rejects_* test here so the
// negative path is locked in.

// TestAccValidator_IPOrCIDR_rejectsInvalidIP covers the bare-IP branch
// of IPOrCIDR via the static_route gateway attribute. Garbage that
// contains no slash falls into net.ParseIP, which returns nil for
// anything that isn't a valid IPv4/IPv6 literal.
func TestAccValidator_IPOrCIDR_rejectsInvalidIP(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "bad_ip" {
  destination = "192.0.2.0/24"
  gateway     = "not-an-ip"
}
`,
				ExpectError: regexp.MustCompile(`(?i)invalid ip address`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_IPOrCIDR_rejectsInvalidCIDR covers the CIDR branch:
// strings containing a slash that net.ParseCIDR rejects.
func TestAccValidator_IPOrCIDR_rejectsInvalidCIDR(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "bad_cidr" {
  destination = "192.0.2.0/99"
  gateway     = "192.0.2.1"
}
`,
				ExpectError: regexp.MustCompile(`(?i)invalid cidr`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_IPOrCIDR_rejectsIPv4WithGarbageSuffix verifies the
// bare-IP branch refuses values like "192.0.2.1.5" (looks numeric but
// is not a valid address).
func TestAccValidator_IPOrCIDR_rejectsIPv4WithGarbageSuffix(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "garbage_octets" {
  destination = "192.0.2.0/24"
  gateway     = "192.0.2.1.5"
}
`,
				ExpectError: regexp.MustCompile(`(?i)invalid ip address`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_IPOrCIDR_rejectsCIDRWithTextHost covers the
// net.ParseCIDR branch with a host portion that fails parsing.
func TestAccValidator_IPOrCIDR_rejectsCIDRWithTextHost(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_static_route" "text_cidr" {
  destination = "not-a-network/24"
  gateway     = "192.0.2.1"
}
`,
				ExpectError: regexp.MustCompile(`(?i)invalid cidr`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_IPOrCIDR_acceptsIPv6 is a positive control: the
// validator must NOT reject a well-formed IPv6 address. If this test
// starts failing the validator has regressed.
func TestAccValidator_IPOrCIDR_acceptsIPv6(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// IPv6 documentation prefix per RFC 3849.
				Config: `
resource "truenas_static_route" "v6" {
  destination = "2001:db8::/32"
  gateway     = "2001:db8::1"
}
`,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

// TestAccValidator_InitScript_typeOneOf locks the init_script.type enum
// (COMMAND, SCRIPT) so adding a typo'd value at the .tf layer fails
// before reaching the API.
func TestAccValidator_InitScript_typeOneOf(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_init_script" "bad_type" {
  type    = "NOT_A_TYPE"
  command = "echo hi"
  when    = "POSTINIT"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute type value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_InitScript_whenOneOf locks the init_script.when enum
// (PREINIT, POSTINIT, SHUTDOWN).
func TestAccValidator_InitScript_whenOneOf(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_init_script" "bad_when" {
  type    = "COMMAND"
  command = "echo hi"
  when    = "WHENEVER"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute when value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_NVMetPort_trtypeOneOf locks the nvmet_port.addr_trtype
// enum (TCP, RDMA, FC). The transport-type API is the kind of value
// TrueNAS extends silently when new hardware ships — locking the enum
// surfaces it.
func TestAccValidator_NVMetPort_trtypeOneOf(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nvmet_port" "bad_trtype" {
  addr_trtype  = "QUIC"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = 4420
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute addr_trtype value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_NVMetPort_portTooLow rejects port 0.
func TestAccValidator_NVMetPort_portTooLow(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nvmet_port" "port_zero" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = 0
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute addr_trsvcid value must be between`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_NVMetPort_portTooHigh rejects port > 65535.
func TestAccValidator_NVMetPort_portTooHigh(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_nvmet_port" "port_huge" {
  addr_trtype  = "TCP"
  addr_traddr  = "127.0.0.1"
  addr_trsvcid = 70000
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute addr_trsvcid value must be between`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_ISCSITarget_modeOneOf locks the iscsi_target.mode enum
// (ISCSI, FC, BOTH).
func TestAccValidator_ISCSITarget_modeOneOf(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_target" "bad_mode" {
  name = "tf-acc-bad-mode"
  mode = "NEW_MODE"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute mode value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_Certificate_keyLengthOneOf locks the
// certificate.key_length enum (1024, 2048, 4096). 3072 is a real RSA
// size but TrueNAS does not accept it; the test verifies our enum
// guard mirrors the upstream.
func TestAccValidator_Certificate_keyLengthOneOf(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_certificate" "bad_key_length" {
  name        = "tf-acc-bad-keylen"
  create_type = "CERTIFICATE_CREATE_INTERNAL"
  key_length  = 3072
  key_type    = "RSA"
  lifetime    = 365
  digest_algorithm = "SHA256"
  common      = "example.com"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute key_length value must be one of`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_DNSNameserver_rejectsGarbage exercises the regex
// validator on dns_nameserver.address. Anything that isn't a valid
// IPv4 / IPv6 literal must fail at plan time.
func TestAccValidator_DNSNameserver_rejectsGarbage(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_dns_nameserver" "bad_ns" {
  address = "not-an-ip"
}
`,
				ExpectError: regexp.MustCompile(`(?i)valid ipv4 or ipv6 address`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_ISCSITarget_iqnRegex locks the regex validator on
// iscsi_target.name. TrueNAS requires the name segment that follows the
// IQN year-month authority prefix to match a specific character class.
func TestAccValidator_ISCSITarget_iqnRegex(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_target" "bad_iqn" {
  name = "UPPER_CASE_BAD"
  mode = "ISCSI"
}
`,
				// Per RFC 3720 IQN naming, target names must be lowercase.
				// The provider's stringvalidator.RegexMatches enforces this
				// at plan time.
				ExpectError: regexp.MustCompile(`(?i)attribute name`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_ISCSITargetExtent_lunidAtLeast verifies the
// iscsi_targetextent.lunid AtLeast(0) lower bound. (LUNID may be 0
// upward; negative values are nonsensical and the validator catches it.)
// terraform-plugin-framework actually treats Int64Attribute as a signed
// 64-bit integer, so writing -1 is syntactically valid and reaches the
// validator. (The framework converts a negative literal into Int64 cleanly.)
func TestAccValidator_ISCSITargetExtent_lunidAtLeast(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_targetextent" "bad_lunid" {
  target = 1
  extent = 1
  lunid  = -1
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute lunid value must be at least`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_Certificate_lifetimeOutOfRange covers the
// certificate.lifetime int64validator.Between(1, 36500) — values
// outside the [1, 36500] day range must be rejected before reaching
// the API.
func TestAccValidator_Certificate_lifetimeOutOfRange(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_certificate" "bad_lifetime" {
  name             = "tf-acc-bad-lifetime"
  create_type      = "CERTIFICATE_CREATE_INTERNAL"
  key_length       = 2048
  key_type         = "RSA"
  digest_algorithm = "SHA256"
  lifetime         = 36501
  common           = "example.com"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute lifetime value must be between`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_ISCSIPortal_portOutOfRange covers the
// iscsi_portal.listen[].port int64validator.Between(1, 65535).
func TestAccValidator_ISCSIPortal_portOutOfRange(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_iscsi_portal" "bad_port" {
  listen = [
    {
      ip   = "0.0.0.0"
      port = 70000
    },
  ]
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute listen\[0\].port value must be between`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_Cronjob_userTooLong locks the
// cronjob.user stringvalidator.LengthBetween(1, 32).
func TestAccValidator_Cronjob_userTooLong(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// 33 chars — POSIX usernames cap at 32.
				Config: `
resource "truenas_cronjob" "bad_user" {
  user            = "thisuserislongerthanthirtytwochars"
  command         = "true"
  schedule_minute = "*"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute user`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_Cronjob_commandEmpty locks the lower bound of
// cronjob.command stringvalidator.LengthBetween(1, 4096).
func TestAccValidator_Cronjob_commandEmpty(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_cronjob" "bad_command" {
  user            = "root"
  command         = ""
  schedule_minute = "*"
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute command`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_NVMetSubsys_nameTooLong locks the upper bound on
// nvmet_subsys.name (LengthBetween 1, 253 per NQN RFC 8009).
func TestAccValidator_NVMetSubsys_nameTooLong(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// 254 'x' characters — one over the 253-char limit.
				Config: fmt.Sprintf(`
resource "truenas_nvmet_subsys" "bad_name" {
  name = %q
}
`, strings.Repeat("x", 254)),
				ExpectError: regexp.MustCompile(`(?i)attribute name`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_MailConfig_portOutOfRange covers the
// mail_config.port int64validator.Between(1, 65535).
func TestAccValidator_MailConfig_portOutOfRange(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_mail_config" "bad_port" {
  fromemail = "test@example.com"
  fromname  = "test"
  outgoingserver = "smtp.example.com"
  port    = 0
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute port value must be between`),
				PlanOnly:    true,
			},
		},
	})
}

// TestAccValidator_UPSConfig_portOutOfRange covers the
// ups_config.remoteport int64validator.Between(1, 65535).
func TestAccValidator_UPSConfig_portOutOfRange(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip(skipMsg)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "truenas_ups_config" "bad_port" {
  identifier = "tf-acc"
  mode       = "MASTER"
  remoteport = 99999
}
`,
				ExpectError: regexp.MustCompile(`(?i)attribute remoteport value must be between`),
				PlanOnly:    true,
			},
		},
	})
}
