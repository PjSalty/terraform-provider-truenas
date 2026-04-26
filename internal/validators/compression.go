package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// validCompressionAlgorithms lists all compression values accepted by TrueNAS SCALE.
var validCompressionAlgorithms = []string{
	"OFF",
	"LZ4",
	"GZIP",
	"GZIP-1",
	"GZIP-2",
	"GZIP-3",
	"GZIP-4",
	"GZIP-5",
	"GZIP-6",
	"GZIP-7",
	"GZIP-8",
	"GZIP-9",
	"ZSTD",
	"ZSTD-FAST",
	"ZLE",
	"LZJB",
}

// compressionValidator validates that a string is a recognized TrueNAS
// compression algorithm name.
type compressionValidator struct{}

// CompressionAlgorithm returns a validator.String for the TrueNAS compression
// enum.  Comparison is case-insensitive so that "lz4" and "LZ4" are both
// accepted (TrueNAS normalises to upper-case in its responses).
func CompressionAlgorithm() validator.String {
	return compressionValidator{}
}

func (v compressionValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value must be one of: %s", strings.Join(validCompressionAlgorithms, ", "))
}

func (v compressionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v compressionValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.ToUpper(strings.TrimSpace(req.ConfigValue.ValueString()))

	for _, valid := range validCompressionAlgorithms {
		if val == valid {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Compression Algorithm",
		fmt.Sprintf(
			"%q is not a valid TrueNAS compression algorithm. Valid values: %s",
			req.ConfigValue.ValueString(),
			strings.Join(validCompressionAlgorithms, ", "),
		),
	)
}
