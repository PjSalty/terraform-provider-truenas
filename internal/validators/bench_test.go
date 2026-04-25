package validators_test

// Benchmarks for the hot validators — IPOrCIDR and ZFSPath are run at
// plan-time for every resource that uses them, so their per-call cost
// matters for large configurations. Validation failures aren't measured;
// only the happy path, which is what real plans exercise most often.
//
// Run with:
//
//	go test -run='^$' -bench=. -benchtime=5s ./internal/validators/...

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

// benchReq constructs a plan-time StringRequest for the given value.
func benchReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("bench"),
		PathExpression: path.MatchRoot("bench"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

// BenchmarkIPOrCIDR_Valid measures the validator on a valid CIDR block,
// which is the common case in firewall/network configs.
func BenchmarkIPOrCIDR_Valid(b *testing.B) {
	v := lv.IPOrCIDR()
	req := benchReq("10.10.20.0/24")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
	}
}

// BenchmarkZFSPath_Valid measures the validator on a typical multi-level
// dataset path, which is the common case for pool/dataset resources.
func BenchmarkZFSPath_Valid(b *testing.B) {
	v := lv.ZFSPath()
	req := benchReq("tank/apps/postgres/data")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &validator.StringResponse{}
		v.ValidateString(ctx, req, resp)
	}
}
