package planhelpers

import (
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// pathRootID returns the path.Path pointing at the top-level `id`
// attribute. Factored out so the destroy_warning helper can pass it
// to Config/State/Plan GetAttribute without importing path in every
// caller. Kept in its own file because the path package is small
// and re-used across multiple helpers planned for this package.
func pathRootID() path.Path {
	return path.Root("id")
}
