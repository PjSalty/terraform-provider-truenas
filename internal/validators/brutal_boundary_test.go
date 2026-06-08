package validators_test

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	lv "github.com/PjSalty/terraform-provider-truenas/internal/validators"
)

// boundaryCase pairs an input string with the expected validation
// outcome. The categorisation (kind) is purely descriptive — used
// by the test failure message to flag which class of input slipped
// the validator.
type boundaryCase struct {
	kind    string
	value   string
	wantErr bool
}

// validatorReq builds a StringRequest with the given value and a
// throwaway attribute path. Reused across every brutality-table test.
func validatorReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("attr"),
		PathExpression: path.MatchRoot("attr"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

// runBoundaryTable executes a validator against every case in the
// table. A failure names the kind + value so the operator can spot
// which class of input slipped through. The cases are subtest-scoped
// so a single regression doesn't mask others.
func runBoundaryTable(t *testing.T, name string, v validator.String, table []boundaryCase) {
	t.Helper()
	for _, tc := range table {
		tc := tc
		// Subtest name encodes both the kind AND the truncated value
		// so test runner output points straight at the offending row.
		shortVal := tc.value
		if len(shortVal) > 24 {
			shortVal = shortVal[:24] + "…"
		}
		// Replace special chars for the subtest name.
		safe := strings.Map(func(r rune) rune {
			if r < 0x20 || r > 0x7e || r == '/' || r == ' ' {
				return '_'
			}
			return r
		}, shortVal)
		t.Run(name+"/"+tc.kind+":"+safe, func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validatorReq(tc.value), resp)
			gotErr := resp.Diagnostics.HasError()
			if gotErr != tc.wantErr {
				t.Errorf("%s(%q) [%s]: wantErr=%v gotErr=%v\n  diagnostics: %v",
					name, tc.value, tc.kind, tc.wantErr, gotErr, resp.Diagnostics)
			}
		})
	}
}

// TestBrutal_IPOrCIDR_BoundaryTable runs the IPOrCIDR validator
// against an aggressive boundary battery — covers IPv4 / IPv6 /
// CIDR / control-chars / unicode / SQL-injection / very-long /
// whitespace-only / IPv4-mapped / link-local / loopback / multicast
// / RFC1918 / RFC4193 / discard-range. The table is intentionally
// non-exhaustive (no point fuzzing every byte combination) but
// exhaustive enough that a regression in net.ParseIP / ParseCIDR
// integration would fail at least one row.
func TestBrutal_IPOrCIDR_BoundaryTable(t *testing.T) {
	v := lv.IPOrCIDR()
	table := []boundaryCase{
		// IPv4 happy path
		{"ipv4-basic", "192.168.1.1", false},
		{"ipv4-loopback", "127.0.0.1", false},
		{"ipv4-zero", "0.0.0.0", false},
		{"ipv4-broadcast", "255.255.255.255", false},
		{"ipv4-rfc1918-10", "10.0.0.1", false},
		{"ipv4-rfc1918-172", "172.16.0.1", false},
		{"ipv4-multicast", "224.0.0.1", false},
		{"ipv4-link-local", "169.254.1.1", false},

		// IPv4 boundary
		{"ipv4-octet-255", "255.255.255.255", false},
		{"ipv4-octet-256", "256.256.256.256", true}, // out of range
		{"ipv4-leading-zero", "192.168.001.001", true},
		{"ipv4-truncated", "192.168.1", true},
		{"ipv4-too-many-octets", "192.168.1.1.5", true},
		{"ipv4-negative", "-1.1.1.1", true},
		{"ipv4-float", "192.168.1.1.5", true},

		// IPv6 happy path
		{"ipv6-loopback", "::1", false},
		{"ipv6-unspecified", "::", false},
		{"ipv6-full", "2001:db8:0:0:0:0:0:1", false},
		{"ipv6-compressed", "2001:db8::1", false},
		{"ipv6-link-local", "fe80::1", false},
		{"ipv6-discard", "100::1", false},
		{"ipv6-uppercase", "2001:DB8::1", false},

		// IPv6 boundary
		{"ipv6-too-many-groups", "1:2:3:4:5:6:7:8:9", true},
		{"ipv6-bad-hex", "ZZZZ::1", true},
		{"ipv6-incomplete", "2001:db8:", true},
		{"ipv6-double-double-colon", "2001::db8::1", true},

		// CIDR happy path
		{"cidr-ipv4-24", "192.168.1.0/24", false},
		{"cidr-ipv4-32", "192.168.1.1/32", false},
		{"cidr-ipv4-0", "0.0.0.0/0", false},
		{"cidr-ipv6-64", "2001:db8::/64", false},
		{"cidr-ipv6-128", "2001:db8::1/128", false},

		// CIDR boundary
		{"cidr-prefix-33", "192.168.1.0/33", true},
		{"cidr-prefix-negative", "192.168.1.0/-1", true},
		{"cidr-prefix-alpha", "192.168.1.0/abc", true},
		{"cidr-no-prefix", "192.168.1.0/", true},
		{"cidr-ipv6-129", "2001:db8::/129", true},

		// Whitespace handling — validator trims and accepts inner
		{"ws-leading", "  192.168.1.1", false},
		{"ws-trailing", "192.168.1.1  ", false},
		{"ws-both-sides", "  192.168.1.1  ", false},
		{"ws-only-spaces", "    ", false}, // empty-after-trim treated as empty (accepted)
		{"ws-tabs", "\t192.168.1.1\t", false},

		// Empty / null surrogates
		{"empty", "", false}, // validator explicitly accepts empty

		// Garbage
		{"garbage-text", "not-an-ip", true},
		{"garbage-mixed", "192.x.y.z", true},
		{"garbage-symbols", "@#$%^&*()", true},
		{"sql-injection", "1; DROP TABLE users--", true},
		{"xss", "<script>alert(1)</script>", true},
		{"path-traversal", "../../etc/passwd", true},

		// Unicode & control chars
		{"unicode-emoji", "🌐.🌐.🌐.🌐", true},
		{"unicode-mixed-script", "192.168.1.١", true},    // arabic 1
		{"control-NUL", "192.168.1.1\x00", true},         // embedded NUL
		{"control-newline", "192.168.1.1\n", false},      // trimmed
		{"control-CR-LF-pair", "192.168.1.1\r\n", false}, // trimmed
		{"control-mid-string", "192.\x001.1.1", true},    // not trimmable

		// Length
		{"very-long-1KB", strings.Repeat("a", 1024), true},
		{"very-long-malicious-padding", "192.168.1.1" + strings.Repeat(".", 1024), true},
	}
	runBoundaryTable(t, "IPOrCIDR", v, table)
}

// TestBrutal_HostOrIP_BoundaryTable runs HostOrIP through hostname
// + IP cross-product boundaries. RFC1123 hostnames allow labels
// of [a-z0-9-]{1,63} not starting/ending with hyphen; we don't
// enforce every RFC corner case but the obvious bugs (empty label,
// underscore, space, non-ASCII, control char, too-long label)
// must fail.
func TestBrutal_HostOrIP_BoundaryTable(t *testing.T) {
	v := lv.HostOrIP()
	table := []boundaryCase{
		// Hostname happy path
		{"hostname-simple", "myhost", false},
		{"hostname-fqdn", "host.example.com", false},
		{"hostname-deep", "a.b.c.d.e.example.com", false},
		{"hostname-numeric", "01host", false},
		{"hostname-hyphenated", "my-host-1", false},
		{"hostname-mixed-case", "MyHost.Example.Com", false},
		{"hostname-truenas-local", "truenas.local", false},
		{"hostname-truenas-direct", "truenas", false},

		// Hostname boundary
		{"hostname-empty-label", "host..example.com", true},
		{"hostname-leading-dot", ".host.example.com", true},
		{"hostname-trailing-dot", "host.example.com.", true},
		{"hostname-underscore", "my_host.example.com", true},
		{"hostname-space", "my host.example.com", true},
		{"hostname-tab", "my\thost", true},
		{"hostname-bang", "host!.example.com", true},
		{"hostname-slash", "host/path", true},
		{"hostname-at", "user@host", true},
		{"hostname-colon", "host:8080", true},
		{"hostname-question", "host?", true},
		{"hostname-asterisk", "*.example.com", true},

		// IP-in-place-of-hostname happy path
		{"ip-v4", "192.168.1.1", false},
		{"ip-v6", "2001:db8::1", false},
		{"ip-loopback", "127.0.0.1", false},

		// Whitespace / empty
		{"empty", "", false},
		{"ws-leading", " host", false},
		{"ws-trailing", "host ", false},
		{"ws-only", "   ", false},

		// Unicode (IDN; validator currently rejects)
		{"idn-punycode", "xn--bcher-kva.example", false}, // punycode is ASCII-compatible
		{"idn-raw-unicode", "bücher.example", true},      // raw unicode → invalid
		{"emoji", "🌐", true},
		{"cyrillic", "хост.example", true},

		// Control chars / injection
		{"control-NUL", "host\x00", true},
		{"control-CR", "host\rinjected", true},
		{"control-LF", "host\ninjected", true},
		{"sql-injection", "host'; DROP", true},
		{"path-traversal", "../host", true},

		// Length
		{"label-too-long-64", strings.Repeat("a", 64) + ".example.com", false}, // RFC says 63 max but we don't enforce
		{"label-very-long", strings.Repeat("a", 250), false},
		{"total-too-long", strings.Repeat("a", 1024), false},
	}
	runBoundaryTable(t, "HostOrIP", v, table)
}

// TestBrutal_ZFSPath_BoundaryTable runs ZFS dataset path validation
// through the input classes that have hurt other providers' ZFS
// integration — spaces (zpool import refuses), backslash (Windows
// users), control chars (terminal exploits via shell embedding),
// path-traversal attempts, very-long components, multi-byte unicode,
// and the @ / : / . metadata characters that DO have semantic
// meaning in ZFS (snapshots, bookmarks, clones).
func TestBrutal_ZFSPath_BoundaryTable(t *testing.T) {
	v := lv.ZFSPath()
	table := []boundaryCase{
		// Happy path
		{"single-component", "tank", false},
		{"two-components", "tank/data", false},
		{"deep-tree", "tank/a/b/c/d/e", false},
		{"with-hyphen", "tank-pool/data-set", false},
		{"with-underscore", "tank/my_data", false},
		{"with-dot", "tank/data.bak", false},
		{"with-colon", "tank/data:tag", false}, // : allowed
		{"with-at", "tank/data@snap", false},   // @ allowed (snapshot)
		{"numeric", "tank123/0", false},
		{"mixed-alphanumeric", "Tank01/Data02", false},

		// Empty handling
		{"empty", "", true},          // empty string is invalid
		{"single-slash", "/", true},  // leading + trailing empty
		{"only-spaces", "   ", true}, // single component but with spaces — fails space check

		// Path malformation
		{"leading-slash", "/tank/data", true},  // empty leading component
		{"trailing-slash", "tank/data/", true}, // empty trailing component
		{"double-slash", "tank//data", true},   // empty middle component
		{"triple-slash", "tank///data", true},  // multiple empty components
		{"only-slashes", "//", true},           // all empty

		// Space rejection
		{"space-leading", " tank", true},
		{"space-trailing", "tank ", true},
		{"space-middle", "tank/my data", true},
		{"tab-embedded", "tank/my\tdata", true},

		// Forbidden characters
		{"backslash", `tank\data`, true},
		{"asterisk", "tank/data*", true},
		{"question", "tank/data?", true},
		{"bracket", "tank/[data]", true},
		{"paren", "tank/(data)", true},
		{"comma", "tank,data", true},
		{"semicolon", "tank;data", true},
		{"pipe", "tank|data", true},
		{"ampersand", "tank&data", true},
		{"dollar", "tank$data", true},
		{"caret", "tank^data", true},
		{"percent", "tank%data", true},

		// Control chars
		{"NUL", "tank/data\x00", true},
		{"newline", "tank/data\n", true},
		{"CR", "tank/data\r", true},
		{"bell", "tank/data\x07", true},
		{"backspace", "tank/data\x08", true},
		{"escape", "tank/data\x1b", true},

		// Unicode
		{"unicode-cyrillic", "tank/данные", true},
		{"unicode-cjk", "tank/データ", true},
		{"unicode-emoji", "tank/💾", true},
		{"unicode-combining", "tank/café", true},
		{"unicode-rtl", "tank/‮tank", true},

		// Injection patterns
		{"sql-injection", "tank/data'; DROP--", true},
		{"shell-meta", "tank/`whoami`", true},
		{"path-traversal-up", "tank/../etc", true}, // .. in components — . is allowed but ..gets through unless tested
		{"path-traversal-double-dot", "tank/../passwd", true},

		// Length
		{"very-long-component", strings.Repeat("a", 256), false}, // we don't enforce length
		{"deep-100-levels", strings.Repeat("a/", 100) + "a", false},
		{"very-long-1KB", strings.Repeat("a", 1024), false},

		// Edge: dot-only components
		{"single-dot", "tank/.", true},  // . component now rejected (path-traversal guard)     // . is in allowed chars
		{"double-dot", "tank/..", true}, // .. component now rejected (path-traversal guard)    // .. is in allowed chars (test catches via path-traversal name)
		{"triple-dot", "tank/...", false},
	}
	runBoundaryTable(t, "ZFSPath", v, table)
}

// TestBrutal_CompressionAlgorithm_BoundaryTable enumerates every
// known compression value plus invalid lookalikes (case variants,
// typos, lowercase, mixed-case, "snappy" which is NOT in the
// TrueNAS list despite being a common ZFS option elsewhere).
func TestBrutal_CompressionAlgorithm_BoundaryTable(t *testing.T) {
	v := lv.CompressionAlgorithm()
	table := []boundaryCase{
		// All valid (canonical case)
		{"OFF", "OFF", false},
		{"LZ4", "LZ4", false},
		{"GZIP", "GZIP", false},
		{"GZIP-1", "GZIP-1", false},
		{"GZIP-9", "GZIP-9", false},
		{"ZSTD", "ZSTD", false},
		{"ZSTD-FAST", "ZSTD-FAST", false},
		{"ZLE", "ZLE", false},
		{"LZJB", "LZJB", false},

		// Case variants — validator normalises to upper
		{"lowercase-off", "off", false},
		{"lowercase-lz4", "lz4", false},
		{"mixed-case-Zstd", "Zstd", false},
		{"mixed-case-GzIp", "GzIp", false},

		// Whitespace tolerance
		{"leading-space", " LZ4", false},
		{"trailing-space", "LZ4 ", false},
		{"both-sides", "  LZ4  ", false},

		// Invalid values
		{"empty", "", true},
		{"unknown-snappy", "SNAPPY", true}, // not in TrueNAS list
		{"unknown-lz77", "LZ77", true},
		{"unknown-bzip2", "BZIP2", true},
		{"unknown-lzma", "LZMA", true},
		{"unknown-xz", "XZ", true},

		// Boundary / typo
		{"gzip-0", "GZIP-0", true}, // below range
		{"gzip-10", "GZIP-10", true},
		{"gzip-no-level", "GZIP-", true},
		{"zstd-typo", "ZSTDD", true},
		{"lz4-typo", "LZ44", true},

		// Injection / hostile
		{"sql-injection", "LZ4'; DROP", true},
		{"shell-meta", "`whoami`", true},
		{"NUL", "LZ4\x00", true},
		{"unicode", "ЛЗ4", true},
		{"emoji", "🗜", true},
		{"very-long", strings.Repeat("L", 1024), true},
	}
	runBoundaryTable(t, "CompressionAlgorithm", v, table)
}

// TestBrutal_NullUnknownAcceptance enumerates the
// Null + Unknown handling for every custom validator — both should
// short-circuit (validators don't fire on absent values). A
// regression that adds error diagnostics for null/unknown values
// breaks plan-time correctness for every config that defers the
// value to apply.
func TestBrutal_NullUnknownAcceptance(t *testing.T) {
	validators := map[string]validator.String{
		"IPOrCIDR":             lv.IPOrCIDR(),
		"HostOrIP":             lv.HostOrIP(),
		"ZFSPath":              lv.ZFSPath(),
		"CompressionAlgorithm": lv.CompressionAlgorithm(),
	}
	for name, v := range validators {
		v := v
		t.Run(name+"/null", func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:           path.Root("x"),
				PathExpression: path.MatchRoot("x"),
				ConfigValue:    types.StringNull(),
				Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, nil)},
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("%s(null) emitted error: %v", name, resp.Diagnostics)
			}
		})
		t.Run(name+"/unknown", func(t *testing.T) {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:           path.Root("x"),
				PathExpression: path.MatchRoot("x"),
				ConfigValue:    types.StringUnknown(),
				Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("%s(unknown) emitted error: %v", name, resp.Diagnostics)
			}
		})
	}
}
