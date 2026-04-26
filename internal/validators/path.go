// Package validators provides custom terraform-plugin-framework validators
// for TrueNAS-specific attribute types.
package validators

import (
	"context"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// zfsPathValidator validates that a string is a legal ZFS dataset path.
// A valid path consists of one or more components separated by '/', where
// each component may contain alphanumerics plus: - _ . : @
// Spaces are never allowed.
type zfsPathValidator struct{}

// ZFSPath returns a validator.String that rejects values containing spaces
// or characters illegal in ZFS dataset names.
func ZFSPath() validator.String {
	return zfsPathValidator{}
}

var zfsComponentRe = regexp.MustCompile(`^[a-zA-Z0-9\-_.:@]+$`)

func (v zfsPathValidator) Description(_ context.Context) string {
	return "value must be a valid ZFS dataset path (no spaces; components may contain a-z A-Z 0-9 - _ . : @)"
}

func (v zfsPathValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v zfsPathValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := req.ConfigValue.ValueString()

	if strings.ContainsRune(val, ' ') {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid ZFS Path",
			"ZFS dataset paths must not contain spaces.",
		)
		return
	}

	components := strings.Split(val, "/")
	for _, component := range components {
		if component == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid ZFS Path",
				"ZFS dataset path components must not be empty (no leading, trailing, or consecutive slashes).",
			)
			return
		}
		if !zfsComponentRe.MatchString(component) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid ZFS Path",
				"ZFS dataset path component "+component+" contains illegal characters. "+
					"Allowed: a-z A-Z 0-9 - _ . : @",
			)
			return
		}
	}
}
