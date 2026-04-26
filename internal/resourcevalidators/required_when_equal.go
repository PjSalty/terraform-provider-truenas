// Package resourcevalidators contains cross-attribute resource
// ConfigValidators — the Plugin Framework equivalent of the
// `ConflictsWith`, `ExactlyOneOf`, and conditional-requirement
// helpers found in many Terraform providers. Each validator returned
// by this package is
// safe to embed into a resource's `ConfigValidators(ctx)` hook and
// runs at config-validation time (before Plan), so invalid HCL is
// caught BEFORE any network round-trip.
package resourcevalidators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RequiredWhenEqual returns a resource.ConfigValidator that enforces
// a conditional-requirement rule: when the discriminator attribute
// (a String) equals the trigger value, every attribute in required
// must be set to a non-null, non-unknown value.
//
// Typical use is a `create_type`/`protocol`/`auth_method` discriminator
// pattern, where one enum value gates which other attributes are
// meaningful. For example, truenas_certificate uses
// this to enforce that `certificate` and `privatekey` are both set
// when `create_type = "CERTIFICATE_CREATE_IMPORTED"`, but not for
// the other three create_type values.
//
// The validator reports the missing attributes as attribute-scoped
// diagnostics so the operator sees the exact path:
//
//	Error: Missing required attribute
//	  with truenas_certificate.test,
//	  on main.tf line 3, in resource "truenas_certificate" "test":
//	   3:   create_type = "CERTIFICATE_CREATE_IMPORTED"
//	Attribute "certificate" is required when create_type = "CERTIFICATE_CREATE_IMPORTED"
//
// Discriminator values that are unknown at validate time (for
// example, a reference to another resource's output) cause the
// validator to silently skip — the framework will re-validate at
// plan time when the reference resolves.
func RequiredWhenEqual(discriminator, trigger string, required []string) resource.ConfigValidator {
	return requiredWhenEqualValidator{
		discriminator: discriminator,
		trigger:       trigger,
		required:      required,
	}
}

type requiredWhenEqualValidator struct {
	discriminator string
	trigger       string
	required      []string
}

func (v requiredWhenEqualValidator) Description(_ context.Context) string {
	return fmt.Sprintf("when %s = %q, the following attributes must be set: %v",
		v.discriminator, v.trigger, v.required)
}

func (v requiredWhenEqualValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v requiredWhenEqualValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var discVal types.String
	diags := req.Config.GetAttribute(ctx, path.Root(v.discriminator), &discVal)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Unknown discriminator: can't know yet — defer to plan time.
	if discVal.IsNull() || discVal.IsUnknown() {
		return
	}
	if discVal.ValueString() != v.trigger {
		return
	}
	// Trigger matched — every required attribute must be non-null
	// and non-empty. We report EACH missing attribute individually so
	// the operator sees the full punch list, not just the first one.
	// Crucially we short-circuit on per-call GetAttribute errors only,
	// not on the cumulative diagnostic bag — otherwise the first
	// AddAttributeError would silence every subsequent check.
	for _, name := range v.required {
		var val types.String
		getDiags := req.Config.GetAttribute(ctx, path.Root(name), &val)
		resp.Diagnostics.Append(getDiags...)
		if getDiags.HasError() {
			continue
		}
		if val.IsNull() || val.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root(name),
				"Missing required attribute",
				fmt.Sprintf("Attribute %q is required when %s = %q.",
					name, v.discriminator, v.trigger),
			)
		}
	}
}
