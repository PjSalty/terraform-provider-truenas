package provider

import (
	"os"
	"regexp"
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
