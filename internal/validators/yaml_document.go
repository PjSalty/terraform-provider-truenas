package validators

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"gopkg.in/yaml.v3"
)

// yamlDocumentValidator validates that a string parses as a non-empty
// YAML document whose top level is a mapping. Wired on
// truenas_app.custom_compose so a broken compose document fails at
// plan time with a line-accurate parse error instead of a mid-apply
// middleware job failure. Structural compose validation (service
// shape, image references, port syntax) stays server-side, this only
// guarantees "is YAML, is a mapping, is not empty".
type yamlDocumentValidator struct{}

// YAMLDocument returns a validator.String asserting the value is a
// non-empty YAML document with a top-level mapping.
func YAMLDocument() validator.String {
	return yamlDocumentValidator{}
}

func (v yamlDocumentValidator) Description(_ context.Context) string {
	return "value must be a non-empty YAML document with a top-level mapping"
}

func (v yamlDocumentValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v yamlDocumentValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	raw := req.ConfigValue.ValueString()
	if strings.TrimSpace(raw) == "" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid YAML Document",
			"value is empty, expected a YAML document (for example a Docker Compose file)",
		)
		return
	}
	dec := yaml.NewDecoder(strings.NewReader(raw))
	var doc map[string]interface{}
	if err := dec.Decode(&doc); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid YAML Document",
			fmt.Sprintf("value does not parse as a YAML mapping: %s", err),
		)
		return
	}
	if len(doc) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid YAML Document",
			"value parses as YAML but has no top-level mapping keys, a Docker Compose file needs at least a services: map",
		)
		return
	}
	// fail closed on multi-document streams: yaml.Unmarshal would
	// silently take the first document and ignore the rest, including
	// garbage after a --- separator
	var extra interface{}
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid YAML Document",
			"value contains more than one YAML document, custom_compose takes a single compose document (remove the --- separator and anything after it)",
		)
	}
}
