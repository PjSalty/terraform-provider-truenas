package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Test helpers shared across _unit_test.go files.

// testCtx returns a background context for test use.
func testCtx() context.Context {
	return context.Background()
}

// diagDiagnostics wraps a Diagnostics pointer-receiver type so tests can pass
// an addressable diagnostics value inline.
type diagDiagnostics struct {
	diag.Diagnostics
}

// stringListValue builds a types.List of strings for test setup.
func stringListValue(t *testing.T, vals []string) types.List {
	t.Helper()
	elems := make([]attr.Value, len(vals))
	for i, v := range vals {
		elems[i] = types.StringValue(v)
	}
	l, d := types.ListValue(types.StringType, elems)
	if d.HasError() {
		t.Fatalf("stringListValue: %v", d)
	}
	return l
}

// --- dataset buildFullName ---

func TestBuildFullName(t *testing.T) {
	cases := []struct {
		name, pool, parent, dsname, want string
	}{
		{name: "no parent", pool: "tank", dsname: "data", want: "tank/data"},
		{name: "with parent", pool: "tank", parent: "apps", dsname: "postgres", want: "tank/apps/postgres"},
		{name: "nested parent", pool: "tank", parent: "a/b/c", dsname: "leaf", want: "tank/a/b/c/leaf"},
		{name: "empty parent explicit", pool: "tank", parent: "", dsname: "x", want: "tank/x"},
		{name: "pool only", pool: "pool1", dsname: "only", want: "pool1/only"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildFullName(tc.pool, tc.parent, tc.dsname)
			if got != tc.want {
				t.Errorf("buildFullName(%q,%q,%q) = %q, want %q", tc.pool, tc.parent, tc.dsname, got, tc.want)
			}
		})
	}
}

// --- app decodeValues and boolOrDefault ---

func TestDecodeValues(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantLen int
		wantErr bool
	}{
		{name: "empty", in: "", wantLen: 0},
		{name: "empty object", in: "{}", wantLen: 0},
		{name: "single key", in: `{"a":1}`, wantLen: 1},
		{name: "multi key", in: `{"a":1,"b":"x","c":true}`, wantLen: 3},
		{name: "nested", in: `{"outer":{"inner":"x"}}`, wantLen: 1},
		{name: "null becomes empty", in: `null`, wantLen: 0},
		{name: "invalid", in: `{bad`, wantErr: true},
		{name: "not object", in: `[1,2,3]`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := decodeValues(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("decodeValues(%q) expected error, got %v", tc.in, m)
				}
				return
			}
			if err != nil {
				t.Errorf("decodeValues(%q) unexpected error: %v", tc.in, err)
				return
			}
			if len(m) != tc.wantLen {
				t.Errorf("decodeValues(%q) len = %d, want %d", tc.in, len(m), tc.wantLen)
			}
		})
	}
}

func TestBoolOrDefault(t *testing.T) {
	cases := []struct {
		name string
		v    types.Bool
		def  bool
		want bool
	}{
		{name: "null default true", v: types.BoolNull(), def: true, want: true},
		{name: "null default false", v: types.BoolNull(), def: false, want: false},
		{name: "unknown default true", v: types.BoolUnknown(), def: true, want: true},
		{name: "true value", v: types.BoolValue(true), def: false, want: true},
		{name: "false value", v: types.BoolValue(false), def: true, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := boolOrDefault(tc.v, tc.def); got != tc.want {
				t.Errorf("boolOrDefault = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- cloudsync_credential buildProviderMap ---

func TestBuildProviderMap(t *testing.T) {
	cases := []struct {
		name         string
		providerType string
		attrs        string
		wantErr      bool
		wantType     string
	}{
		{name: "empty attributes", providerType: "S3", attrs: "", wantType: "S3"},
		{name: "with attributes", providerType: "B2", attrs: `{"account":"abc","key":"xyz"}`, wantType: "B2"},
		{name: "empty object", providerType: "AZURE_BLOB", attrs: "{}", wantType: "AZURE_BLOB"},
		{name: "attrs override type key", providerType: "S3", attrs: `{"type":"OVERRIDDEN","bucket":"b"}`, wantType: "S3"},
		{name: "invalid JSON", providerType: "S3", attrs: `{bad`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := buildProviderMap(tc.providerType, tc.attrs)
			if tc.wantErr {
				if err == nil {
					t.Errorf("buildProviderMap expected error")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tp, ok := m["type"].(string); !ok || tp != tc.wantType {
				t.Errorf("type = %v, want %q", m["type"], tc.wantType)
			}
		})
	}
}

// --- vm nullableString ---

func TestNullableString(t *testing.T) {
	if got := nullableString(nil); !got.IsNull() {
		t.Error("nullableString(nil) should be null")
	}
	s := "hello"
	if got := nullableString(&s); got.IsNull() || got.ValueString() != "hello" {
		t.Errorf("nullableString(&hello) = %v", got)
	}
	empty := ""
	if got := nullableString(&empty); got.IsNull() || got.ValueString() != "" {
		t.Error("nullableString(&empty) should be empty value, not null")
	}
}

// --- vm buildVMCreateRequest / buildVMUpdateRequest ---

func TestBuildVMCreateRequest(t *testing.T) {
	t.Run("minimal required fields only", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:   types.StringValue("vm1"),
			Memory: types.Int64Value(2048),
		}
		req := buildVMCreateRequest(plan)
		if req.Name != "vm1" {
			t.Errorf("Name = %q", req.Name)
		}
		if req.Memory != 2048 {
			t.Errorf("Memory = %d", req.Memory)
		}
		if req.Vcpus != nil {
			t.Errorf("Vcpus should be nil, got %v", *req.Vcpus)
		}
		if req.Bootloader != nil {
			t.Errorf("Bootloader should be nil, got %v", *req.Bootloader)
		}
	})
	t.Run("full fields", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:                  types.StringValue("full"),
			Description:           types.StringValue("test"),
			Memory:                types.Int64Value(4096),
			MinMemory:             types.Int64Value(2048),
			Vcpus:                 types.Int64Value(4),
			Cores:                 types.Int64Value(2),
			Threads:               types.Int64Value(2),
			Bootloader:            types.StringValue("UEFI"),
			Autostart:             types.BoolValue(true),
			HideFromMsr:           types.BoolValue(false),
			EnsureDisplayDevice:   types.BoolValue(true),
			Time:                  types.StringValue("UTC"),
			ShutdownTimeout:       types.Int64Value(90),
			ArchType:              types.StringValue("x86_64"),
			MachineType:           types.StringValue("q35"),
			CPUMode:               types.StringValue("HOST-PASSTHROUGH"),
			EnableSecureBoot:      types.BoolValue(true),
			TrustedPlatformModule: types.BoolValue(true),
		}
		req := buildVMCreateRequest(plan)
		if req.Vcpus == nil || *req.Vcpus != 4 {
			t.Errorf("Vcpus mismatch")
		}
		if req.Cores == nil || *req.Cores != 2 {
			t.Errorf("Cores mismatch")
		}
		if req.MinMemory == nil || *req.MinMemory != 2048 {
			t.Errorf("MinMemory mismatch")
		}
		if req.Bootloader == nil || *req.Bootloader != "UEFI" {
			t.Errorf("Bootloader mismatch")
		}
		if req.Autostart == nil || !*req.Autostart {
			t.Errorf("Autostart mismatch")
		}
		if req.EnableSecureBoot == nil || !*req.EnableSecureBoot {
			t.Errorf("EnableSecureBoot mismatch")
		}
		if req.TrustedPlatformModule == nil || !*req.TrustedPlatformModule {
			t.Errorf("TPM mismatch")
		}
	})
	t.Run("null/unknown values skipped", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:        types.StringValue("x"),
			Memory:      types.Int64Value(1024),
			Description: types.StringNull(),
			Vcpus:       types.Int64Unknown(),
			Bootloader:  types.StringNull(),
			Autostart:   types.BoolUnknown(),
		}
		req := buildVMCreateRequest(plan)
		if req.Description != nil {
			t.Errorf("Description should be nil")
		}
		if req.Vcpus != nil {
			t.Errorf("Vcpus should be nil")
		}
		if req.Bootloader != nil {
			t.Errorf("Bootloader should be nil")
		}
		if req.Autostart != nil {
			t.Errorf("Autostart should be nil")
		}
	})
}

func TestBuildVMUpdateRequest(t *testing.T) {
	t.Run("update always sets name and memory", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:   types.StringValue("vmU"),
			Memory: types.Int64Value(8192),
		}
		req := buildVMUpdateRequest(plan)
		if req.Name == nil || *req.Name != "vmU" {
			t.Errorf("Name mismatch: %v", req.Name)
		}
		if req.Memory == nil || *req.Memory != 8192 {
			t.Errorf("Memory mismatch: %v", req.Memory)
		}
	})
	t.Run("update with optional fields", func(t *testing.T) {
		plan := &VMResourceModel{
			Name:            types.StringValue("big"),
			Memory:          types.Int64Value(16384),
			Vcpus:           types.Int64Value(8),
			Cores:           types.Int64Value(4),
			Threads:         types.Int64Value(2),
			Description:     types.StringValue("upd"),
			Autostart:       types.BoolValue(false),
			ShutdownTimeout: types.Int64Value(60),
			CPUMode:         types.StringValue("CUSTOM"),
			CPUModel:        types.StringValue("EPYC"),
		}
		req := buildVMUpdateRequest(plan)
		if req.Vcpus == nil || *req.Vcpus != 8 {
			t.Errorf("Vcpus mismatch")
		}
		if req.CPUMode == nil || *req.CPUMode != "CUSTOM" {
			t.Errorf("CPUMode mismatch")
		}
		if req.CPUModel == nil || *req.CPUModel != "EPYC" {
			t.Errorf("CPUModel mismatch")
		}
		if req.ShutdownTimeout == nil || *req.ShutdownTimeout != 60 {
			t.Errorf("ShutdownTimeout mismatch")
		}
	})
}

// --- systemdataset buildSystemDatasetUpdate ---

func TestBuildSystemDatasetUpdate(t *testing.T) {
	t.Run("set to tank", func(t *testing.T) {
		plan := &SystemDatasetResourceModel{Pool: types.StringValue("tank")}
		req := buildSystemDatasetUpdate(plan)
		if req.Pool == nil || *req.Pool != "tank" {
			t.Errorf("Pool = %v", req.Pool)
		}
	})
	t.Run("empty string resets to nil", func(t *testing.T) {
		plan := &SystemDatasetResourceModel{Pool: types.StringValue("")}
		req := buildSystemDatasetUpdate(plan)
		if req.Pool != nil {
			t.Errorf("Pool should be nil for empty string")
		}
	})
	t.Run("null pool omitted", func(t *testing.T) {
		plan := &SystemDatasetResourceModel{Pool: types.StringNull()}
		req := buildSystemDatasetUpdate(plan)
		if req.Pool != nil {
			t.Errorf("Pool should be nil for null")
		}
	})
	t.Run("unknown pool omitted", func(t *testing.T) {
		plan := &SystemDatasetResourceModel{Pool: types.StringUnknown()}
		req := buildSystemDatasetUpdate(plan)
		if req.Pool != nil {
			t.Errorf("Pool should be nil for unknown")
		}
	})
}

// --- vm_device vmDeviceAttrsToAPI ---

func TestVMDeviceAttrsToAPI(t *testing.T) {
	ctx := context.Background()
	t.Run("null map returns dtype only", func(t *testing.T) {
		m := types.MapNull(types.StringType)
		attrs := vmDeviceAttrsToAPI(ctx, "DISK", m)
		if attrs["dtype"] != "DISK" {
			t.Errorf("dtype mismatch")
		}
		if len(attrs) != 1 {
			t.Errorf("expected only dtype, got %v", attrs)
		}
	})
	t.Run("unknown map returns dtype only", func(t *testing.T) {
		m := types.MapUnknown(types.StringType)
		attrs := vmDeviceAttrsToAPI(ctx, "NIC", m)
		if attrs["dtype"] != "NIC" {
			t.Errorf("dtype mismatch")
		}
	})
	t.Run("boolean and int coercion", func(t *testing.T) {
		elems := map[string]attr.Value{
			"path":       types.StringValue("/dev/zvol/tank/v1"),
			"readonly":   types.StringValue("true"),
			"sectorsize": types.StringValue("512"),
			"pin":        types.StringValue("false"),
		}
		m, diags := types.MapValue(types.StringType, elems)
		if diags.HasError() {
			t.Fatalf("map value errors: %v", diags)
		}
		attrs := vmDeviceAttrsToAPI(ctx, "DISK", m)
		if attrs["path"] != "/dev/zvol/tank/v1" {
			t.Errorf("path mismatch: %v", attrs["path"])
		}
		if b, ok := attrs["readonly"].(bool); !ok || !b {
			t.Errorf("readonly should be bool true, got %T %v", attrs["readonly"], attrs["readonly"])
		}
		if b, ok := attrs["pin"].(bool); !ok || b {
			t.Errorf("pin should be bool false, got %T %v", attrs["pin"], attrs["pin"])
		}
		if n, ok := attrs["sectorsize"].(int64); !ok || n != 512 {
			t.Errorf("sectorsize should be int64 512, got %T %v", attrs["sectorsize"], attrs["sectorsize"])
		}
		if attrs["dtype"] != "DISK" {
			t.Errorf("dtype mismatch")
		}
	})
	t.Run("dtype key in map is ignored", func(t *testing.T) {
		elems := map[string]attr.Value{
			"dtype": types.StringValue("BADDTYPE"),
			"x":     types.StringValue("y"),
		}
		m, _ := types.MapValue(types.StringType, elems)
		attrs := vmDeviceAttrsToAPI(ctx, "CDROM", m)
		// dtype from args must win over map value
		if attrs["dtype"] != "CDROM" {
			t.Errorf("dtype = %v, want CDROM", attrs["dtype"])
		}
	})
}

// --- catalog listToStringSlice ---

func TestListToStringSlice(t *testing.T) {
	ctx := context.Background()
	t.Run("null list", func(t *testing.T) {
		l := types.ListNull(types.StringType)
		out, d := listToStringSlice(ctx, l)
		if d.HasError() {
			t.Fatalf("errors: %v", d)
		}
		if len(out) != 0 {
			t.Errorf("expected empty, got %v", out)
		}
	})
	t.Run("unknown list", func(t *testing.T) {
		l := types.ListUnknown(types.StringType)
		out, _ := listToStringSlice(ctx, l)
		if len(out) != 0 {
			t.Errorf("expected empty, got %v", out)
		}
	})
	t.Run("populated list", func(t *testing.T) {
		l, _ := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("a"), types.StringValue("b"), types.StringValue("c"),
		})
		out, _ := listToStringSlice(ctx, l)
		if len(out) != 3 || out[0] != "a" || out[2] != "c" {
			t.Errorf("unexpected output: %v", out)
		}
	})
}

// --- privilege list converters ---

func TestPrivilegeListToIntSlice(t *testing.T) {
	ctx := context.Background()
	t.Run("null returns empty", func(t *testing.T) {
		var d diag.Diagnostics
		l := types.ListNull(types.Int64Type)
		out := privilegeListToIntSlice(ctx, l, &d)
		if len(out) != 0 {
			t.Errorf("expected empty")
		}
	})
	t.Run("populated list", func(t *testing.T) {
		var d diag.Diagnostics
		l, _ := types.ListValue(types.Int64Type, []attr.Value{
			types.Int64Value(10), types.Int64Value(20), types.Int64Value(30),
		})
		out := privilegeListToIntSlice(ctx, l, &d)
		if len(out) != 3 || out[0] != 10 || out[2] != 30 {
			t.Errorf("unexpected: %v", out)
		}
	})
}

func TestPrivilegeListToStringSlice(t *testing.T) {
	ctx := context.Background()
	var d diag.Diagnostics
	t.Run("null", func(t *testing.T) {
		out := privilegeListToStringSlice(ctx, types.ListNull(types.StringType), &d)
		if len(out) != 0 {
			t.Errorf("expected empty, got %v", out)
		}
	})
	t.Run("populated", func(t *testing.T) {
		l, _ := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("FULL_ADMIN"), types.StringValue("READONLY_ADMIN"),
		})
		out := privilegeListToStringSlice(ctx, l, &d)
		if len(out) != 2 || out[0] != "FULL_ADMIN" {
			t.Errorf("unexpected: %v", out)
		}
	})
}

func TestPrivilegeListToDSGroupSlice(t *testing.T) {
	ctx := context.Background()
	var d diag.Diagnostics
	t.Run("null returns empty", func(t *testing.T) {
		out := privilegeListToDSGroupSlice(ctx, types.ListNull(types.StringType), &d)
		if len(out) != 0 {
			t.Errorf("expected empty")
		}
	})
	t.Run("mixed int and string", func(t *testing.T) {
		l, _ := types.ListValue(types.StringType, []attr.Value{
			types.StringValue("12345"),
			types.StringValue("S-1-5-21-abc"),
			types.StringValue("999"),
		})
		out := privilegeListToDSGroupSlice(ctx, l, &d)
		if len(out) != 3 {
			t.Fatalf("want 3, got %d", len(out))
		}
		if n, ok := out[0].(int); !ok || n != 12345 {
			t.Errorf("[0] should be int 12345, got %T %v", out[0], out[0])
		}
		if s, ok := out[1].(string); !ok || s != "S-1-5-21-abc" {
			t.Errorf("[1] should be string, got %T %v", out[1], out[1])
		}
		if n, ok := out[2].(int); !ok || n != 999 {
			t.Errorf("[2] should be int 999, got %T %v", out[2], out[2])
		}
	})
}

// --- network_interface aliasObjectType ---

func TestAliasObjectType(t *testing.T) {
	ot := aliasObjectType()
	if ot.AttrTypes["type"] != types.StringType {
		t.Error("type should be StringType")
	}
	if ot.AttrTypes["address"] != types.StringType {
		t.Error("address should be StringType")
	}
	if ot.AttrTypes["netmask"] != types.Int64Type {
		t.Error("netmask should be Int64Type")
	}
}
