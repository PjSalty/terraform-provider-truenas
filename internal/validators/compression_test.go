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

func compressionReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("compression"),
		PathExpression: path.MatchRoot("compression"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

func TestCompressionAlgorithm_Valid(t *testing.T) {
	cases := []string{
		"OFF", "LZ4", "GZIP", "GZIP-1", "GZIP-9",
		"ZSTD", "ZSTD-FAST", "ZLE", "LZJB",
		// Case-insensitive
		"lz4", "gzip", "zstd",
	}
	v := lv.CompressionAlgorithm()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), compressionReq(tc), resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("CompressionAlgorithm(%q) got unexpected error: %v", tc, resp.Diagnostics)
		}
	}
}

func TestCompressionAlgorithm_Invalid(t *testing.T) {
	cases := []string{
		"INVALID",
		"ZLIB",
		"BROTLI",
		"lz77",
		"snappy",
	}
	v := lv.CompressionAlgorithm()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), compressionReq(tc), resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("CompressionAlgorithm(%q) expected error, got none", tc)
		}
	}
}

func TestCompressionAlgorithm_NullSkipped(t *testing.T) {
	v := lv.CompressionAlgorithm()
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("compression"),
		PathExpression: path.MatchRoot("compression"),
		ConfigValue:    types.StringNull(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, nil)},
	}, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("null value should be skipped, got error: %v", resp.Diagnostics)
	}
}

func TestCompressionAlgorithm_Descriptions(t *testing.T) {
	v := lv.CompressionAlgorithm()
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
