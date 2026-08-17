package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live CRUD coverage for the firewall resources (rules, schedules, NAT and the
// traffic shaper family). Everything here runs against a real pfSense box and
// is gated by testAccPreCheck, so `go test ./...` without TF_ACC stays
// hermetic.
//
// Each test owns a fixed `tftest_`-prefixed identity (the traffic shaper is the
// exception: it is keyed on the interface it attaches to), refuses to start
// when that identity is already on the box, and asserts through CheckDestroy
// that the object is gone once the test finishes.

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
// `ipprotocol` and the three booleans are set explicitly because the API
// defaults them and echoes the defaults back on every read. `associated_rule_id`
// is deliberately left unset: the API reports it as null on read, so pinning it
// to the empty string it accepts on write would leave a permanent diff.
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

// ---------------------------------------------------------------------------
// pfsense_firewall_schedule
// ---------------------------------------------------------------------------

const testAccLiveScheduleName = "tftest_live_sched"

func testAccFirewallScheduleExists(name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, firewallSchedulePlural, "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up firewall schedule %q: %w", name, err)
	}
	return found, nil
}

func testAccPreCheckLiveFirewallScheduleAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccFirewallScheduleExists(testAccLiveScheduleName)
	if err != nil {
		t.Fatalf("checking firewall schedule %q does not already exist: %v", testAccLiveScheduleName, err)
	}
	if found {
		t.Fatalf("firewall schedule %q already exists on the box; remove it before running the live test", testAccLiveScheduleName)
	}
}

func testAccCheckFirewallScheduleAbsent(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccFirewallScheduleExists(name)
		if err != nil {
			return fmt.Errorf("checking firewall schedule %q was destroyed: %w", name, err)
		}
		if found {
			return fmt.Errorf("firewall schedule %q still exists after destroy", name)
		}
		return nil
	}
}

func testAccFirewallScheduleLiveConfig(descr, rangeDescr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_schedule" "live" {
  name  = %q
  descr = %q
  timerange = [
    {
      month      = [1]
      day        = [1]
      hour       = "0:00-23:59"
      rangedescr = %q
    }
  ]
}
`, testAccLiveScheduleName, descr, rangeDescr)
}

func TestAccFirewallScheduleResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveFirewallScheduleAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckFirewallScheduleAbsent(testAccLiveScheduleName),
		Steps: []resource.TestStep{
			{
				Config: testAccFirewallScheduleLiveConfig("acceptance test", "new year"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "id", testAccLiveScheduleName),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "name", testAccLiveScheduleName),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "descr", "acceptance test"),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "timerange.#", "1"),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "timerange.0.hour", "0:00-23:59"),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "timerange.0.month.0", "1"),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "timerange.0.day.0", "1"),
				),
			},
			{
				// `name` (the natural key) is untouched; the description and
				// the time range label move.
				Config: testAccFirewallScheduleLiveConfig("acceptance test (updated)", "new year (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "name", testAccLiveScheduleName),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "descr", "acceptance test (updated)"),
					resource.TestCheckResourceAttr("pfsense_firewall_schedule.live", "timerange.0.rangedescr", "new year (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_schedule.live",
				ImportState:       true,
				ImportStateId:     testAccLiveScheduleName,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_virtual_ip
// ---------------------------------------------------------------------------

// The virtual IP is a CARP address in the WAN subnet: `vhid`, `password` and
// `carp_peer` are required by the schema and are only accepted by the API in
// CARP mode. 10.99.0.222 is a spare address, so nothing about the box's own
// management address changes.
const testAccLiveVirtualIPDescr = "tftest_live_vip"

func testAccVirtualIPExists(descr string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, virtualIPsPlural, "descr", descr)
	if err != nil {
		return false, fmt.Errorf("looking up virtual IP %q: %w", descr, err)
	}
	return found, nil
}

func testAccPreCheckLiveVirtualIPAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccVirtualIPExists(testAccLiveVirtualIPDescr)
	if err != nil {
		t.Fatalf("checking virtual IP %q does not already exist: %v", testAccLiveVirtualIPDescr, err)
	}
	if found {
		t.Fatalf("virtual IP %q already exists on the box; remove it before running the live test", testAccLiveVirtualIPDescr)
	}
}

func testAccCheckVirtualIPAbsent(descr string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccVirtualIPExists(descr)
		if err != nil {
			return fmt.Errorf("checking virtual IP %q was destroyed: %w", descr, err)
		}
		if found {
			return fmt.Errorf("virtual IP %q still exists after destroy", descr)
		}
		return nil
	}
}

func testAccVirtualIPLiveConfig(advskew int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_virtual_ip" "live" {
  mode        = "carp"
  interface   = "wan"
  type        = "single"
  subnet      = "10.99.0.222"
  subnet_bits = 24
  descr       = %q
  vhid        = 210
  advbase     = 1
  advskew     = %d
  password    = "tftestcarp"
  carp_mode   = "ucast"
  carp_peer   = "10.99.0.3"
}
`, testAccLiveVirtualIPDescr, advskew)
}

func TestAccFirewallVirtualIPResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveVirtualIPAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckVirtualIPAbsent(testAccLiveVirtualIPDescr),
		Steps: []resource.TestStep{
			{
				Config: testAccVirtualIPLiveConfig(100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "id", testAccLiveVirtualIPDescr),
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "descr", testAccLiveVirtualIPDescr),
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "mode", "carp"),
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "subnet", "10.99.0.222"),
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "advskew", "100"),
				),
			},
			{
				// `descr` (the natural key) is unchanged; only the CARP
				// advertisement skew moves.
				Config: testAccVirtualIPLiveConfig(50),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "descr", testAccLiveVirtualIPDescr),
					resource.TestCheckResourceAttr("pfsense_firewall_virtual_ip.live", "advskew", "50"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_virtual_ip.live",
				ImportState:       true,
				ImportStateId:     testAccLiveVirtualIPDescr,
				ImportStateVerify: true,
				// The API never returns the CARP password, so an imported
				// virtual IP cannot reproduce it.
				ImportStateVerifyIgnore: []string{"password"},
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper
// ---------------------------------------------------------------------------

// A traffic shaper is keyed on the interface it is attached to, so this is the
// one live object that cannot carry a `tftest_` name. It is created disabled
// so ALTQ never actually shapes traffic on the management interface.
const testAccLiveShaperInterface = "wan"

func testAccTrafficShaperExists(iface string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, trafficShapersPlural, "interface", iface)
	if err != nil {
		return false, fmt.Errorf("looking up traffic shaper on %q: %w", iface, err)
	}
	return found, nil
}

func testAccPreCheckLiveTrafficShaperAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccTrafficShaperExists(testAccLiveShaperInterface)
	if err != nil {
		t.Fatalf("checking no traffic shaper exists on %q: %v", testAccLiveShaperInterface, err)
	}
	if found {
		t.Fatalf("a traffic shaper already exists on %q; remove it before running the live test", testAccLiveShaperInterface)
	}
}

func testAccCheckTrafficShaperAbsent(iface string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccTrafficShaperExists(iface)
		if err != nil {
			return fmt.Errorf("checking traffic shaper on %q was destroyed: %w", iface, err)
		}
		if found {
			return fmt.Errorf("traffic shaper on %q still exists after destroy", iface)
		}
		return nil
	}
}

// testAccTrafficShaperLiveConfig renders the shaper config for one step. The
// empty `queue` list is explicit because the API always reports the child queue
// collection, even when it is empty.
func testAccTrafficShaperLiveConfig(bandwidth int) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_traffic_shaper" "live" {
  interface     = %q
  enabled       = false
  scheduler     = "HFSC"
  bandwidthtype = "Mb"
  bandwidth     = %d
  qlimit        = 50
  queue         = []
}
`, testAccLiveShaperInterface, bandwidth)
}

func TestAccFirewallTrafficShaperResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveTrafficShaperAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTrafficShaperAbsent(testAccLiveShaperInterface),
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficShaperLiveConfig(100),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "id", testAccLiveShaperInterface),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "interface", testAccLiveShaperInterface),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "scheduler", "HFSC"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "bandwidth", "100"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "enabled", "false"),
				),
			},
			{
				// `interface` (the natural key) is unchanged; only the
				// bandwidth allowance moves.
				Config: testAccTrafficShaperLiveConfig(200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "interface", testAccLiveShaperInterface),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper.live", "bandwidth", "200"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_traffic_shaper.live",
				ImportState:       true,
				ImportStateId:     testAccLiveShaperInterface,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper_limiter
// ---------------------------------------------------------------------------

const testAccLiveLimiterName = "tftest_limiter"

func testAccTrafficShaperLimiterExists(name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, trafficShaperLimitersPlural, "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up traffic shaper limiter %q: %w", name, err)
	}
	return found, nil
}

func testAccPreCheckLiveTrafficShaperLimiterAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccTrafficShaperLimiterExists(testAccLiveLimiterName)
	if err != nil {
		t.Fatalf("checking traffic shaper limiter %q does not already exist: %v", testAccLiveLimiterName, err)
	}
	if found {
		t.Fatalf("traffic shaper limiter %q already exists on the box; remove it before running the live test", testAccLiveLimiterName)
	}
}

func testAccCheckTrafficShaperLimiterAbsent(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccTrafficShaperLimiterExists(name)
		if err != nil {
			return fmt.Errorf("checking traffic shaper limiter %q was destroyed: %w", name, err)
		}
		if found {
			return fmt.Errorf("traffic shaper limiter %q still exists after destroy", name)
		}
		return nil
	}
}

// testAccTrafficShaperLimiterLiveConfig renders the limiter config for one
// step. `bwsched` defaults to "none" on the box and the queue collection is
// always reported, so both are pinned to keep the plan empty.
func testAccTrafficShaperLimiterLiveConfig(description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_traffic_shaper_limiter" "live" {
  name        = %q
  enabled     = true
  aqm         = "droptail"
  sched       = "fifo"
  mask        = "none"
  description = %q
  bandwidth = [
    {
      bw      = 10
      bwscale = "Mb"
      bwsched = "none"
    }
  ]
  queue = []
}
`, testAccLiveLimiterName, description)
}

func TestAccFirewallTrafficShaperLimiterResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveTrafficShaperLimiterAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTrafficShaperLimiterAbsent(testAccLiveLimiterName),
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficShaperLimiterLiveConfig("acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "id", testAccLiveLimiterName),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "name", testAccLiveLimiterName),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "aqm", "droptail"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "sched", "fifo"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "bandwidth.0.bw", "10"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "description", "acceptance test"),
				),
			},
			{
				// `name` (the natural key) is unchanged; only the description
				// moves.
				Config: testAccTrafficShaperLimiterLiveConfig("acceptance test (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "name", testAccLiveLimiterName),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter.live", "description", "acceptance test (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_traffic_shaper_limiter.live",
				ImportState:       true,
				ImportStateId:     testAccLiveLimiterName,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper_queue
// ---------------------------------------------------------------------------

// Queues are keyed on parent|name, where the parent key is the shaper's
// interface. The parent shaper is created through the API instead of Terraform
// because the shaper resource exposes its own `queue` collection: as soon as
// the queue resource adds a child, a Terraform-managed parent would drift and
// never settle on an empty plan.
const (
	testAccLiveShaperQueueParent = "wan"
	testAccLiveShaperQueueName   = "tftest_shq"
	testAccLiveShaperQueueID     = testAccLiveShaperQueueParent + "|" + testAccLiveShaperQueueName
)

// testAccEnsureLiveShaperParent creates the throwaway parent traffic shaper and
// registers its removal, so the box is left exactly as it was found. The shaper
// is disabled, so ALTQ never shapes traffic on the management interface.
func testAccEnsureLiveShaperParent(t *testing.T) {
	t.Helper()

	c, err := testAccClient()
	if err != nil {
		t.Fatalf("building fixture client: %v", err)
	}
	ctx := context.Background()

	_, _, found, err := findByKey(ctx, c, trafficShapersPlural, "interface", testAccLiveShaperQueueParent)
	if err != nil {
		t.Fatalf("checking parent traffic shaper on %q: %v", testAccLiveShaperQueueParent, err)
	}
	if found {
		t.Fatalf("a traffic shaper already exists on %q; remove it before running the live test", testAccLiveShaperQueueParent)
	}

	_, err = c.Create(ctx, trafficShaperSingular, map[string]any{
		"interface":     testAccLiveShaperQueueParent,
		"enabled":       false,
		"scheduler":     "HFSC",
		"bandwidthtype": "Mb",
		"bandwidth":     100,
		"qlimit":        50,
		"apply":         true,
	})
	if err != nil {
		t.Fatalf("creating parent traffic shaper on %q: %v", testAccLiveShaperQueueParent, err)
	}

	t.Cleanup(func() {
		id, _, found, err := findByKey(ctx, c, trafficShapersPlural, "interface", testAccLiveShaperQueueParent)
		if err != nil {
			t.Errorf("removing parent traffic shaper on %q: %v", testAccLiveShaperQueueParent, err)
			return
		}
		if !found {
			return
		}
		q := client.Query{}.Set("id", formatID(id)).Set("apply", "true")
		if err := c.Delete(ctx, trafficShaperSingular, q); err != nil {
			t.Errorf("removing parent traffic shaper on %q: %v", testAccLiveShaperQueueParent, err)
		}
	})
}

func testAccTrafficShaperQueueExists(parentKey, name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	ctx := context.Background()
	pid, err := resolveParentID(ctx, c, trafficShapersPlural, "interface", parentKey)
	if err != nil {
		// A missing parent means the queue cannot exist either.
		return false, nil
	}
	_, _, found, err := findByKeyInParent(ctx, c, trafficShaperQueuesPlural, formatID(pid), "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up traffic shaper queue %q on %q: %w", name, parentKey, err)
	}
	return found, nil
}

func testAccPreCheckLiveTrafficShaperQueueAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccTrafficShaperQueueExists(testAccLiveShaperQueueParent, testAccLiveShaperQueueName)
	if err != nil {
		t.Fatalf("checking traffic shaper queue %q does not already exist: %v", testAccLiveShaperQueueID, err)
	}
	if found {
		t.Fatalf("traffic shaper queue %q already exists on the box; remove it before running the live test", testAccLiveShaperQueueID)
	}
}

func testAccCheckTrafficShaperQueueAbsent(parentKey, name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccTrafficShaperQueueExists(parentKey, name)
		if err != nil {
			return fmt.Errorf("checking traffic shaper queue %q was destroyed: %w", parentKey+"|"+name, err)
		}
		if found {
			return fmt.Errorf("traffic shaper queue %q still exists after destroy", parentKey+"|"+name)
		}
		return nil
	}
}

// testAccTrafficShaperQueueLiveConfig renders the queue config for one step.
// The three HFSC service curves are enabled because their `*_m2` limits are
// required by the schema and the API only accepts them when the matching
// toggle is on. The four scheduler-option booleans default to false on the box
// and are echoed back on every read, so they are pinned to keep the plan empty.
func testAccTrafficShaperQueueLiveConfig(description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_traffic_shaper_queue" "live" {
  parent_id     = %q
  name          = %q
  enabled       = true
  qlimit        = 50
  bandwidth     = 10
  bandwidthtype = "Mb"
  description   = %q
  default       = true
  red           = false
  rio           = false
  ecn           = false
  codel         = false
  upperlimit    = true
  upperlimit_m2 = "10Mb"
  realtime      = true
  realtime_m2   = "2Mb"
  linkshare     = true
  linkshare_m2  = "10Mb"
}
`, testAccLiveShaperQueueParent, testAccLiveShaperQueueName, description)
}

func TestAccFirewallTrafficShaperQueueResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccEnsureLiveShaperParent(t)
			testAccPreCheckLiveTrafficShaperQueueAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTrafficShaperQueueAbsent(testAccLiveShaperQueueParent, testAccLiveShaperQueueName),
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficShaperQueueLiveConfig("acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "id", testAccLiveShaperQueueID),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "parent_id", testAccLiveShaperQueueParent),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "name", testAccLiveShaperQueueName),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "bandwidth", "10"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "description", "acceptance test"),
				),
			},
			{
				// Both key components (`parent_id` and `name`) are unchanged;
				// only the description moves.
				Config: testAccTrafficShaperQueueLiveConfig("acceptance test (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "id", testAccLiveShaperQueueID),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_queue.live", "description", "acceptance test (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_traffic_shaper_queue.live",
				ImportState:       true,
				ImportStateId:     testAccLiveShaperQueueID,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_firewall_traffic_shaper_limiter_queue
// ---------------------------------------------------------------------------

// Limiter queues are keyed on parent|name, where the parent key is the
// limiter's name. As with the shaper queue, the parent limiter is created
// through the API: the limiter resource owns a `queue` collection that would
// drift the moment this resource adds a child.
const (
	testAccLiveLimiterQueueParent = "tftest_lqparent"
	testAccLiveLimiterQueueName   = "tftest_lq"
	testAccLiveLimiterQueueID     = testAccLiveLimiterQueueParent + "|" + testAccLiveLimiterQueueName
)

// testAccEnsureLiveLimiterParent creates the throwaway parent limiter and
// registers its removal so the box is left clean.
func testAccEnsureLiveLimiterParent(t *testing.T) {
	t.Helper()

	c, err := testAccClient()
	if err != nil {
		t.Fatalf("building fixture client: %v", err)
	}
	ctx := context.Background()

	_, _, found, err := findByKey(ctx, c, trafficShaperLimitersPlural, "name", testAccLiveLimiterQueueParent)
	if err != nil {
		t.Fatalf("checking parent limiter %q: %v", testAccLiveLimiterQueueParent, err)
	}
	if found {
		t.Fatalf("limiter %q already exists on the box; remove it before running the live test", testAccLiveLimiterQueueParent)
	}

	_, err = c.Create(ctx, trafficShaperLimiterSingular, map[string]any{
		"name":    testAccLiveLimiterQueueParent,
		"enabled": true,
		"aqm":     "droptail",
		"sched":   "fifo",
		"mask":    "none",
		"bandwidth": []map[string]any{
			{"bw": 10, "bwscale": "Mb"},
		},
		"apply": true,
	})
	if err != nil {
		t.Fatalf("creating parent limiter %q: %v", testAccLiveLimiterQueueParent, err)
	}

	t.Cleanup(func() {
		id, _, found, err := findByKey(ctx, c, trafficShaperLimitersPlural, "name", testAccLiveLimiterQueueParent)
		if err != nil {
			t.Errorf("removing parent limiter %q: %v", testAccLiveLimiterQueueParent, err)
			return
		}
		if !found {
			return
		}
		q := client.Query{}.Set("id", formatID(id)).Set("apply", "true")
		if err := c.Delete(ctx, trafficShaperLimiterSingular, q); err != nil {
			t.Errorf("removing parent limiter %q: %v", testAccLiveLimiterQueueParent, err)
		}
	})
}

func testAccTrafficShaperLimiterQueueExists(parentKey, name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	ctx := context.Background()
	pid, err := resolveParentID(ctx, c, trafficShaperLimitersPlural, "name", parentKey)
	if err != nil {
		// A missing parent limiter means the queue cannot exist either.
		return false, nil
	}
	_, _, found, err := findByKeyInParent(ctx, c, trafficShaperLimiterQueuesPlural, formatID(pid), "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up limiter queue %q on %q: %w", name, parentKey, err)
	}
	return found, nil
}

func testAccPreCheckLiveTrafficShaperLimiterQueueAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccTrafficShaperLimiterQueueExists(testAccLiveLimiterQueueParent, testAccLiveLimiterQueueName)
	if err != nil {
		t.Fatalf("checking limiter queue %q does not already exist: %v", testAccLiveLimiterQueueID, err)
	}
	if found {
		t.Fatalf("limiter queue %q already exists on the box; remove it before running the live test", testAccLiveLimiterQueueID)
	}
}

func testAccCheckTrafficShaperLimiterQueueAbsent(parentKey, name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccTrafficShaperLimiterQueueExists(parentKey, name)
		if err != nil {
			return fmt.Errorf("checking limiter queue %q was destroyed: %w", parentKey+"|"+name, err)
		}
		if found {
			return fmt.Errorf("limiter queue %q still exists after destroy", parentKey+"|"+name)
		}
		return nil
	}
}

func testAccTrafficShaperLimiterQueueLiveConfig(description string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_firewall_traffic_shaper_limiter_queue" "live" {
  parent_id   = %q
  name        = %q
  enabled     = true
  aqm         = "droptail"
  mask        = "none"
  description = %q
  weight      = 10
}
`, testAccLiveLimiterQueueParent, testAccLiveLimiterQueueName, description)
}

func TestAccFirewallTrafficShaperLimiterQueueResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccEnsureLiveLimiterParent(t)
			testAccPreCheckLiveTrafficShaperLimiterQueueAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckTrafficShaperLimiterQueueAbsent(testAccLiveLimiterQueueParent, testAccLiveLimiterQueueName),
		Steps: []resource.TestStep{
			{
				Config: testAccTrafficShaperLimiterQueueLiveConfig("acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "id", testAccLiveLimiterQueueID),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "parent_id", testAccLiveLimiterQueueParent),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "name", testAccLiveLimiterQueueName),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "weight", "10"),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "description", "acceptance test"),
				),
			},
			{
				// Both key components (`parent_id` and `name`) are unchanged;
				// only the description moves.
				Config: testAccTrafficShaperLimiterQueueLiveConfig("acceptance test (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "id", testAccLiveLimiterQueueID),
					resource.TestCheckResourceAttr("pfsense_firewall_traffic_shaper_limiter_queue.live", "description", "acceptance test (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_firewall_traffic_shaper_limiter_queue.live",
				ImportState:       true,
				ImportStateId:     testAccLiveLimiterQueueID,
				ImportStateVerify: true,
			},
		},
	})
}
