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

func stringReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("test"),
		PathExpression: path.MatchRoot("test"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

func TestZFSPath_Valid(t *testing.T) {
	cases := []string{
		"tank",
		"tank/data",
		"tank/data/sub",
		"pool-1/dataset_2",
		"pool.mirror",
	}
	v := lv.ZFSPath()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), stringReq(tc), resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ZFSPath(%q) got unexpected error: %v", tc, resp.Diagnostics)
		}
	}
}

func TestZFSPath_Invalid(t *testing.T) {
	cases := []string{
		"tank/data with spaces",
		"tank//double",
		"tank/",
		"tank/bad!char",
		"tank/bad char",
	}
	v := lv.ZFSPath()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), stringReq(tc), resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("ZFSPath(%q) expected error, got none", tc)
		}
	}
}

func TestZFSPath_NullUnknownSkipped(t *testing.T) {
	v := lv.ZFSPath()
	// Null
	respNull := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("test"),
		PathExpression: path.MatchRoot("test"),
		ConfigValue:    types.StringNull(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, nil)},
	}, respNull)
	if respNull.Diagnostics.HasError() {
		t.Errorf("null value should be skipped, got: %v", respNull.Diagnostics)
	}
	// Unknown
	respUnk := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("test"),
		PathExpression: path.MatchRoot("test"),
		ConfigValue:    types.StringUnknown(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
	}, respUnk)
	if respUnk.Diagnostics.HasError() {
		t.Errorf("unknown value should be skipped, got: %v", respUnk.Diagnostics)
	}
}

func TestZFSPath_Descriptions(t *testing.T) {
	v := lv.ZFSPath()
	ctx := context.Background()
	desc := v.Description(ctx)
	if desc == "" {
		t.Error("Description() returned empty string")
	}
	md := v.MarkdownDescription(ctx)
	if md == "" {
		t.Error("MarkdownDescription() returned empty string")
	}
	if md != desc {
		t.Errorf("MarkdownDescription should equal Description; got %q vs %q", md, desc)
	}
}
