package resources

import (
	"context"
	"strings"
	"testing"

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
