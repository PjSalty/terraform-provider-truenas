package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// validCompressionAlgorithms lists every compression value the TrueNAS
// SCALE 25.10 dataset API accepts. Verified at runtime against
// `pool.dataset.compression_choices` on a 25.10.0 instance.
//
// Three notes:
//  1. "ON" is a valid value, it means "use the pool default", which
//     TrueNAS normalises in the response. Earlier provider versions
//     omitted it and silently rejected configs using it.
//  2. The full ZSTD-N ladder runs 1–19, not just the GZIP-style 1–9.
//     Earlier provider versions only allowed GZIP-1 through GZIP-9
//     and ZSTD / ZSTD-FAST without the numeric variants, which
//     rejected valid configs.
//  3. ZSTD-FAST has both N variants (1–10) and skip-step
//     N×{20,30,…,90,100,500,1000} variants. iX maintains this exact
//     list in middlewared; pin it here verbatim and regenerate from
//     `pool.dataset.compression_choices` when probing a new SCALE.
var validCompressionAlgorithms = []string{
	"ON", "OFF",
	"LZ4", "LZJB", "ZLE",
	"GZIP", "GZIP-1", "GZIP-9",
	"ZSTD", "ZSTD-FAST",
	"ZSTD-1", "ZSTD-2", "ZSTD-3", "ZSTD-4", "ZSTD-5",
	"ZSTD-6", "ZSTD-7", "ZSTD-8", "ZSTD-9", "ZSTD-10",
	"ZSTD-11", "ZSTD-12", "ZSTD-13", "ZSTD-14", "ZSTD-15",
	"ZSTD-16", "ZSTD-17", "ZSTD-18", "ZSTD-19",
	"ZSTD-FAST-1", "ZSTD-FAST-2", "ZSTD-FAST-3", "ZSTD-FAST-4", "ZSTD-FAST-5",
	"ZSTD-FAST-6", "ZSTD-FAST-7", "ZSTD-FAST-8", "ZSTD-FAST-9", "ZSTD-FAST-10",
	"ZSTD-FAST-20", "ZSTD-FAST-30", "ZSTD-FAST-40", "ZSTD-FAST-50",
	"ZSTD-FAST-60", "ZSTD-FAST-70", "ZSTD-FAST-80", "ZSTD-FAST-90",
	"ZSTD-FAST-100", "ZSTD-FAST-500", "ZSTD-FAST-1000",
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
