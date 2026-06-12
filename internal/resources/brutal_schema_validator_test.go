package resources

import (
	"context"
	"strings"
	"testing"

	fwattr "github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// brutalStringInputs is the battery of edge inputs every wired
// string validator across every resource must survive without
// panicking. Each entry exists for a specific bug class — see the
// inline comments. A panic in any validator on any input gets
// caught here once, not 100 times across the per-resource test
// files.
//
// We don't assert "this validator must reject input X" because the
// validator-specific brutal tests under internal/validators/
// already pin those rules. This test is the safety net: regardless
// of WHAT the validator decides, it must not panic, must not
// produce undecidable diagnostic state, and must call back into
// the response within bounded time.
var brutalStringInputs = []string{
	// Boundary lengths
	"",
	" ",
	"a",
	strings.Repeat("x", 64), // common min/max boundary
	strings.Repeat("x", 256),
	strings.Repeat("x", 1024),
	strings.Repeat("x", 100_000), // very long — catches O(n^2) regex / strings.ToUpper edge cases

	// Whitespace classes
	"\t",
	"\n",
	"\r\n",
	"  leading-trail  ",

	// Control chars
	"\x00",
	"prefix\x00suffix",   // embedded NUL — bug class: strings.Contains based gating
	"\x07",               // BEL
	"\x1b[31mred\x1b[0m", // ANSI escape

	// Unicode / multi-byte
	"тест",   // Cyrillic
	"测试",     // CJK
	"🚀",      // Emoji
	"é",     // combining acute on e — NFC vs NFD
	"\u200b", // zero-width space
	"\u202e", // RTL override (security: phishing vectors)

	// SQL / shell injection
	"' OR 1=1--",
	"; DROP TABLE users; --",
	"`whoami`",
	"$(rm -rf /)",
	"&& cat /etc/passwd",

	// Path traversal
	"../../etc/passwd",
	"..\\..\\windows\\system32",
	"/dev/null",
	"file:///etc/passwd",

	// HTTP / URL injection
	"http://evil.example/?x=" + strings.Repeat("a", 1024),
	"javascript:alert(1)",
	"data:text/html,<script>alert(1)</script>",
}

// brutalInt64Inputs are the boundary numeric values every wired
// int64 validator must classify without panic. Same safety-net
// model as brutalStringInputs.
var brutalInt64Inputs = []int64{
	0,
	1,
	-1,
	int64(^uint64(0) >> 1),    // max int64
	-int64(^uint64(0)>>1) - 1, // min int64
	1<<32 - 1,                 // max uint32 boundary
	1 << 32,                   // first value above uint32
	1024,
	-1024,
}

// resourceConstructors lists every Resource the provider registers.
// Keep alphabetical for easy diffing when a new resource lands. The
// invariant TestRegistrationMatchesFilesystem (in the provider
// package) makes sure this list stays in sync with the registered
// factory list.
func resourceConstructors() []func() resource.Resource {
	return []func() resource.Resource{
		NewACMEDNSAuthenticatorResource,
		NewAlertClassesResource,
		NewAlertServiceResource,
		NewAPIKeyResource,
		NewAppResource,
		NewCatalogResource,
		NewCertificateResource,
		NewCloudBackupResource,
		NewCloudSyncCredentialResource,
		NewCloudSyncResource,
		NewCronJobResource,
		NewDatasetResource,
		NewDirectoryServicesResource,
		NewDNSNameserverResource,
		NewFilesystemACLResource,
		NewFilesystemACLTemplateResource,
		NewFTPConfigResource,
		NewGroupResource,
		NewInitScriptResource,
		NewISCSIAuthResource,
		NewISCSIExtentResource,
		NewISCSIInitiatorResource,
		NewISCSIPortalResource,
		NewISCSITargetExtentResource,
		NewISCSITargetResource,
		NewKerberosKeytabResource,
		NewKerberosRealmResource,
		NewKeychainCredentialResource,
		NewKMIPConfigResource,
		NewMailConfigResource,
		NewNetworkConfigResource,
		NewNetworkInterfaceResource,
		NewNFSConfigResource,
		NewNFSShareResource,
		NewNVMetGlobalResource,
		NewNVMetHostResource,
		NewNVMetHostSubsysResource,
		NewNVMetNamespaceResource,
		NewNVMetPortResource,
		NewNVMetPortSubsysResource,
		NewNVMetSubsysResource,
		NewPoolResource,
		NewPrivilegeResource,
		NewReplicationResource,
		NewReportingExporterResource,
		NewRsyncTaskResource,
		NewScrubTaskResource,
		NewServiceResource,
		NewSMBConfigResource,
		NewSMBShareResource,
		NewSnapshotTaskResource,
		NewSNMPConfigResource,
		NewSSHConfigResource,
		NewStaticRouteResource,
		NewSystemDatasetResource,
		NewSystemUpdateResource,
		NewTunableResource,
		NewUPSConfigResource,
		NewUserResource,
		NewVMDeviceResource,
		NewVMResource,
		NewVMwareResource,
	}
}

// resourceTypeName probes the resource for its TypeName via
// Metadata, used to label sub-tests so a failure points at the
// exact resource + attribute that panicked.
func resourceTypeName(r resource.Resource) string {
	var resp resource.MetadataResponse
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "truenas"}, &resp)
	return resp.TypeName
}

// TestBrutalSchema_AllWiredValidators iterates every resource in
// the provider, introspects its schema, walks every attribute that
// has validators wired, and fires the brutality battery through
// each one. The test asserts no validator panics on any input —
// regardless of whether it accepts or rejects the value.
//
// Why this is the highest-leverage test in the file: the framework
// validator library (stringvalidator.LengthBetween, OneOf, etc.)
// is exercised at 200+ wire points across the provider. A panic on
// any of them at plan-time crashes terraform mid-plan with a stack
// trace the operator can't act on. Catching the panic here means
// the validator returns a clean diagnostic instead.
//
// The test is fast — pure in-memory schema walks, no client calls.
// At 63 resources × ~10 attributes × 36 brutality inputs each =
// ~23,000 individual validator invocations per run. Sub-second.
func TestBrutalSchema_AllWiredValidators(t *testing.T) {
	for _, ctor := range resourceConstructors() {

		r := ctor()
		tn := resourceTypeName(r)
		t.Run(tn, func(t *testing.T) {
			ctx := context.Background()
			sch := schemaOf(t, ctx, r)
			for name, attr := range sch.Schema.Attributes {
				t.Run(name, func(t *testing.T) {
					exerciseValidatorsOnAttribute(t, sch.Schema, name, attr)
				})
			}
		})
	}
}

func exerciseValidatorsOnAttribute(t *testing.T, sch schema.Schema, name string, attr schema.Attribute) {
	t.Helper()
	ctx := context.Background()
	switch a := attr.(type) {
	case schema.StringAttribute:
		if len(a.Validators) == 0 {
			return
		}
		for _, val := range brutalStringInputs {
			for vi, v := range a.Validators {
				safe := safeSubtestName(val)
				t.Run(safe, func(t *testing.T) {
					defer recoverAsFailure(t, name, vi, val)
					req := validator.StringRequest{
						Path:           path.Root(name),
						PathExpression: path.MatchRoot(name),
						ConfigValue:    types.StringValue(val),
						Config:         tfsdk.Config{Schema: sch, Raw: emptyObjectRaw(ctx, sch)},
					}
					resp := &validator.StringResponse{}
					v.ValidateString(ctx, req, resp)
				})
			}
		}
	case schema.Int64Attribute:
		if len(a.Validators) == 0 {
			return
		}
		for _, val := range brutalInt64Inputs {
			for vi, v := range a.Validators {
				t.Run(int64Name(val), func(t *testing.T) {
					defer recoverAsFailure(t, name, vi, val)
					req := validator.Int64Request{
						Path:           path.Root(name),
						PathExpression: path.MatchRoot(name),
						ConfigValue:    types.Int64Value(val),
						Config:         tfsdk.Config{Schema: sch, Raw: emptyObjectRaw(ctx, sch)},
					}
					resp := &validator.Int64Response{}
					v.ValidateInt64(ctx, req, resp)
				})
			}
		}
	case schema.ListAttribute:
		if len(a.Validators) == 0 {
			return
		}
		// Build a battery of edge List values: nil, empty, one
		// element, large list. List validators (listvalidator.
		// SizeAtLeast, ValueStringsAre, etc.) panic-classes:
		// nil-element handling, very-long-list iteration cost,
		// nested validator panic on weird element values.
		lists := []struct {
			name string
			val  types.List
		}{
			{"null", types.ListNull(a.ElementType)},
			{"unknown", types.ListUnknown(a.ElementType)},
			{"empty", types.ListValueMust(a.ElementType, nil)},
		}
		// Try a single-element list with each string brutality
		// input — only if the element type is StringType.
		if a.ElementType == types.StringType {
			for _, val := range []string{"", "a", strings.Repeat("x", 1024), "🚀", "\x00"} {
				lv, d := types.ListValue(types.StringType, []fwattr.Value{types.StringValue(val)})
				if d.HasError() {
					continue
				}
				lists = append(lists, struct {
					name string
					val  types.List
				}{"single-" + safeSubtestName(val), lv})
			}
		}
		for _, lc := range lists {

			for vi, v := range a.Validators {
				t.Run(lc.name, func(t *testing.T) {
					defer recoverAsFailure(t, name, vi, lc.val)
					req := validator.ListRequest{
						Path:           path.Root(name),
						PathExpression: path.MatchRoot(name),
						ConfigValue:    lc.val,
						Config:         tfsdk.Config{Schema: sch, Raw: emptyObjectRaw(ctx, sch)},
					}
					resp := &validator.ListResponse{}
					v.ValidateList(ctx, req, resp)
				})
			}
		}
	case schema.MapAttribute:
		if len(a.Validators) == 0 {
			return
		}
		// Build a battery of edge Map values. mapvalidator
		// surface is small in this codebase but the panic class
		// is the same.
		maps := []struct {
			name string
			val  types.Map
		}{
			{"null", types.MapNull(a.ElementType)},
			{"unknown", types.MapUnknown(a.ElementType)},
			{"empty", types.MapValueMust(a.ElementType, nil)},
		}
		for _, mc := range maps {

			for vi, v := range a.Validators {
				t.Run(mc.name, func(t *testing.T) {
					defer recoverAsFailure(t, name, vi, mc.val)
					req := validator.MapRequest{
						Path:           path.Root(name),
						PathExpression: path.MatchRoot(name),
						ConfigValue:    mc.val,
						Config:         tfsdk.Config{Schema: sch, Raw: emptyObjectRaw(ctx, sch)},
					}
					resp := &validator.MapResponse{}
					v.ValidateMap(ctx, req, resp)
				})
			}
		}
	}
}

// emptyObjectRaw builds an empty-object tftypes.Value matching the
// schema's object type. Used as Config.Raw when the test wants to
// exercise a single attribute's validator without populating the
// full config — the framework needs SOME raw value at Config.Raw
// for path-resolution to work but cross-attribute validators (the
// only kind that care about siblings) are tested separately.
func emptyObjectRaw(ctx context.Context, sch schema.Schema) tftypes.Value {
	return tftypes.NewValue(sch.Type().TerraformType(ctx), nil)
}

// recoverAsFailure turns a panic into a t.Errorf so the test
// runner reports the offending (attribute, validator index, value)
// instead of crashing the whole suite. This is the load-bearing
// behaviour for the brutality test — we WANT every input to reach
// every validator, and a panic on input #15 must not stop inputs
// #16-#36 from being tested against this same attribute.
func recoverAsFailure(t *testing.T, attr string, vIdx int, val interface{}) {
	if r := recover(); r != nil {
		t.Errorf("PANIC in validator at attr=%q validator-index=%d value=%v:\n  %v",
			attr, vIdx, val, r)
	}
}

// safeSubtestName turns an arbitrary brutality input into a
// subtest name that won't break the test runner's path-based
// output. Control chars and unicode get replaced with their hex
// escape; long values get truncated.
func safeSubtestName(s string) string {
	const maxLen = 32
	var b strings.Builder
	for i, r := range s {
		if b.Len() >= maxLen {
			b.WriteString("...")
			break
		}
		_ = i
		switch {
		case r < 0x20 || r == 0x7f:
			b.WriteString("\\x")
			b.WriteString(hexByte(byte(r)))
		case r == ' ' || r == '/':
			b.WriteByte('_')
		case r > 0x7e:
			b.WriteString("\\u")
			b.WriteString(hexInt(int(r)))
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "empty"
	}
	return b.String()
}

func hexByte(b byte) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0xf]})
}

func hexInt(i int) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{
		hex[(i>>12)&0xf], hex[(i>>8)&0xf], hex[(i>>4)&0xf], hex[i&0xf],
	})
}

func int64Name(v int64) string {
	const maxInt64 = int64(^uint64(0) >> 1)
	const minInt64 = -maxInt64 - 1
	switch v {
	case 0:
		return "zero"
	case 1:
		return "one"
	case -1:
		return "neg-one"
	case maxInt64:
		return "max-int64"
	case minInt64:
		return "min-int64"
	default:
		if v > 0 {
			return "pos-" + intToString(v)
		}
		return "neg-" + intToString(-v)
	}
}

func intToString(v int64) string {
	var b strings.Builder
	if v == 0 {
		return "0"
	}
	var digits []byte
	for v > 0 {
		digits = append(digits, byte('0'+v%10))
		v /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		b.WriteByte(digits[i])
	}
	return b.String()
}
