package provider

import (
	"context"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Validators for the value shapes that recur across pfSense schemas. The
// terraform-plugin-framework-validators module has no IP/CIDR/hostname
// validators (those live in the legacy SDKv2 helper/validation package), so
// the address forms below are implemented here against net/netip.

// stringCheck adapts a plain predicate over a known, non-null string into a
// validator.String.
type stringCheck struct {
	desc string
	ok   func(string) bool
}

var _ validator.String = stringCheck{}

func (v stringCheck) Description(context.Context) string         { return v.desc }
func (v stringCheck) MarkdownDescription(context.Context) string { return v.desc }

func (v stringCheck) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if v.ok(value) {
		return
	}
	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Attribute Value",
		fmt.Sprintf("Attribute %s %s, got: %q", req.Path, v.desc, value),
	)
}

// isIPAddress accepts a bare IPv4 or IPv6 address (no prefix length).
func isIPAddress() validator.String {
	return stringCheck{
		desc: "must be a valid IP address",
		ok: func(s string) bool {
			_, err := netip.ParseAddr(s)
			return err == nil
		},
	}
}

// isCIDR accepts an IPv4 or IPv6 network in prefix notation. Host bits are
// permitted, matching what pfSense itself accepts for tunnel networks.
func isCIDR() validator.String {
	return stringCheck{
		desc: "must be a valid CIDR network (e.g. `10.0.0.0/24`)",
		ok: func(s string) bool {
			_, err := netip.ParsePrefix(s)
			return err == nil
		},
	}
}

// Deliberately lenient: pfSense accepts short names as well as FQDNs, and a
// bare IPv4 address also satisfies this shape.
var hostnameRe = regexp.MustCompile(`^[A-Za-z0-9_]([A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?(\.[A-Za-z0-9_]([A-Za-z0-9_-]{0,61}[A-Za-z0-9_])?)*\.?$`)

func isHostname() validator.String {
	return stringCheck{
		desc: "must be a valid hostname",
		ok:   hostnameRe.MatchString,
	}
}

// isIPAddressOrHostname covers endpoint attributes documented as accepting
// either form.
func isIPAddressOrHostname() validator.String {
	return stringvalidator.Any(isIPAddress(), isHostname())
}

// isIPAddressOrDynamic accepts an IP address or the literal "dynamic" that
// pfSense uses for gateways reachable via DHCP or PPP.
func isIPAddressOrDynamic() validator.String {
	return stringvalidator.Any(isIPAddress(), stringvalidator.OneOf("dynamic"))
}

var (
	portishRe = regexp.MustCompile(`^[0-9:]+$`)
	macRe     = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$`)
	dateRe    = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// isPort matches a single port number (1-65535) carried as a string.
func isPort() validator.String {
	return stringCheck{
		desc: "must be a port number (1-65535)",
		ok: func(s string) bool {
			n, err := strconv.Atoi(s)
			return err == nil && n >= 1 && n <= 65535
		},
	}
}

// isPortOrRange accepts a port (1-65535), a `start:end` port range, or a named
// port alias. pfSense lets port fields hold alias names as well as numbers, so
// any non-numeric value is accepted as an alias while numeric values are
// bounded to the valid port range.
func isPortOrRange() validator.String {
	return stringCheck{
		desc: "must be a port (1-65535), a port range, or a port alias",
		ok: func(s string) bool {
			if s == "" {
				return false
			}
			if !portishRe.MatchString(s) {
				return true // non-numeric: a port alias
			}
			parts := strings.Split(s, ":")
			if len(parts) > 2 {
				return false
			}
			for _, p := range parts {
				n, err := strconv.Atoi(p)
				if err != nil || n < 1 || n > 65535 {
					return false
				}
			}
			return true
		},
	}
}

// isMAC matches a MAC address delimited with either `:` or `-`.
func isMAC() validator.String {
	return stringvalidator.RegexMatches(macRe, "must be a MAC address")
}

// isDate matches a YYYY-MM-DD calendar date.
func isDate() validator.String {
	return stringvalidator.RegexMatches(dateRe, "must be a date in YYYY-MM-DD format")
}

// isPortNumber is the Int64 counterpart of isPort.
func isPortNumber() validator.Int64 {
	return int64validator.Between(1, 65535)
}

// isSubnetBits bounds an IPv4/IPv6 prefix length.
func isSubnetBits(max int64) validator.Int64 {
	return int64validator.Between(0, max)
}
