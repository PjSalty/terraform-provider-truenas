package validators_test

// Fuzz tests for validators. Property under test: ValidateString must never
// panic for any input string. Validation errors are expected and fine; the
// goal is to catch crashes on pathological input (nil pointers, index out
// of range, regex panics, etc.).

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

// makeReq builds a validator.StringRequest with the given raw value.
func makeReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("fuzz"),
		PathExpression: path.MatchRoot("fuzz"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

// FuzzIPOrCIDR feeds arbitrary strings into the IPOrCIDR validator.
// Property: ValidateString never panics regardless of input.
func FuzzIPOrCIDR(f *testing.F) {
	seeds := []string{
		"",
		"192.168.1.1",
		"::1",
		"10.0.0.0/24",
		"not-an-ip",
		"999.999.999.999",
		"2001:db8::/32",
		"/",
		"/32",
		"192.168.1.1/",
		"\x00",
		"abc/def/ghi",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	v := lv.IPOrCIDR()
	f.Fuzz(func(t *testing.T, input string) {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), makeReq(input), resp)
	})
}

// FuzzHostOrIP feeds arbitrary strings into the HostOrIP validator.
// Property: ValidateString never panics regardless of input.
func FuzzHostOrIP(f *testing.F) {
	seeds := []string{
		"",
		"example.com",
		"sub.example.com",
		"localhost",
		"192.168.1.1",
		"::1",
		"-leading-hyphen",
		"trailing-hyphen-",
		".leading.dot",
		"trailing.dot.",
		"bad..label",
		"has space",
		"has_underscore",
		"\x00",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	v := lv.HostOrIP()
	f.Fuzz(func(t *testing.T, input string) {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), makeReq(input), resp)
	})
}

// FuzzZFSPath feeds arbitrary strings into the ZFSPath validator.
// Property: ValidateString never panics regardless of input.
func FuzzZFSPath(f *testing.F) {
	seeds := []string{
		"",
		"tank",
		"tank/dataset",
		"tank/nested/deep",
		"tank/has space",
		"//leading",
		"trailing/",
		"tank//double",
		"tank/bad!chars",
		"tank/unicode\u00e9",
		"\x00",
		"/",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	v := lv.ZFSPath()
	f.Fuzz(func(t *testing.T, input string) {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), makeReq(input), resp)
	})
}

// FuzzCompressionAlgorithm feeds arbitrary strings into the
// CompressionAlgorithm validator. Property: ValidateString never panics
// regardless of input.
func FuzzCompressionAlgorithm(f *testing.F) {
	seeds := []string{
		"",
		"LZ4",
		"lz4",
		"OFF",
		"ZSTD",
		"GZIP-9",
		"invalid",
		"  LZ4  ",
		"\x00",
		"gzip-999",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	v := lv.CompressionAlgorithm()
	f.Fuzz(func(t *testing.T, input string) {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), makeReq(input), resp)
	})
}
