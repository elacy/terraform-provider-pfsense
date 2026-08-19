package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestResourceSchemasAreValid runs the framework's own schema implementation
// checks over every registered resource, catching malformed attributes and
// misapplied validators at build time rather than at plan time.
func TestResourceSchemasAreValid(t *testing.T) {
	ctx := context.Background()
	p := New("test")().(*pfsenseProvider)

	for _, fn := range p.Resources(ctx) {
		r := fn()

		metaResp := &resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "pfsense"}, metaResp)

		resp := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("%s: schema returned diagnostics: %v", metaResp.TypeName, resp.Diagnostics)
			continue
		}
		if diags := resp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: invalid schema implementation: %v", metaResp.TypeName, diags)
		}
	}
}

func acceptsString(v validator.String, s string) bool {
	resp := &validator.StringResponse{}
	v.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("test"),
		ConfigValue: types.StringValue(s),
	}, resp)
	return !resp.Diagnostics.HasError()
}

func TestStringValidators(t *testing.T) {
	tests := []struct {
		name  string
		v     validator.String
		value string
		want  bool
	}{
		{"ipv4", isIPAddress(), "192.168.1.1", true},
		{"ipv6", isIPAddress(), "2001:db8::1", true},
		{"ip octet out of range", isIPAddress(), "192.168.1.256", false},
		{"ip rejects prefix", isIPAddress(), "10.0.0.0/24", false},
		{"ip rejects empty", isIPAddress(), "", false},

		{"cidr v4", isCIDR(), "10.0.0.0/24", true},
		{"cidr v6", isCIDR(), "2001:db8::/64", true},
		{"cidr requires prefix", isCIDR(), "10.0.0.0", false},
		{"cidr prefix out of range", isCIDR(), "10.0.0.0/99", false},

		{"fqdn", isHostname(), "vpn.example.com", true},
		{"short hostname", isHostname(), "firewall", true},
		{"hostname rejects space", isHostname(), "bad host", false},

		{"ip or hostname takes ip", isIPAddressOrHostname(), "10.0.0.1", true},
		{"ip or hostname takes hostname", isIPAddressOrHostname(), "vpn.example.com", true},
		{"ip or hostname rejects junk", isIPAddressOrHostname(), "not a host!", false},

		{"ip or dynamic takes ip", isIPAddressOrDynamic(), "10.0.0.1", true},
		{"ip or dynamic takes dynamic", isIPAddressOrDynamic(), "dynamic", true},
		{"ip or dynamic rejects junk", isIPAddressOrDynamic(), "not-an-ip", false},

		{"port", isPort(), "1194", true},
		{"port rejects name", isPort(), "http", false},
		{"port rejects out of range", isPort(), "99999", false},

		{"port range", isPortOrRange(), "8000:9000", true},
		{"single port is a range", isPortOrRange(), "80", true},
		{"port range accepts alias", isPortOrRange(), "web-alias", true},
		{"port range rejects out of range", isPortOrRange(), "8000:99999", false},
		{"port range rejects triple", isPortOrRange(), "80:90:100", false},
		{"port range rejects empty", isPortOrRange(), "", false},

		{"mac colon delimited", isMAC(), "aa:bb:cc:dd:ee:ff", true},
		{"mac dash delimited", isMAC(), "AA-BB-CC-DD-EE-FF", true},
		{"mac too short", isMAC(), "aa:bb:cc:dd:ee", false},

		{"date", isDate(), "2026-08-19", true},
		{"date wrong format", isDate(), "19/08/2026", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := acceptsString(tt.v, tt.value); got != tt.want {
				t.Errorf("%q accepted = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// Null and unknown values must pass every validator: they are validated when
// they become known, and rejecting them breaks computed/optional attributes.
func TestStringValidatorsSkipNullAndUnknown(t *testing.T) {
	validators := map[string]validator.String{
		"isIPAddress":           isIPAddress(),
		"isCIDR":                isCIDR(),
		"isHostname":            isHostname(),
		"isIPAddressOrHostname": isIPAddressOrHostname(),
		"isIPAddressOrDynamic":  isIPAddressOrDynamic(),
		"isPort":                isPort(),
		"isPortOrRange":         isPortOrRange(),
		"isMAC":                 isMAC(),
		"isDate":                isDate(),
	}
	values := map[string]types.String{
		"null":    types.StringNull(),
		"unknown": types.StringUnknown(),
	}

	for name, v := range validators {
		for kind, value := range values {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("test"),
				ConfigValue: value,
			}, resp)
			if resp.Diagnostics.HasError() {
				t.Errorf("%s rejected a %s value: %v", name, kind, resp.Diagnostics)
			}
		}
	}
}
