package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestDirectoryResource_CRUD drives Create/Read/Update/Delete against the
// mock WS server, exercising mkdir + stat + the setperm job and the
// state-only Delete path. Mirrors the zz_crud batch style used for every
// other resource. The not-found branches are covered in
// zz_notfound_batch_test.go, and the stat->model mapping is asserted in
// directory_unit_test.go.
func TestDirectoryResource_CRUD(t *testing.T) {
	body := map[string]interface{}{
		"realpath": "/mnt/tank/d", "type": "DIRECTORY", "size": 4096,
		"mode": 0o40755, "uid": 1000, "gid": 1000,
		"acl": false, "is_mountpoint": false,
	}
	c := newWSJSONServerClient(t, body)
	r := &DirectoryResource{client: c}
	crudDrive(t, r, c, "/mnt/tank/d", map[string]tftypes.Value{
		"path":           str("/mnt/tank/d"),
		"mode":           str("755"),
		"create_parents": flag(false),
		"uid":            num(1000),
		"gid":            num(1000),
	})
}
