package resources

import (
	"testing"

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
		{"setgid four digit", types.StringValue("2775"), 0o42775, "2775"},
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
