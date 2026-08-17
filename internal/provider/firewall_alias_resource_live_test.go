package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testAccLiveAliasName is the throwaway alias the live CRUD test creates. The
// tftest_ prefix keeps it distinguishable from real configuration on the box.
const testAccLiveAliasName = "tftest_live_alias"

// testAccPreCheckLiveAliasAbsent refuses to start when the throwaway alias is
// already on the box. The name is fixed, so a leftover from an aborted run (or
// a second run in parallel) would otherwise be adopted and then destroyed by
// this test — better to stop and let a human clean up.
func testAccPreCheckLiveAliasAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccFirewallAliasExists(testAccLiveAliasName)
	if err != nil {
		t.Fatalf("checking firewall alias %q does not already exist: %v", testAccLiveAliasName, err)
	}
	if found {
		t.Fatalf("firewall alias %q already exists on the box; remove it before running the live test", testAccLiveAliasName)
	}
}

// testAccFirewallAliasLiveConfig renders the alias config for one step. `detail`
// is set explicitly because the API always echoes the per-entry detail list
// back, so leaving it unmanaged would produce a permanent diff.
func testAccFirewallAliasLiveConfig(descr, addresses, detail string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_alias" "live" {
  name    = %q
  type    = "host"
  descr   = %q
  address = %s
  detail  = %s
}
`, testAccLiveAliasName, descr, addresses, detail)
}

// TestAccFirewallAliasResourceLive exercises create, update, import and destroy
// of a firewall alias against a real pfSense instance.
func TestAccFirewallAliasResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveAliasAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFirewallAliasAbsent(testAccLiveAliasName),
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallAliasLiveConfig(
					"acceptance test",
					`["10.0.0.10", "10.0.0.11"]`,
					`["first host", "second host"]`,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "id", testAccLiveAliasName),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "name", testAccLiveAliasName),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "type", "host"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "descr", "acceptance test"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "address.#", "2"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "address.0", "10.0.0.10"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "address.1", "10.0.0.11"),
				),
			},
			{
				Config: testAccFirewallAliasLiveConfig(
					"acceptance test (updated)",
					`["10.0.0.20"]`,
					`["only host"]`,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "descr", "acceptance test (updated)"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "address.#", "1"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "address.0", "10.0.0.20"),
					resource.TestCheckResourceAttr("pfsense_firewall_alias.live", "detail.0", "only host"),
				),
			},
			{
				// Aliases are imported by name — their natural key doubles as
				// the Terraform ID.
				ResourceName:      "pfsense_firewall_alias.live",
				ImportState:       true,
				ImportStateId:     testAccLiveAliasName,
				ImportStateVerify: true,
			},
		},
	})
}
