package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccDataSourceLive runs a read-only live test for one data source: the
// config is the provider block plus a single unfiltered data block, and the
// check asserts the list attribute came back with at least minLen entries.
// Nothing is mutated on the box.
func testAccDataSourceLive(t *testing.T, typeName, attribute string, minLen int) {
	t.Helper()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig() + fmt.Sprintf(`
data %q "live" {}
`, typeName),
				Check: testAccCheckListAttrMinLen(fmt.Sprintf("data.%s.live", typeName), attribute, minLen),
			},
		},
	})
}

// TestAccInterfacesDataSourceLive reads the interface inventory. Every pfSense
// box has at least one interface, so an empty result means the read failed to
// decode rather than that the box is genuinely empty.
func TestAccInterfacesDataSourceLive(t *testing.T) {
	testAccDataSourceLive(t, "pfsense_interfaces", "interfaces", 1)
}

// TestAccFirewallAliasesDataSourceLive reads the alias list. A box with no
// aliases configured is valid, so this only asserts the read succeeded and the
// list attribute is present.
func TestAccFirewallAliasesDataSourceLive(t *testing.T) {
	testAccDataSourceLive(t, "pfsense_firewall_aliases", "aliases", 0)
}

// TestAccRoutingGatewaysDataSourceLive reads the gateway list. Gateway count
// depends on the box's routing configuration, so no minimum is asserted.
func TestAccRoutingGatewaysDataSourceLive(t *testing.T) {
	testAccDataSourceLive(t, "pfsense_routing_gateways", "gateways", 0)
}

// TestAccSystemCertificatesDataSourceLive reads the certificate store. Its
// contents vary per box, so no minimum is asserted.
func TestAccSystemCertificatesDataSourceLive(t *testing.T) {
	testAccDataSourceLive(t, "pfsense_system_certificates", "certificates", 0)
}
