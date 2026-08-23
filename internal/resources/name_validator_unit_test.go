package resources

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	fwschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestNameValidatorsMatchUpstream drives attribute validators directly, off the
// schema. That is the layer a ValidateConfig test cannot reach, because schema
// validators run first, and the layer statement coverage cannot see, because a
// validator is declarative data rather than a statement.
//
// It is where six name rules had drifted from what TrueNAS accepts: john.doe
// was not a valid username, web_server was not a valid VM name, and an FTP
// umask of 1777 passed plan and then failed at apply. Each case cites the
// upstream rule it encodes.
func TestNameValidatorsMatchUpstream(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		what     string
		res      resource.Resource
		attr     string
		upstream string
		valid    []string
		invalid  []string
	}{
		{
			what: "user.username", res: &UserResource{}, attr: "username",
			upstream: "validate_local_username: chars letters+digits+_-. , start letters+_ , max 32",
			valid:    []string{"alice", "John", "john.doe", "sys_user", "a-b", "_svc", strings.Repeat("a", 32)},
			invalid:  []string{"", "1alice", "-alice", ".alice", "al ice", "alice$", strings.Repeat("a", 33)},
		},
		{
			what: "group.name", res: &GroupResource{}, attr: "name",
			upstream: "GroupName: chars letters+digits+_-. , start letters+digits+_. , no length cap",
			valid:    []string{"devs", "Devs", "docker.users", "0group", "_g", "a-b", strings.Repeat("g", 64)},
			invalid:  []string{"", "-devs", "dev s", "devs$"},
		},
		{
			what: "vm.name", res: &VMResource{}, attr: "name",
			upstream: "NonEmptyString: no character rule at all",
			valid:    []string{"vm1", "web_server", "web-server", "Web Server", "vm.one"},
			invalid:  []string{"", strings.Repeat("v", 151)},
		},
		{
			what: "iscsi_target.name", res: &ISCSITargetResource{}, attr: "name",
			upstream: "RE_TARGET_NAME ^[-a-z0-9.:]+$ with max_length 120",
			valid:    []string{"tgt0", "-tgt0", ".2005-10.org.example:tgt0", "iqn.2005-10.org.example:tgt0"},
			invalid:  []string{"", "TGT0", "tgt_0", "tgt 0", strings.Repeat("t", 121)},
		},
		{
			what: "ftp_config.filemask", res: &FTPConfigResource{}, attr: "filemask",
			upstream: "UnixPerm: parses as octal and mode & 0o777 == mode",
			valid:    []string{"0", "77", "777", "077", "0777", "000"},
			invalid:  []string{"", "778", "1777", "7777", "abc"},
		},
		{
			what: "snmp_config.v3_privproto", res: &SNMPConfigResource{}, attr: "v3_privproto",
			upstream: "Literal[None, 'AES', 'DES']: null or one of the two, never an empty string",
			valid:    []string{"AES", "DES"},
			invalid:  []string{"", "aes", "3DES"},
		},
		{
			what: "ftp_config.dirmask", res: &FTPConfigResource{}, attr: "dirmask",
			upstream: "UnixPerm: parses as octal and mode & 0o777 == mode",
			valid:    []string{"22", "022", "755"},
			invalid:  []string{"1777", "888"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.what, func(t *testing.T) {
			sch := schemaOf(t, ctx, tc.res)
			a, ok := sch.Schema.Attributes[tc.attr].(fwschema.StringAttribute)
			if !ok {
				t.Fatalf("%s is not a StringAttribute", tc.attr)
			}
			if len(a.Validators) == 0 {
				t.Fatalf("%s has no validators, so any string is accepted", tc.attr)
			}
			check := func(in string) bool {
				resp := &validator.StringResponse{}
				for _, v := range a.Validators {
					v.ValidateString(ctx, validator.StringRequest{
						Path:        path.Root(tc.attr),
						ConfigValue: types.StringValue(in),
					}, resp)
				}
				return resp.Diagnostics.HasError()
			}
			for _, in := range tc.valid {
				if check(in) {
					t.Errorf("%q is rejected but upstream accepts it (%s)", in, tc.upstream)
				}
			}
			for _, in := range tc.invalid {
				if !check(in) {
					t.Errorf("%q is accepted but upstream rejects it (%s)", in, tc.upstream)
				}
			}
		})
	}
}

// TestNameserverValidatorAcceptsEmpty pins the clear-a-nameserver case and the
// two shapes the old regex got wrong in opposite directions.
func TestNameserverValidatorAcceptsEmpty(t *testing.T) {
	ctx := context.Background()
	v := nameserverValidator{}
	check := func(in string) bool {
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, validator.StringRequest{
			Path: path.Root("nameserver1"), ConfigValue: types.StringValue(in),
		}, resp)
		return resp.Diagnostics.HasError()
	}
	for _, in := range []string{"", "1.1.1.1", "2606:4700:4700::1111", "::ffff:1.2.3.4"} {
		if check(in) {
			t.Errorf("%q rejected; upstream IPvAnyAddress is Literal[''] plus a real IP parse", in)
		}
	}
	for _, in := range []string{"999.999.999.999", "not-an-ip", "1.1.1", "deadbeef"} {
		if !check(in) {
			t.Errorf("%q accepted; it is not an IP address", in)
		}
	}
	if v.Description(ctx) == "" || v.MarkdownDescription(ctx) == "" {
		t.Error("validator has no description")
	}

	// A null or unknown value is not the practitioner's to justify: it is
	// absent, or not resolved yet. Judging it would fail plans that are still
	// waiting on another resource.
	for _, in := range []types.String{types.StringNull(), types.StringUnknown()} {
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, validator.StringRequest{
			Path: path.Root("nameserver1"), ConfigValue: in,
		}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%v was judged; null and unknown must pass through", in)
		}
	}
}

// TestContainerIdmapWireShape pins the discriminator-sensitive slice key.
// ISOLATED must carry it, null meaning "pick one"; DEFAULT must not carry it
// at all, because that union member forbids extras.
func TestContainerIdmapWireShape(t *testing.T) {
	obj := func(typ string, slice types.Int64) types.Object {
		return types.ObjectValueMust(containerIdmapAttrTypes, map[string]attr.Value{
			"type": types.StringValue(typ), "slice": slice,
		})
	}

	got := containerIdmapFromObject(obj(containerIdmapIsolated, types.Int64Null()))
	if got == nil || got.Slice == nil {
		t.Fatalf("ISOLATED with no slice must still send the key, got %+v", got)
	}
	if *got.Slice != nil {
		t.Errorf("ISOLATED with no slice must send null, got %v", **got.Slice)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(body), `"slice":null`) {
		t.Errorf("ISOLATED body must contain a null slice, got %s", body)
	}

	got = containerIdmapFromObject(obj(containerIdmapIsolated, types.Int64Value(7)))
	if got == nil || got.Slice == nil || *got.Slice == nil || **got.Slice != 7 {
		t.Fatalf("explicit slice lost, got %+v", got)
	}

	// An unset authmethod must fall back to the upstream default rather than
	// going out as an empty string, which is not a member of IscsiAuthType.
	wire := iscsiGroupToWire(ISCSITargetGroupModel{
		Portal:     types.Int64Value(1),
		Initiator:  types.Int64Null(),
		AuthMethod: types.StringNull(),
		Auth:       types.Int64Null(),
	})
	if wire.AuthMethod != iscsiAuthMethodNone {
		t.Errorf("authmethod = %q, want the upstream default %q", wire.AuthMethod, iscsiAuthMethodNone)
	}
	if wire.Initiator != nil || wire.Auth != nil {
		t.Errorf("an unset initiator or auth must be null, got %v / %v", wire.Initiator, wire.Auth)
	}

	got = containerIdmapFromObject(obj(containerIdmapDefault, types.Int64Null()))
	if got == nil {
		t.Fatal("DEFAULT produced no idmap")
	}
	if got.Slice != nil {
		t.Errorf("DEFAULT must not carry a slice key, got %v", got.Slice)
	}
	body, err = json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(body), "slice") {
		t.Errorf("DEFAULT body must omit slice entirely, got %s", body)
	}
}

// TestOptionalAttributesMatchUpstream pins attributes upstream defaults but the
// provider used to force the practitioner to supply.
func TestOptionalAttributesMatchUpstream(t *testing.T) {
	ctx := context.Background()

	tgt := schemaOf(t, ctx, &ISCSITargetResource{})
	groups, ok := tgt.Schema.Attributes["groups"].(fwschema.ListNestedAttribute)
	if !ok {
		t.Fatal("groups is not a ListNestedAttribute")
	}
	init := groups.NestedObject.Attributes["initiator"]
	if init.IsRequired() {
		t.Error("groups.initiator is Required, but upstream defaults it to null, " +
			"which means allow any initiator")
	}

	key := schemaOf(t, ctx, &APIKeyResource{})
	user, ok := key.Schema.Attributes["username"].(fwschema.StringAttribute)
	if !ok {
		t.Fatal("api_key.username is not a StringAttribute")
	}
	long := strings.Repeat("a", 40) + "@corp.example.com"
	resp := &validator.StringResponse{}
	for _, v := range user.Validators {
		v.ValidateString(ctx, validator.StringRequest{
			Path: path.Root("username"), ConfigValue: types.StringValue(long),
		}, resp)
	}
	if resp.Diagnostics.HasError() {
		t.Errorf("a %d-character username is rejected, but upstream accepts "+
			"LocalUsername | RemoteUsername and the remote arm has no maximum", len(long))
	}
}
