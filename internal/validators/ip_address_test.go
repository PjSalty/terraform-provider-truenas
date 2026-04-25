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

func ipReq(val string) validator.StringRequest {
	return validator.StringRequest{
		Path:           path.Root("ip"),
		PathExpression: path.MatchRoot("ip"),
		ConfigValue:    types.StringValue(val),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, val)},
	}
}

func TestIPOrCIDR_ValidIPs(t *testing.T) {
	cases := []string{
		"192.168.1.1",
		"10.0.0.1",
		"::1",
		"2001:db8::1",
	}
	v := lv.IPOrCIDR()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), ipReq(tc), resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("IPOrCIDR(%q) got unexpected error: %v", tc, resp.Diagnostics)
		}
	}
}

func TestIPOrCIDR_ValidCIDRs(t *testing.T) {
	cases := []string{
		"192.168.1.0/24",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"2001:db8::/32",
	}
	v := lv.IPOrCIDR()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), ipReq(tc), resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("IPOrCIDR(%q) got unexpected error: %v", tc, resp.Diagnostics)
		}
	}
}

func TestIPOrCIDR_Invalid(t *testing.T) {
	cases := []string{
		"not-an-ip",
		"999.999.999.999",
		"192.168.1.0/33",
		"192.168.1.0/abc",
	}
	v := lv.IPOrCIDR()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), ipReq(tc), resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("IPOrCIDR(%q) expected error, got none", tc)
		}
	}
}

func TestIPOrCIDR_EmptyAccepted(t *testing.T) {
	v := lv.IPOrCIDR()
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), ipReq(""), resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("empty string should be accepted, got error: %v", resp.Diagnostics)
	}
}

func TestHostOrIP_ValidHosts(t *testing.T) {
	cases := []string{
		"myhost",
		"myhost.example.com",
		"truenas.local",
		"192.168.1.5",
	}
	v := lv.HostOrIP()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), ipReq(tc), resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("HostOrIP(%q) got unexpected error: %v", tc, resp.Diagnostics)
		}
	}
}

func TestHostOrIP_Invalid(t *testing.T) {
	cases := []string{
		"host with space",
		"host_underscore.example.com",
		"bad!host",
		"foo..bar",  // empty label between dots
		".leading",  // leading dot → empty first label
		"trailing.", // trailing dot → empty last label
	}
	v := lv.HostOrIP()
	for _, tc := range cases {
		resp := &validator.StringResponse{}
		v.ValidateString(context.Background(), ipReq(tc), resp)
		if !resp.Diagnostics.HasError() {
			t.Errorf("HostOrIP(%q) expected error, got none", tc)
		}
	}
}

func TestIPOrCIDR_NullUnknownSkipped(t *testing.T) {
	v := lv.IPOrCIDR()
	// Null
	respNull := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("ip"),
		PathExpression: path.MatchRoot("ip"),
		ConfigValue:    types.StringNull(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, nil)},
	}, respNull)
	if respNull.Diagnostics.HasError() {
		t.Errorf("null value should be skipped, got: %v", respNull.Diagnostics)
	}
	// Unknown
	respUnk := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("ip"),
		PathExpression: path.MatchRoot("ip"),
		ConfigValue:    types.StringUnknown(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
	}, respUnk)
	if respUnk.Diagnostics.HasError() {
		t.Errorf("unknown value should be skipped, got: %v", respUnk.Diagnostics)
	}
}

func TestHostOrIP_NullUnknownEmptySkipped(t *testing.T) {
	v := lv.HostOrIP()
	// Null
	respNull := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("ip"),
		PathExpression: path.MatchRoot("ip"),
		ConfigValue:    types.StringNull(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, nil)},
	}, respNull)
	if respNull.Diagnostics.HasError() {
		t.Errorf("null value should be skipped, got: %v", respNull.Diagnostics)
	}
	// Unknown
	respUnk := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:           path.Root("ip"),
		PathExpression: path.MatchRoot("ip"),
		ConfigValue:    types.StringUnknown(),
		Config:         tfsdk.Config{Raw: tftypes.NewValue(tftypes.String, tftypes.UnknownValue)},
	}, respUnk)
	if respUnk.Diagnostics.HasError() {
		t.Errorf("unknown value should be skipped, got: %v", respUnk.Diagnostics)
	}
	// Empty string
	respEmpty := &validator.StringResponse{}
	v.ValidateString(context.Background(), ipReq(""), respEmpty)
	if respEmpty.Diagnostics.HasError() {
		t.Errorf("empty string should be accepted, got: %v", respEmpty.Diagnostics)
	}
}

func TestIPOrCIDR_Descriptions(t *testing.T) {
	v := lv.IPOrCIDR()
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

func TestHostOrIP_Descriptions(t *testing.T) {
	v := lv.HostOrIP()
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
