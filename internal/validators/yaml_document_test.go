package validators_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	lv "github.com/PjSalty/terraform-provider-truenas/internal/validators"
)

func yamlDocReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("custom_compose"),
		PathExpression: path.MatchRoot("custom_compose"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

func TestYAMLDocument_Valid(t *testing.T) {
	cases := []string{
		// minimal compose
		"services:\n  app:\n    image: busybox:1.36\n    command: [\"sleep\", \"infinity\"]\n    restart: unless-stopped\n",
		// compose with env + volumes
		"services:\n  web:\n    image: nginx:1.27\n    environment:\n      FOO: bar\n    volumes:\n      - /mnt/tank/web:/usr/share/nginx/html\n",
		// any top-level mapping is accepted, compose shape is server-side
		"version: \"3.9\"\nservices: {}\n",
	}
	v := lv.YAMLDocument()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), yamlDocReq(tc), resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("YAMLDocument(%q) got unexpected error: %v", tc, resp.Diagnostics)
		}
	}
}

func TestYAMLDocument_Invalid(t *testing.T) {
	cases := []string{
		// empty and whitespace-only
		"",
		"   \n\t",
		// tab-indented, YAML forbids tabs for indentation
		"services:\n\tapp:\n\t\timage: busybox\n",
		// plain garbage, not a mapping
		"not: [valid",
		"just a scalar string",
		"- a\n- list\n",
		// parses but empty mapping
		"{}",
		"# only a comment\n",
		// multi-document streams fail closed: yaml.Unmarshal would
		// silently drop everything after the first doc
		"services:\n  app:\n    image: busybox\n---\nnot: [valid",
		"services:\n  app:\n    image: busybox\n---\nsecond: doc\n",
	}
	v := lv.YAMLDocument()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), yamlDocReq(tc), resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("YAMLDocument(%q) expected error, got none", tc)
		}
	}
}

func TestYAMLDocument_NullUnknownSkipped(t *testing.T) {
	v := lv.YAMLDocument()
	for _, cv := range []types.String{types.StringNull(), types.StringUnknown()} {
		req := validator.StringRequest{
			Path:           path.Root("custom_compose"),
			PathExpression: path.MatchRoot("custom_compose"),
			ConfigValue:    cv,
		}
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("YAMLDocument null/unknown should be skipped, got: %v", resp.Diagnostics)
		}
	}
}

func TestYAMLDocument_Descriptions(t *testing.T) {
	v := lv.YAMLDocument()
	d := v.Description(context.Background())
	if d == "" || d != v.MarkdownDescription(context.Background()) {
		t.Errorf("descriptions mismatch or empty: %q", d)
	}
}
