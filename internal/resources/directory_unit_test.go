package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	truenas "github.com/PjSalty/terraform-provider-truenas/internal/types"
)

// mapStatToModel keeps the user's octal spelling when it matches the
// on-disk perm bits (so "0755" and "755" don't churn the plan), and
// overwrites on real drift or when mode is unset (import).
func TestDirectoryResource_MapStatToModel_ModeRoundTrip(t *testing.T) {
	r := &DirectoryResource{}
	cases := []struct {
		name     string
		inMode   types.String
		statMode int // full st_mode incl the directory type bits
		wantMode string
	}{
		{"three digit match", types.StringValue("755"), 0o40755, "755"},
		{"leading zero kept", types.StringValue("0755"), 0o40755, "0755"},
		{"drift overwrites", types.StringValue("755"), 0o40700, "700"},
		{"unset on import", types.StringNull(), 0o40755, "755"},
		{"out of band setgid ignored", types.StringValue("775"), 0o42775, "775"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &DirectoryResourceModel{
				Path: types.StringValue("/mnt/tank/x"),
				Mode: tc.inMode,
			}
			r.mapStatToModel(&truenas.FilesystemStat{Mode: tc.statMode, UID: 1000, GID: 1000}, m)
			if got := m.Mode.ValueString(); got != tc.wantMode {
				t.Fatalf("mode: got %q want %q", got, tc.wantMode)
			}
			if m.ID.ValueString() != "/mnt/tank/x" {
				t.Fatalf("id: got %q want the path", m.ID.ValueString())
			}
			if m.UID.ValueInt64() != 1000 || m.GID.ValueInt64() != 1000 {
				t.Fatalf("uid/gid: got %d/%d want 1000/1000", m.UID.ValueInt64(), m.GID.ValueInt64())
			}
		})
	}
}

// the mode validator must reject special-bit modes: the middleware
// UnixPerm type behind filesystem.mkdir and filesystem.setperm only
// accepts 000-777, so a 4-digit mode would fail at apply (issue #17).
func TestDirectoryResource_ModeValidator_RejectsSpecialBits(t *testing.T) {
	ctx := context.Background()
	r := NewDirectoryResource()
	sch := schemaOf(t, ctx, r)
	modeAttr, ok := sch.Schema.GetAttributes()["mode"].(schema.StringAttribute)
	if !ok {
		t.Fatal("mode attribute missing or not a StringAttribute")
	}
	cases := []struct {
		val     string
		wantErr bool
	}{
		{"755", false},
		{"000", false},
		{"777", false},
		{"0755", false},
		{"2770", true},
		{"4755", true},
		{"1777", true},
		{"77", true},
	}
	for _, tc := range cases {
		t.Run(tc.val, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("mode"),
				ConfigValue: types.StringValue(tc.val),
			}
			resp := &validator.StringResponse{}
			for _, v := range modeAttr.Validators {
				v.ValidateString(ctx, req, resp)
			}
			if resp.Diagnostics.HasError() != tc.wantErr {
				t.Fatalf("mode %q: gotErr=%v wantErr=%v (%v)", tc.val, resp.Diagnostics.HasError(), tc.wantErr, resp.Diagnostics)
			}
		})
	}
}
