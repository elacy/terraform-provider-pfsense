package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live CRUD coverage for the firewall resources (rules, schedules, NAT and the
// traffic shaper family). Everything here runs against a real pfSense box and
// is gated by testAccPreCheck, so `go test ./...` without TF_ACC stays
// hermetic.
//
// Each test owns a fixed `tftest_`-prefixed identity, refuses to start when
// that identity is already on the box, and asserts through CheckDestroy that
// the object is gone once the test finishes.

// ---------------------------------------------------------------------------
// pfsense_firewall_rule
// ---------------------------------------------------------------------------

// testAccLiveRuleDescr is the natural key (and Terraform ID) of the throwaway
// firewall rule. The rule itself only matches documentation-range addresses, so
// it can never affect management traffic to the box.
const testAccLiveRuleDescr = "tftest_live_rule"

func testAccFirewallRuleExists(descr string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, firewallRulePlural, "descr", descr)
	if err != nil {
		return false, fmt.Errorf("looking up firewall rule %q: %w", descr, err)
	}
	return found, nil
}

func testAccPreCheckLiveFirewallRuleAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccFirewallRuleExists(testAccLiveRuleDescr)
	if err != nil {
		t.Fatalf("checking firewall rule %q does not already exist: %v", testAccLiveRuleDescr, err)
	}
	if found {
		t.Fatalf("firewall rule %q already exists on the box; remove it before running the live test", testAccLiveRuleDescr)
	}
}

func testAccCheckFirewallRuleAbsent(descr string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccFirewallRuleExists(descr)
		if err != nil {
			return fmt.Errorf("checking firewall rule %q was destroyed: %w", descr, err)
		}
		if found {
			return fmt.Errorf("firewall rule %q still exists after destroy", descr)
		}
		return nil
	}
}

// testAccFirewallRuleLiveConfig renders the rule config for one step. Every
// field the API echoes back (`disabled`, `log`, `tag`, `statetype`,
// `tcp_flags_any` and the two TCP flag lists) is set explicitly so the
// post-apply plan stays empty; `icmptype` is only echoed for ICMP rules, which
// is why this rule matches ICMP rather than TCP.
func testAccFirewallRuleLiveConfig(source string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_rule" "live" {
  type             = "pass"
  interface        = ["wan"]
  ipprotocol       = "inet"
  protocol         = "icmp"
  icmptype         = ["echoreq"]
  source           = %q
  destination      = "198.51.100.10"
  descr            = %q
  disabled         = false
  log              = false
  tag              = ""
  statetype        = "keep state"
  tcp_flags_any    = false
  tcp_flags_out_of = []
  tcp_flags_set    = []
}
`, source, testAccLiveRuleDescr)
}

func TestAccFirewallRuleResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveFirewallRuleAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFirewallRuleAbsent(testAccLiveRuleDescr),
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallRuleLiveConfig("192.0.2.0/24"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "id", testAccLiveRuleDescr),
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "descr", testAccLiveRuleDescr),
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "type", "pass"),
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "interface.#", "1"),
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "interface.0", "wan"),
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "source", "192.0.2.0/24"),
				),
			},
			{
				// The natural key (`descr`) is unchanged; only the source
				// network moves, so this is an in-place update rather than a
				// replacement.
				Config: testAccFirewallRuleLiveConfig("192.0.2.128/25"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "descr", testAccLiveRuleDescr),
					resource.TestCheckResourceAttr("pfsense_firewall_rule.live", "source", "192.0.2.128/25"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_rule.live",
				ImportState:       true,
				ImportStateId:     testAccLiveRuleDescr,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_nat_port_forward
// ---------------------------------------------------------------------------

// Port forwards have no unique description in the API; the resource keys them
// on interface|protocol|destination|target, which is also the Terraform ID and
// the import ID.
const (
	testAccLivePortForwardIface       = "wan"
	testAccLivePortForwardProtocol    = "tcp"
	testAccLivePortForwardDestination = "198.51.100.20"
	testAccLivePortForwardTarget      = "10.99.0.50"
	testAccLivePortForwardID          = testAccLivePortForwardIface + "|" + testAccLivePortForwardProtocol + "|" +
		testAccLivePortForwardDestination + "|" + testAccLivePortForwardTarget
	testAccLivePortForwardDescr = "tftest_live_port_forward"
)

func testAccNATPortForwardExists() (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKeys(context.Background(), c, firewallNATPortForwardPlural, map[string]string{
		"interface":   testAccLivePortForwardIface,
		"protocol":    testAccLivePortForwardProtocol,
		"destination": testAccLivePortForwardDestination,
		"target":      testAccLivePortForwardTarget,
	})
	if err != nil {
		return false, fmt.Errorf("looking up NAT port forward %q: %w", testAccLivePortForwardID, err)
	}
	return found, nil
}

func testAccPreCheckLiveNATPortForwardAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccNATPortForwardExists()
	if err != nil {
		t.Fatalf("checking NAT port forward %q does not already exist: %v", testAccLivePortForwardID, err)
	}
	if found {
		t.Fatalf("NAT port forward %q already exists on the box; remove it before running the live test", testAccLivePortForwardID)
	}
}

func testAccCheckNATPortForwardAbsent() resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccNATPortForwardExists()
		if err != nil {
			return fmt.Errorf("checking NAT port forward %q was destroyed: %w", testAccLivePortForwardID, err)
		}
		if found {
			return fmt.Errorf("NAT port forward %q still exists after destroy", testAccLivePortForwardID)
		}
		return nil
	}
}

// testAccNATPortForwardLiveConfig renders the port forward config for one step.
// `ipprotocol`, the three booleans and `associated_rule_id` are set explicitly
// because the API defaults them and echoes the defaults back on every read.
func testAccNATPortForwardLiveConfig(localPort, descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_nat_port_forward" "live" {
  interface          = %q
  ipprotocol         = "inet"
  protocol           = %q
  source             = "any"
  destination        = %q
  destination_port   = "8080"
  target             = %q
  local_port         = %q
  descr              = %q
  disabled           = false
  nordr              = false
  nosync             = false
  associated_rule_id = ""
}
`, testAccLivePortForwardIface, testAccLivePortForwardProtocol, testAccLivePortForwardDestination,
		testAccLivePortForwardTarget, localPort, descr)
}

func TestAccFirewallNATPortForwardResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveNATPortForwardAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNATPortForwardAbsent(),
		Steps: []resource.TestStep{
			{
				Config: testAccNATPortForwardLiveConfig("8080", testAccLivePortForwardDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "id", testAccLivePortForwardID),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "interface", testAccLivePortForwardIface),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "destination", testAccLivePortForwardDestination),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "target", testAccLivePortForwardTarget),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "local_port", "8080"),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "descr", testAccLivePortForwardDescr),
				),
			},
			{
				// Every key component (interface, protocol, destination,
				// target) is identical; only the forwarded port and the
				// description change.
				Config: testAccNATPortForwardLiveConfig("9090", testAccLivePortForwardDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "id", testAccLivePortForwardID),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "local_port", "9090"),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_port_forward.live", "descr", testAccLivePortForwardDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_nat_port_forward.live",
				ImportState:       true,
				ImportStateId:     testAccLivePortForwardID,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_nat_one_to_one
// ---------------------------------------------------------------------------

// 1:1 mappings are keyed on interface|external, which doubles as the Terraform
// ID and the import ID.
const (
	testAccLiveOneToOneIface    = "wan"
	testAccLiveOneToOneExternal = "198.51.100.30"
	testAccLiveOneToOneID       = testAccLiveOneToOneIface + "|" + testAccLiveOneToOneExternal
	testAccLiveOneToOneDescr    = "tftest_live_one_to_one"
)

func testAccNATOneToOneExists() (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKeys(context.Background(), c, firewallNATOneToOnePlural, map[string]string{
		"interface": testAccLiveOneToOneIface,
		"external":  testAccLiveOneToOneExternal,
	})
	if err != nil {
		return false, fmt.Errorf("looking up 1:1 NAT mapping %q: %w", testAccLiveOneToOneID, err)
	}
	return found, nil
}

func testAccPreCheckLiveNATOneToOneAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccNATOneToOneExists()
	if err != nil {
		t.Fatalf("checking 1:1 NAT mapping %q does not already exist: %v", testAccLiveOneToOneID, err)
	}
	if found {
		t.Fatalf("1:1 NAT mapping %q already exists on the box; remove it before running the live test", testAccLiveOneToOneID)
	}
}

func testAccCheckNATOneToOneAbsent() resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccNATOneToOneExists()
		if err != nil {
			return fmt.Errorf("checking 1:1 NAT mapping %q was destroyed: %w", testAccLiveOneToOneID, err)
		}
		if found {
			return fmt.Errorf("1:1 NAT mapping %q still exists after destroy", testAccLiveOneToOneID)
		}
		return nil
	}
}

// testAccNATOneToOneLiveConfig renders the 1:1 mapping config for one step.
// `ipprotocol`, `disabled` and `nobinat` are API defaults that are echoed back
// on every read, so they are pinned here to keep the plan empty.
func testAccNATOneToOneLiveConfig(descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_nat_one_to_one" "live" {
  interface   = %q
  ipprotocol  = "inet"
  external    = %q
  source      = "10.99.0.60"
  destination = "198.51.100.40"
  descr       = %q
  disabled    = false
  nobinat     = false
}
`, testAccLiveOneToOneIface, testAccLiveOneToOneExternal, descr)
}

func TestAccFirewallNATOneToOneResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveNATOneToOneAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNATOneToOneAbsent(),
		Steps: []resource.TestStep{
			{
				Config: testAccNATOneToOneLiveConfig(testAccLiveOneToOneDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "id", testAccLiveOneToOneID),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "interface", testAccLiveOneToOneIface),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "external", testAccLiveOneToOneExternal),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "source", "10.99.0.60"),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "descr", testAccLiveOneToOneDescr),
				),
			},
			{
				// Both key components stay put; only the description changes.
				Config: testAccNATOneToOneLiveConfig(testAccLiveOneToOneDescr + " (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "id", testAccLiveOneToOneID),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_one_to_one.live", "descr", testAccLiveOneToOneDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_nat_one_to_one.live",
				ImportState:       true,
				ImportStateId:     testAccLiveOneToOneID,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_nat_outbound
// ---------------------------------------------------------------------------

// Outbound mappings are keyed on interface|protocol|source|destination|target.
const (
	testAccLiveOutboundIface       = "wan"
	testAccLiveOutboundProtocol    = "tcp"
	testAccLiveOutboundSource      = "192.0.2.0/24"
	testAccLiveOutboundDestination = "198.51.100.0/24"
	testAccLiveOutboundTarget      = "10.99.0.70"
	testAccLiveOutboundID          = testAccLiveOutboundIface + "|" + testAccLiveOutboundProtocol + "|" +
		testAccLiveOutboundSource + "|" + testAccLiveOutboundDestination + "|" + testAccLiveOutboundTarget
	testAccLiveOutboundDescr = "tftest_live_outbound"
)

func testAccNATOutboundExists() (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKeys(context.Background(), c, firewallNATOutboundPlural, map[string]string{
		"interface":   testAccLiveOutboundIface,
		"protocol":    testAccLiveOutboundProtocol,
		"source":      testAccLiveOutboundSource,
		"destination": testAccLiveOutboundDestination,
		"target":      testAccLiveOutboundTarget,
	})
	if err != nil {
		return false, fmt.Errorf("looking up outbound NAT mapping %q: %w", testAccLiveOutboundID, err)
	}
	return found, nil
}

func testAccPreCheckLiveNATOutboundAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccNATOutboundExists()
	if err != nil {
		t.Fatalf("checking outbound NAT mapping %q does not already exist: %v", testAccLiveOutboundID, err)
	}
	if found {
		t.Fatalf("outbound NAT mapping %q already exists on the box; remove it before running the live test", testAccLiveOutboundID)
	}
}

func testAccCheckNATOutboundAbsent() resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccNATOutboundExists()
		if err != nil {
			return fmt.Errorf("checking outbound NAT mapping %q was destroyed: %w", testAccLiveOutboundID, err)
		}
		if found {
			return fmt.Errorf("outbound NAT mapping %q still exists after destroy", testAccLiveOutboundID)
		}
		return nil
	}
}

// testAccNATOutboundLiveConfig renders the outbound mapping config for one
// step. The three booleans, `target_subnet` and `nat_port` (which the API
// stores as an empty string) are pinned because they come back on every read.
func testAccNATOutboundLiveConfig(descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_nat_outbound" "live" {
  interface       = %q
  protocol        = %q
  source          = %q
  destination     = %q
  target          = %q
  target_subnet   = 32
  nat_port        = ""
  descr           = %q
  disabled        = false
  nonat           = false
  nosync          = false
  static_nat_port = false
}
`, testAccLiveOutboundIface, testAccLiveOutboundProtocol, testAccLiveOutboundSource,
		testAccLiveOutboundDestination, testAccLiveOutboundTarget, descr)
}

func TestAccFirewallNATOutboundResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveNATOutboundAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckNATOutboundAbsent(),
		Steps: []resource.TestStep{
			{
				Config: testAccNATOutboundLiveConfig(testAccLiveOutboundDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "id", testAccLiveOutboundID),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "interface", testAccLiveOutboundIface),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "source", testAccLiveOutboundSource),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "target", testAccLiveOutboundTarget),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "descr", testAccLiveOutboundDescr),
				),
			},
			{
				// All five key components are identical; only the description
				// changes.
				Config: testAccNATOutboundLiveConfig(testAccLiveOutboundDescr + " (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "id", testAccLiveOutboundID),
					resource.TestCheckResourceAttr("pfsense_firewall_nat_outbound.live", "descr", testAccLiveOutboundDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_nat_outbound.live",
				ImportState:       true,
				ImportStateId:     testAccLiveOutboundID,
				ImportStateVerify: true,
			},
		},
	})
}
