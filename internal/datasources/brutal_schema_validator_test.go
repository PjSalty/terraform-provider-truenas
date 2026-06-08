package datasources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Mirror of internal/resources/brutal_schema_validator_test.go for
// the datasource side. Datasources have a smaller attribute surface
// per type (read-only lookup keys, no Required Create-side fields)
// but the same wired-validator panic class still applies — a
// stringvalidator.LengthBetween that panics on input X stops
// terraform mid-plan regardless of whether it sits behind a
// resource or a datasource.
//
// Kept package-local rather than imported from the resources test
// because the framework types diverge (datasource.Schema vs
// resource.Schema have different attribute interfaces).

var brutalStringInputsDS = []string{
	"",
	" ",
	"a",
	strings.Repeat("x", 64),
	strings.Repeat("x", 256),
	strings.Repeat("x", 1024),
	strings.Repeat("x", 100_000),
	"\t",
	"\n",
	"\r\n",
	"  leading-trail  ",
	"\x00",
	"prefix\x00suffix",
	"\x07",
	"\x1b[31mred\x1b[0m",
	"тест",
	"测试",
	"🚀",
	"é",
	"​",
	"‮",
	"' OR 1=1--",
	"; DROP TABLE users; --",
	"`whoami`",
	"$(rm -rf /)",
	"&& cat /etc/passwd",
	"../../etc/passwd",
	"..\\..\\windows\\system32",
	"/dev/null",
	"file:///etc/passwd",
	"http://evil.example/?x=" + strings.Repeat("a", 1024),
	"javascript:alert(1)",
	"data:text/html,<script>alert(1)</script>",
}

var brutalInt64InputsDS = []int64{
	0, 1, -1,
	int64(^uint64(0) >> 1),
	-int64(^uint64(0)>>1) - 1,
	1<<32 - 1,
	1 << 32,
	1024,
	-1024,
}

func datasourceConstructors() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewAlertServiceDataSource,
		NewAPIKeyDataSource,
		NewAppDataSource,
		NewAppsDataSource,
		NewCatalogDataSource,
		NewCertificateDataSource,
		NewCloudSyncCredentialDataSource,
		NewCronJobDataSource,
		NewDatasetDataSource,
		NewDatasetsDataSource,
		NewDirectoryServicesDataSource,
		NewDiskDataSource,
		NewGroupDataSource,
		NewISCSIExtentDataSource,
		NewISCSIInitiatorDataSource,
		NewISCSIPortalDataSource,
		NewISCSITargetDataSource,
		NewKerberosRealmDataSource,
		NewKeychainCredentialDataSource,
		NewNetworkConfigDataSource,
		NewNetworkInterfaceDataSource,
		NewPrivilegeDataSource,
		NewServiceDataSource,
		NewShareNFSDataSource,
		NewShareSMBDataSource,
		NewSnapshotTaskDataSource,
		NewSystemInfoDataSource,
		NewUserDataSource,
		NewVMDataSource,
		NewVMsDataSource,
	}
}

func datasourceTypeName(ds datasource.DataSource) string {
	var resp datasource.MetadataResponse
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "truenas"}, &resp)
	return resp.TypeName
}

// TestBrutalSchema_DataSourceValidators is the datasource mirror of
// TestBrutalSchema_AllWiredValidators. Per-input recover ensures a
// panic on one input doesn't suppress the rest of the brutality
// surface for the same attribute.
func TestBrutalSchema_DataSourceValidators(t *testing.T) {
	for _, ctor := range datasourceConstructors() {
		ctor := ctor
		ds := ctor()
		tn := datasourceTypeName(ds)
		t.Run(tn, func(t *testing.T) {
			ctx := context.Background()
			resp := datasource.SchemaResponse{}
			ds.Schema(ctx, datasource.SchemaRequest{}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Schema returned error: %v", resp.Diagnostics)
			}
			for name, attr := range resp.Schema.Attributes {
				name, attr := name, attr
				t.Run(name, func(t *testing.T) {
					exerciseDSValidatorsOnAttribute(t, resp.Schema, name, attr)
				})
			}
		})
	}
}

func exerciseDSValidatorsOnAttribute(t *testing.T, sch schema.Schema, name string, attr schema.Attribute) {
	t.Helper()
	ctx := context.Background()
	switch a := attr.(type) {
	case schema.StringAttribute:
		if len(a.Validators) == 0 {
			return
		}
		for _, val := range brutalStringInputsDS {
			for vi, v := range a.Validators {
				vi, v, val := vi, v, val
				t.Run(dsSafeName(val), func(t *testing.T) {
					defer dsRecoverAsFailure(t, name, vi, val)
					req := validator.StringRequest{
						Path:           path.Root(name),
						PathExpression: path.MatchRoot(name),
						ConfigValue:    types.StringValue(val),
						Config:         tfsdk.Config{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(ctx), nil)},
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
		for _, val := range brutalInt64InputsDS {
			for vi, v := range a.Validators {
				vi, v, val := vi, v, val
				t.Run(dsInt64Name(val), func(t *testing.T) {
					defer dsRecoverAsFailure(t, name, vi, val)
					req := validator.Int64Request{
						Path:           path.Root(name),
						PathExpression: path.MatchRoot(name),
						ConfigValue:    types.Int64Value(val),
						Config:         tfsdk.Config{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(ctx), nil)},
					}
					resp := &validator.Int64Response{}
					v.ValidateInt64(ctx, req, resp)
				})
			}
		}
	}
}

func dsRecoverAsFailure(t *testing.T, attr string, vIdx int, val interface{}) {
	if r := recover(); r != nil {
		t.Errorf("PANIC in DS validator at attr=%q validator-index=%d value=%v:\n  %v",
			attr, vIdx, val, r)
	}
}

func dsSafeName(s string) string {
	const maxLen = 32
	var b strings.Builder
	for _, r := range s {
		if b.Len() >= maxLen {
			b.WriteString("...")
			break
		}
		switch {
		case r < 0x20 || r == 0x7f:
			b.WriteString("\\x")
			b.WriteString(dsHexByte(byte(r)))
		case r == ' ' || r == '/':
			b.WriteByte('_')
		case r > 0x7e:
			b.WriteString("\\u")
			b.WriteString(dsHexInt(int(r)))
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "empty"
	}
	return b.String()
}

func dsHexByte(b byte) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[b>>4], hex[b&0xf]})
}

func dsHexInt(i int) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{
		hex[(i>>12)&0xf], hex[(i>>8)&0xf], hex[(i>>4)&0xf], hex[i&0xf],
	})
}

func dsInt64Name(v int64) string {
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
	}
	var b strings.Builder
	if v < 0 {
		b.WriteByte('-')
		v = -v
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
