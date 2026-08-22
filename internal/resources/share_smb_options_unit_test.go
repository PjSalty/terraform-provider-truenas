package resources

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
	"github.com/PjSalty/terraform-provider-truenas/internal/wsclient"
)

func smbOptsObject(t *testing.T, set map[string]attr.Value) types.Object {
	t.Helper()
	vals := map[string]attr.Value{
		"remote_path":           types.ListNull(types.StringType),
		"hostsallow":            types.ListNull(types.StringType),
		"hostsdeny":             types.ListNull(types.StringType),
		"aapl_name_mangling":    types.BoolNull(),
		"auto_snapshot":         types.BoolNull(),
		"auto_dataset_creation": types.BoolNull(),
		"timemachine_quota":     types.Int64Null(),
		"auto_quota":            types.Int64Null(),
		"grace_period":          types.Int64Null(),
		"dataset_naming_schema": types.StringNull(),
		"vuid":                  types.StringNull(),
	}
	for k, v := range set {
		vals[k] = v
	}
	o, diags := types.ObjectValue(smbOptionAttrTypes, vals)
	if diags.HasError() {
		t.Fatalf("building options object: %v", diags)
	}
	return o
}

func strListVal(t *testing.T, items ...string) types.List {
	t.Helper()
	elems := make([]attr.Value, 0, len(items))
	for _, item := range items {
		elems = append(elems, types.StringValue(item))
	}
	return types.ListValueMust(types.StringType, elems)
}

// Only attributes the configuration actually set may be sent. An omitted one
// must stay omitted so the purpose's own default applies, rather than being
// overwritten with a zero value chosen here.
func TestSMBOptionsFromModel_sendsOnlyWhatWasSet(t *testing.T) {
	ctx := context.Background()

	t.Run("null object sends nothing", func(t *testing.T) {
		got, err := smbOptionsFromModel(ctx, types.ObjectNull(smbOptionAttrTypes))
		if err != nil || got != nil {
			t.Errorf("got %+v, %v", got, err)
		}
	})

	t.Run("unset fields are omitted", func(t *testing.T) {
		o := smbOptsObject(t, map[string]attr.Value{
			"remote_path": strListVal(t, `1.2.3.4\SHARE`),
		})
		got, err := smbOptionsFromModel(ctx, o)
		if err != nil {
			t.Fatalf("smbOptionsFromModel: %v", err)
		}
		if got.RemotePath == nil || (*got.RemotePath)[0] != `1.2.3.4\SHARE` {
			t.Errorf("remote_path = %v", got.RemotePath)
		}
		for name, v := range map[string]interface{}{
			"hostsallow": got.HostsAllow, "hostsdeny": got.HostsDeny,
		} {
			if v != (*[]string)(nil) {
				t.Errorf("%s was sent for a config that did not set it", name)
			}
		}
		if got.AAPLNameMangling != nil || got.GracePeriod != nil || got.VUID != nil {
			t.Errorf("an unset scalar was sent: %+v", got)
		}
	})

	t.Run("every field round-trips when set", func(t *testing.T) {
		o := smbOptsObject(t, map[string]attr.Value{
			"hostsallow":            strListVal(t, "10.0.0.0/8"),
			"hostsdeny":             strListVal(t, "0.0.0.0/0"),
			"aapl_name_mangling":    types.BoolValue(true),
			"auto_snapshot":         types.BoolValue(true),
			"auto_dataset_creation": types.BoolValue(true),
			"timemachine_quota":     types.Int64Value(1024),
			"auto_quota":            types.Int64Value(10),
			"grace_period":          types.Int64Value(120),
			"dataset_naming_schema": types.StringValue("%D/%U"),
			"vuid":                  types.StringValue("uuid-1"),
		})
		got, err := smbOptionsFromModel(ctx, o)
		if err != nil {
			t.Fatalf("smbOptionsFromModel: %v", err)
		}
		if got.HostsAllow == nil || got.HostsDeny == nil ||
			got.AAPLNameMangling == nil || !*got.AAPLNameMangling ||
			got.AutoSnapshot == nil || got.AutoDatasetCreation == nil ||
			got.TimeMachineQuota == nil || *got.TimeMachineQuota != 1024 ||
			got.AutoQuota == nil || *got.AutoQuota != 10 ||
			got.GracePeriod == nil || *got.GracePeriod != 120 ||
			got.DatasetNamingSchema == nil || *got.DatasetNamingSchema != "%D/%U" ||
			got.VUID == nil || *got.VUID != "uuid-1" {
			t.Errorf("a set field did not reach the wire: %+v", got)
		}
	})

	// An empty list means "clear this". ElementsAs yields a nil slice, and a
	// non-nil pointer to a nil slice marshals as JSON null, which middleware
	// rejects as not-a-list. This asserts the encoded JSON, not the Go value.
	t.Run("an empty list encodes as [] not null", func(t *testing.T) {
		// An explicitly EMPTY list, not a null one: types.ListValueFrom
		// with a nil slice yields null, which is "unset" and would be
		// omitted for the right reason, proving nothing.
		o := smbOptsObject(t, map[string]attr.Value{
			"hostsallow": types.ListValueMust(types.StringType, []attr.Value{}),
		})
		got, err := smbOptionsFromModel(ctx, o)
		if err != nil {
			t.Fatalf("smbOptionsFromModel: %v", err)
		}
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if string(body["hostsallow"]) != "[]" {
			t.Errorf("hostsallow encoded as %s, want []", body["hostsallow"])
		}
	})
}

// The server echoes only the fields belonging to the share's purpose, so the
// rest must read back NULL. A zero would misreport "this purpose has no such
// setting" as "it is set to 0" or "off".
func TestSMBOptionsToObject_absentFieldsAreNull(t *testing.T) {
	if got := smbOptionsToObject(nil); !got.IsNull() {
		t.Errorf("nil options produced %v, want a null object", got)
	}

	quota := int64(0)
	mangling := false
	obj := smbOptionsToObject(&truenas.SMBShareOptions{
		HostsAllow:       &[]string{"10.0.0.0/8"},
		AAPLNameMangling: &mangling,
		AutoQuota:        &quota,
	})
	attrs := obj.Attributes()

	// Present-but-falsy must be a known value, not null.
	if v, _ := attrs["aapl_name_mangling"].(types.Bool); v.IsNull() || v.ValueBool() {
		t.Errorf("a present false read back as %v", attrs["aapl_name_mangling"])
	}
	if v, _ := attrs["auto_quota"].(types.Int64); v.IsNull() || v.ValueInt64() != 0 {
		t.Errorf("a present 0 read back as %v", attrs["auto_quota"])
	}
	// Absent must be null, so a purpose that has no such field says so.
	for _, name := range []string{"remote_path", "grace_period", "vuid", "dataset_naming_schema", "timemachine_quota"} {
		if !attrs[name].IsNull() {
			t.Errorf("%s was absent from the server but read back as %v", name, attrs[name])
		}
	}
}

func TestSMBShareResource_ValidateConfigOptions(t *testing.T) {
	ctx := context.Background()
	r := &SMBShareResource{}
	sch := schemaOf(t, ctx, r)

	optsType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"remote_path":           tftypes.List{ElementType: tftypes.String},
		"hostsallow":            tftypes.List{ElementType: tftypes.String},
		"hostsdeny":             tftypes.List{ElementType: tftypes.String},
		"aapl_name_mangling":    tftypes.Bool,
		"auto_snapshot":         tftypes.Bool,
		"auto_dataset_creation": tftypes.Bool,
		"timemachine_quota":     tftypes.Number,
		"auto_quota":            tftypes.Number,
		"grace_period":          tftypes.Number,
		"dataset_naming_schema": tftypes.String,
		"vuid":                  tftypes.String,
	}}
	nullOpts := map[string]tftypes.Value{
		"remote_path":           tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"hostsallow":            tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"hostsdeny":             tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"aapl_name_mangling":    tftypes.NewValue(tftypes.Bool, nil),
		"auto_snapshot":         tftypes.NewValue(tftypes.Bool, nil),
		"auto_dataset_creation": tftypes.NewValue(tftypes.Bool, nil),
		"timemachine_quota":     tftypes.NewValue(tftypes.Number, nil),
		"auto_quota":            tftypes.NewValue(tftypes.Number, nil),
		"grace_period":          tftypes.NewValue(tftypes.Number, nil),
		"dataset_naming_schema": tftypes.NewValue(tftypes.String, nil),
		"vuid":                  tftypes.NewValue(tftypes.String, nil),
	}
	opts := func(set map[string]tftypes.Value) tftypes.Value {
		v := map[string]tftypes.Value{}
		for k, x := range nullOpts {
			v[k] = x
		}
		for k, x := range set {
			v[k] = x
		}
		return tftypes.NewValue(optsType, v)
	}
	strList := func(items ...string) tftypes.Value {
		var elems []tftypes.Value
		for _, s := range items {
			elems = append(elems, tftypes.NewValue(tftypes.String, s))
		}
		return tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, elems)
	}

	cases := []struct {
		name    string
		vals    map[string]tftypes.Value
		wantErr string
	}{
		{"default share, no options", map[string]tftypes.Value{
			"purpose": str("DEFAULT_SHARE"), "path": str("/mnt/tank"),
			"options": tftypes.NewValue(optsType, nil)}, ""},
		{"default share with hostsallow", map[string]tftypes.Value{
			"purpose": str("DEFAULT_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{"hostsallow": strList("10.0.0.0/8")})}, ""},
		{"remote_path on a default share", map[string]tftypes.Value{
			"purpose": str("DEFAULT_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{"remote_path": strList(`1.2.3.4\S`)})},
			"does not apply to purpose DEFAULT_SHARE"},
		{"external without options", map[string]tftypes.Value{
			"purpose": str("EXTERNAL_SHARE"), "path": str("EXTERNAL"),
			"options": tftypes.NewValue(optsType, nil)}, "requires an options block"},
		{"external with a local path", map[string]tftypes.Value{
			"purpose": str("EXTERNAL_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{"remote_path": strList(`1.2.3.4\S`)})},
			`path to be the literal string "EXTERNAL"`},
		{"external done right", map[string]tftypes.Value{
			"purpose": str("EXTERNAL_SHARE"), "path": str("EXTERNAL"),
			"options": opts(map[string]tftypes.Value{"remote_path": strList(`1.2.3.4\S`)})}, ""},
		{"remote_path without a backslash", map[string]tftypes.Value{
			"purpose": str("EXTERNAL_SHARE"), "path": str("EXTERNAL"),
			"options": opts(map[string]tftypes.Value{"remote_path": strList("no-backslash")})},
			"SERVER\\SHARE"},
		{"hostsallow on an external share", map[string]tftypes.Value{
			"purpose": str("EXTERNAL_SHARE"), "path": str("EXTERNAL"),
			"options": opts(map[string]tftypes.Value{
				"remote_path": strList(`1.2.3.4\S`), "hostsallow": strList("10.0.0.0/8")})},
			"does not apply to purpose EXTERNAL_SHARE"},
		{"grace_period too small", map[string]tftypes.Value{
			"purpose": str("TIME_LOCKED_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{
				"grace_period": tftypes.NewValue(tftypes.Number, 10)})}, "between 60 and"},
		{"grace_period at the floor", map[string]tftypes.Value{
			"purpose": str("TIME_LOCKED_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{
				"grace_period": tftypes.NewValue(tftypes.Number, 60)})}, ""},
		// The ceiling itself must be accepted. Without this, narrowing the
		// range would still reject everything the "above" case rejects and no
		// test would notice.
		{"grace_period at the ceiling", map[string]tftypes.Value{
			"purpose": str("TIME_LOCKED_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{
				"grace_period": tftypes.NewValue(tftypes.Number, 15552000)})}, ""},
		{"grace_period above the ceiling", map[string]tftypes.Value{
			"purpose": str("TIME_LOCKED_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{
				"grace_period": tftypes.NewValue(tftypes.Number, 15552001)})}, "between 60 and"},
		// FCP_SHARE reached the provider in 25.10.1 and takes the same three
		// options as DEFAULT_SHARE.
		{"fcp with aapl_name_mangling", map[string]tftypes.Value{
			"purpose": str("FCP_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{
				"aapl_name_mangling": tftypes.NewValue(tftypes.Bool, true)})}, ""},
		{"fcp with a time-locked field", map[string]tftypes.Value{
			"purpose": str("FCP_SHARE"), "path": str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{
				"grace_period": tftypes.NewValue(tftypes.Number, 600)})},
			"does not apply to purpose FCP_SHARE"},
		// purpose unset defaults to DEFAULT_SHARE server-side, so a
		// TIMEMACHINE-only field is still wrong.
		{"timemachine field with no purpose", map[string]tftypes.Value{
			"path":    str("/mnt/tank"),
			"options": opts(map[string]tftypes.Value{"auto_snapshot": tftypes.NewValue(tftypes.Bool, true)})},
			"does not apply to purpose DEFAULT_SHARE"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, tc.vals).Raw}
			resp := &resource.ValidateConfigResponse{}
			r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
			if tc.wantErr == "" {
				if resp.Diagnostics.HasError() {
					t.Fatalf("valid config rejected: %v", resp.Diagnostics)
				}
				return
			}
			if !resp.Diagnostics.HasError() {
				t.Fatal("invalid config accepted")
			}
			var joined string
			for _, d := range resp.Diagnostics.Errors() {
				joined += d.Detail()
			}
			if !strings.Contains(joined, tc.wantErr) {
				t.Errorf("diagnostic %q does not mention %q", joined, tc.wantErr)
			}
		})
	}

	t.Run("a config that cannot be decoded stops validation", func(t *testing.T) {
		bogus := tfsdk.Config{Schema: sch.Schema, Raw: tftypes.NewValue(tftypes.String, "nope")}
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: bogus}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ValidateConfig accepted a config it could not decode")
		}
	})
}

// Every options attribute in the schema must have a purpose mapping. Without
// this, adding one to the schema and forgetting the map entry would make it
// silently unvalidated: the lookup returns an empty list, nothing matches, and
// the attribute sails through to a middleware rejection instead.
func TestSMBOptionPurposesCoversSchema(t *testing.T) {
	ctx := context.Background()
	sch := schemaOf(t, ctx, &SMBShareResource{})

	nested, ok := sch.Schema.Attributes["options"].(fwschema.SingleNestedAttribute)
	if !ok {
		t.Fatal("options is not a SingleNestedAttribute")
	}
	for name := range nested.Attributes {
		if _, mapped := smbOptionPurposes[name]; !mapped {
			t.Errorf("options.%s has no entry in smbOptionPurposes, so it is never validated "+
				"against the share purpose", name)
		}
		if _, typed := smbOptionAttrTypes[name]; !typed {
			t.Errorf("options.%s has no entry in smbOptionAttrTypes", name)
		}
	}
	// And nothing stale in the other direction.
	for name := range smbOptionPurposes {
		if _, present := nested.Attributes[name]; !present {
			t.Errorf("smbOptionPurposes has %q, which is not an options attribute", name)
		}
	}
}

// An element that is unknown at apply time has no Go string to become. That
// must surface as an error rather than silently dropping the entry, and it
// must stop the write rather than sending a partial body.
func TestSMBOptions_unknownElementStopsTheWrite(t *testing.T) {
	ctx := context.Background()

	for _, field := range []string{"remote_path", "hostsallow", "hostsdeny"} {
		t.Run(field, func(t *testing.T) {
			o := smbOptsObject(t, map[string]attr.Value{
				field: types.ListValueMust(types.StringType, []attr.Value{types.StringUnknown()}),
			})
			_, err := smbOptionsFromModel(ctx, o)
			if err == nil {
				t.Fatalf("an unknown %s element was accepted", field)
			}
			if !strings.Contains(err.Error(), field) {
				t.Errorf("error should name the attribute, got: %v", err)
			}
		})
	}

	t.Run("Create and Update stop", func(t *testing.T) {
		var wrote bool
		c := newWSTestClient(ctx, t, func(_ context.Context, m string, _ []interface{}) (interface{}, *wsclient.RPCError) {
			if strings.HasPrefix(m, "sharing.smb.create") || strings.HasPrefix(m, "sharing.smb.update") {
				wrote = true
			}
			return map[string]interface{}{"id": 1, "path": "/mnt/tank", "name": "n"}, nil
		})
		r := &SMBShareResource{client: c}
		sch := schemaOf(t, ctx, r)

		optsType := sch.Schema.Attributes["options"].GetType().TerraformType(ctx)
		vals := map[string]tftypes.Value{
			"id": str("1"), "path": str("/mnt/tank"), "name": str("n"),
			"options": tftypes.NewValue(optsType, map[string]tftypes.Value{
				"remote_path": tftypes.NewValue(tftypes.List{ElementType: tftypes.String},
					[]tftypes.Value{tftypes.NewValue(tftypes.String, tftypes.UnknownValue)}),
				"hostsallow":            tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
				"hostsdeny":             tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
				"aapl_name_mangling":    tftypes.NewValue(tftypes.Bool, nil),
				"auto_snapshot":         tftypes.NewValue(tftypes.Bool, nil),
				"auto_dataset_creation": tftypes.NewValue(tftypes.Bool, nil),
				"timemachine_quota":     tftypes.NewValue(tftypes.Number, nil),
				"auto_quota":            tftypes.NewValue(tftypes.Number, nil),
				"grace_period":          tftypes.NewValue(tftypes.Number, nil),
				"dataset_naming_schema": tftypes.NewValue(tftypes.String, nil),
				"vuid":                  tftypes.NewValue(tftypes.String, nil),
			}),
		}
		st := stateFromValues(t, ctx, sch, vals)
		plan := planFromValues(t, ctx, sch, vals)

		cResp := &resource.CreateResponse{State: st}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, cResp)
		if !cResp.Diagnostics.HasError() {
			t.Error("Create accepted a body it could not build")
		}
		uResp := &resource.UpdateResponse{State: st}
		r.Update(ctx, resource.UpdateRequest{State: st, Plan: plan}, uResp)
		if !uResp.Diagnostics.HasError() {
			t.Error("Update accepted a body it could not build")
		}
		if wrote {
			t.Error("a write went out despite the body failing to build")
		}
	})
}

// A null element inside remote_path is skipped rather than judged: there is
// nothing to check, and reporting it as malformed would be wrong.
func TestSMBOptions_nullRemotePathElementSkipped(t *testing.T) {
	ctx := context.Background()
	r := &SMBShareResource{}
	sch := schemaOf(t, ctx, r)
	optsType := sch.Schema.Attributes["options"].GetType().TerraformType(ctx)

	vals := map[string]tftypes.Value{
		"purpose": str("EXTERNAL_SHARE"), "path": str("EXTERNAL"),
		"options": tftypes.NewValue(optsType, map[string]tftypes.Value{
			"remote_path": tftypes.NewValue(tftypes.List{ElementType: tftypes.String},
				[]tftypes.Value{tftypes.NewValue(tftypes.String, nil)}),
			"hostsallow":            tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"hostsdeny":             tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"aapl_name_mangling":    tftypes.NewValue(tftypes.Bool, nil),
			"auto_snapshot":         tftypes.NewValue(tftypes.Bool, nil),
			"auto_dataset_creation": tftypes.NewValue(tftypes.Bool, nil),
			"timemachine_quota":     tftypes.NewValue(tftypes.Number, nil),
			"auto_quota":            tftypes.NewValue(tftypes.Number, nil),
			"grace_period":          tftypes.NewValue(tftypes.Number, nil),
			"dataset_naming_schema": tftypes.NewValue(tftypes.String, nil),
			"vuid":                  tftypes.NewValue(tftypes.String, nil),
		}),
	}
	cfg := tfsdk.Config{Schema: sch.Schema, Raw: stateFromValues(t, ctx, sch, vals).Raw}
	resp := &resource.ValidateConfigResponse{}
	r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: cfg}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("a null remote_path element was judged instead of skipped: %v", resp.Diagnostics)
	}
}

// The string fields must round-trip, not just read back null.
func TestSMBOptionsToObject_stringsRoundTrip(t *testing.T) {
	schema := "%D/%U"
	vuid := "uuid-9"
	obj := smbOptionsToObject(&truenas.SMBShareOptions{
		DatasetNamingSchema: &schema,
		VUID:                &vuid,
		RemotePath:          &[]string{`srv\share`},
	})
	a := obj.Attributes()
	if v, _ := a["dataset_naming_schema"].(types.String); v.ValueString() != schema {
		t.Errorf("dataset_naming_schema = %v", a["dataset_naming_schema"])
	}
	if v, _ := a["vuid"].(types.String); v.ValueString() != vuid {
		t.Errorf("vuid = %v", a["vuid"])
	}
	if v, _ := a["remote_path"].(types.List); v.IsNull() || len(v.Elements()) != 1 {
		t.Errorf("remote_path = %v", a["remote_path"])
	}
}

// Every purpose the schema accepts must appear in at least one options list,
// and every purpose named in an options list must be one the schema accepts.
// A purpose missing from the map would silently reject all of its options; a
// purpose in the map but not the vocabulary is dead weight that reads as
// support for something the validator refuses.
func TestSMBPurposeVocabularyIsConsistent(t *testing.T) {
	accepted := make(map[string]bool, len(smbPurposes))
	for _, p := range smbPurposes {
		accepted[p] = true
	}

	used := map[string]bool{}
	for attrName, purposes := range smbOptionPurposes {
		for _, p := range purposes {
			used[p] = true
			if !accepted[p] {
				t.Errorf("smbOptionPurposes[%q] names %q, which the purpose validator rejects", attrName, p)
			}
		}
	}
	for _, p := range smbPurposes {
		if !used[p] {
			t.Errorf("purpose %q accepts no options at all; if that is right, say so here, "+
				"otherwise its options are being rejected", p)
		}
	}
}

// The validator must actually be driven by smbPurposes. Reading it back off the
// schema is what proves the single source reaches the schema, rather than the
// list existing beside a hand-maintained copy.
func TestSMBPurposeValidatorUsesTheVocabulary(t *testing.T) {
	ctx := context.Background()
	sch := schemaOf(t, ctx, &SMBShareResource{})
	attrDef, ok := sch.Schema.Attributes["purpose"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("purpose is not a StringAttribute")
	}
	if len(attrDef.Validators) == 0 {
		t.Fatal("purpose has no validators, so any string would be accepted")
	}
	desc := attrDef.Validators[0].Description(ctx)
	for _, p := range smbPurposes {
		if !strings.Contains(desc, p) {
			t.Errorf("purpose validator does not accept %q (description: %s)", p, desc)
		}
	}
	if strings.Contains(desc, "TIMEMACHINE\"") || strings.Contains(desc, "NO_PRESET") {
		t.Errorf("purpose validator still carries retired vocabulary: %s", desc)
	}
}
