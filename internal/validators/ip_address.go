package validators

import (
	"context"
	"net"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// ipOrCIDRValidator validates that a string is a valid IPv4/IPv6 address
// or CIDR block (e.g., "192.168.1.0/24").
type ipOrCIDRValidator struct{}

// IPOrCIDR returns a validator.String that accepts bare IP addresses and
// CIDR notation.  An empty string is accepted (some fields treat it as
// "all hosts").
func IPOrCIDR() validator.String {
	return ipOrCIDRValidator{}
}

func (v ipOrCIDRValidator) Description(_ context.Context) string {
	return "value must be a valid IP address or CIDR block (e.g., 192.168.1.10 or 10.0.0.0/24)"
}

func (v ipOrCIDRValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v ipOrCIDRValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.TrimSpace(req.ConfigValue.ValueString())
	if val == "" {
		return
	}

	if strings.Contains(val, "/") {
		// CIDR notation
		_, _, err := net.ParseCIDR(val)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid CIDR",
				"Expected a valid CIDR block (e.g., 10.0.0.0/24), got: "+val,
			)
		}
		return
	}

	// Bare IP address
	if net.ParseIP(val) == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IP Address",
			"Expected a valid IP address (e.g., 192.168.1.10), got: "+val,
		)
	}
}

// hostOrIPValidator validates that a string is a valid hostname or IP address.
type hostOrIPValidator struct{}

// HostOrIP returns a validator.String that accepts hostnames and IP addresses.
func HostOrIP() validator.String {
	return hostOrIPValidator{}
}

func (v hostOrIPValidator) Description(_ context.Context) string {
	return "value must be a valid hostname or IP address"
}

func (v hostOrIPValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v hostOrIPValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	val := strings.TrimSpace(req.ConfigValue.ValueString())
	if val == "" {
		return
	}

	// If it looks like an IP, validate as IP.
	if net.ParseIP(val) != nil {
		return
	}

	// Otherwise validate as hostname: labels must be non-empty, alphanumeric + hyphens.
	labels := strings.Split(val, ".")
	for _, label := range labels {
		if label == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Hostname",
				"Hostname labels must not be empty: "+val,
			)
			return
		}
		for _, ch := range label {
			if !isAlphaNum(ch) && ch != '-' {
				resp.Diagnostics.AddAttributeError(
					req.Path,
					"Invalid Hostname",
					"Hostname contains invalid character in: "+val,
				)
				return
			}
		}
	}
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}
