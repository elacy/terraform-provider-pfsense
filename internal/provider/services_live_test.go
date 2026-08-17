package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live CRUD coverage for the services family: the DHCP server and its three
// child collections, the DNS resolver and DNS forwarder overrides (and their
// aliases), the NTP settings singleton and its time servers, cron jobs, the
// Service Watchdog, and the BIND and FreeRADIUS package models. Everything here
// runs against a real pfSense box and is gated by testAccPreCheck, so
// `go test ./...` without TF_ACC stays hermetic.
//
// SAFETY. This batch configures the services the box runs on, so the shape of
// each change matters:
//
//   - DHCP server: the reference box is a single-NIC VM whose only interface,
//     `wan`, carries the 10.99.0.0/24 management network these tests reach it
//     over. A DHCP server handing out leases there could collide with the
//     management addressing, so the tests never enable one: every step pins
//     `enable = false`, which writes the configuration without starting dhcpd
//     for the interface (the box's service inventory has no dhcpd entry before
//     or after the run). The address range is 10.99.0.200-.210, outside any
//     address in use, and is never served. The three child resources need a
//     parent DHCP server to exist, and each brings its own disabled one.
//   - DNS resolver/forwarder: only throwaway host, alias and domain overrides
//     for documentation-range addresses (192.0.2.0/24, RFC 5737) under
//     `example.com` are created, and each is destroyed by its test. The
//     resolver's and the forwarder's own service-enabled state is never
//     touched, so the box keeps resolving exactly as it did before.
//   - NTP settings: a singleton, so it has no create/destroy semantics. The
//     test captures what the box holds, applies a reversible change to the
//     orphan-mode stratum (a fallback used only when no time source is
//     reachable), restores the captured value in a later step, and asserts over
//     the API that it stuck. The guard is idempotent: a box found on the test
//     value is a leftover from a run that died mid-test and is rolled back
//     before this one starts. `enable` is pinned to the box's own value in
//     every step, so ntpd is never started or stopped.
//   - NTP time server: a documentation-range address is added alongside the
//     box's existing time server, which is neither modified nor removed.
//   - Cron job: `/usr/bin/true`, which does nothing whether or not it fires
//     between create and destroy.
//
// Six of the eighteen resources in this batch are backed by pfSense packages
// that are not installed on the reference box; see the skip tests at the bottom
// of this file for what each one reported.

// ---------------------------------------------------------------------------
// Shared helpers
//
// The natural-key helpers (testAccLiveObjectExists and friends) live in
// system_live_test.go; the ones below cover the parent-scoped objects this
// batch adds.
// ---------------------------------------------------------------------------

// testAccLiveChild identifies one parent-scoped object on the box: how to find
// its parent (the parent's plural endpoint plus the fields its natural key is
// made of) and how to find the child inside it. It resolves both exactly the way
// the child resources do.
type testAccLiveChild struct {
	kind          string
	parentPlural  string
	parentFilters map[string]string
	childPlural   string
	keyField      string
	keyValue      string
}

// exists reports whether the child is currently on the box. A parent that is not
// there at all means the child is gone too — which is what a destroy check needs,
// since Terraform tears the parent down after the child.
func (c testAccLiveChild) exists() (bool, error) {
	client, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	ctx := context.Background()
	parentID, _, found, err := findByKeys(ctx, client, c.parentPlural, c.parentFilters)
	if err != nil {
		return false, fmt.Errorf("looking up the parent of %s %q: %w", c.kind, c.keyValue, err)
	}
	if !found {
		return false, nil
	}
	_, _, found, err = findByKeyInParent(ctx, client, c.childPlural, formatID(parentID), c.keyField, c.keyValue)
	if err != nil {
		return false, fmt.Errorf("looking up %s %q: %w", c.kind, c.keyValue, err)
	}
	return found, nil
}

// preCheckAbsent refuses to start when the throwaway child is already on the box.
func (c testAccLiveChild) preCheckAbsent(t *testing.T) {
	t.Helper()

	found, err := c.exists()
	if err != nil {
		t.Fatalf("checking %s %q does not already exist: %v", c.kind, c.keyValue, err)
	}
	if found {
		t.Fatalf("%s %q already exists on the box; remove it before running the live test", c.kind, c.keyValue)
	}
}

// checkAbsent verifies the child is gone from the box.
func (c testAccLiveChild) checkAbsent() resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := c.exists()
		if err != nil {
			return fmt.Errorf("checking %s %q was destroyed: %w", c.kind, c.keyValue, err)
		}
		if found {
			return fmt.Errorf("%s %q still exists after destroy", c.kind, c.keyValue)
		}
		return nil
	}
}

// testAccLiveComposite identifies an object whose natural key spans several
// fields — a DNS override's host and domain, say — and resolves it the way the
// composite-key resources do. `label` is the rendered key, used in messages.
type testAccLiveComposite struct {
	kind    string
	plural  string
	filters map[string]string
	label   string
}

func (o testAccLiveComposite) exists() (bool, error) {
	client, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKeys(context.Background(), client, o.plural, o.filters)
	if err != nil {
		return false, fmt.Errorf("looking up %s %q: %w", o.kind, o.label, err)
	}
	return found, nil
}

func (o testAccLiveComposite) preCheckAbsent(t *testing.T) {
	t.Helper()

	found, err := o.exists()
	if err != nil {
		t.Fatalf("checking %s %q does not already exist: %v", o.kind, o.label, err)
	}
	if found {
		t.Fatalf("%s %q already exists on the box; remove it before running the live test", o.kind, o.label)
	}
}

func (o testAccLiveComposite) checkAbsent() resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := o.exists()
		if err != nil {
			return fmt.Errorf("checking %s %q was destroyed: %w", o.kind, o.label, err)
		}
		if found {
			return fmt.Errorf("%s %q still exists after destroy", o.kind, o.label)
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_server
// ---------------------------------------------------------------------------

// The throwaway DHCP server is always disabled (see the safety note at the top
// of this file): it is configuration on the management interface that dhcpd
// never acts on. The model has no free-form description field, so nothing here
// can carry the `tftest_` marker — the guard below refuses to run when a DHCP
// server is already configured on the interface, so the test can only ever
// create and destroy its own.
const (
	testAccLiveDHCPInterface = "wan"
	testAccLiveDHCPRangeFrom = "10.99.0.200"
	testAccLiveDHCPRangeTo   = "10.99.0.210"
)

// testAccServicesDHCPServerBlock renders one disabled DHCP server. Every field
// the API echoes back is pinned: they are all Optional (never Computed), so an
// omitted one would read back as the API default and leave a diff that never
// settles. `extra` carries per-test additions such as a lifecycle block.
func testAccServicesDHCPServerBlock(label string, defaultLeaseTime int, extra string) string {
	return fmt.Sprintf(`
resource "pfsense_services_dhcp_server" %q {
  interface        = %q
  enable           = false
  range_from       = %q
  range_to         = %q
  domain           = ""
  gateway          = ""
  defaultleasetime = %d
  maxleasetime     = 86400
  staticmap        = []
%s}
`, label, testAccLiveDHCPInterface, testAccLiveDHCPRangeFrom, testAccLiveDHCPRangeTo, defaultLeaseTime, extra)
}

func testAccServicesDHCPServerLiveConfig(defaultLeaseTime int) string {
	return testAccProviderConfig() + testAccServicesDHCPServerBlock("live", defaultLeaseTime, "")
}

func TestAccServicesDHCPServerResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccCheckLiveObjectAbsent("DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface),
			testAccCheckLiveDHCPDaemonStopped(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDHCPServerLiveConfig(7200),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "id", testAccLiveDHCPInterface),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "interface", testAccLiveDHCPInterface),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "enable", "false"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "range_from", testAccLiveDHCPRangeFrom),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "range_to", testAccLiveDHCPRangeTo),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "defaultleasetime", "7200"),
					// The box must agree, and dhcpd must still be stopped.
					testAccCheckLiveDHCPServerDisabled(),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// `interface` (the natural key) is unchanged, so this is an in-place
				// update; only the default lease time moves.
				Config: testAccServicesDHCPServerLiveConfig(3600),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "id", testAccLiveDHCPInterface),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "defaultleasetime", "3600"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_server.live", "enable", "false"),
					testAccCheckLiveDHCPServerDisabled(),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// DHCP servers are imported by interface — their natural key doubles
				// as the Terraform ID.
				ResourceName:      "pfsense_services_dhcp_server.live",
				ImportState:       true,
				ImportStateId:     testAccLiveDHCPInterface,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccCheckLiveDHCPServerDisabled asserts the box stored the DHCP server as
// disabled. It is the check that keeps this test honest: a step that ever
// enabled DHCP on the management interface would fail here.
func testAccCheckLiveDHCPServerDisabled() resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := testAccClient()
		if err != nil {
			return fmt.Errorf("building verification client: %w", err)
		}
		_, obj, found, err := findByKey(context.Background(), client, dhcpServerPlural, "interface", testAccLiveDHCPInterface)
		if err != nil {
			return fmt.Errorf("reading the DHCP server on %s: %w", testAccLiveDHCPInterface, err)
		}
		if !found {
			return fmt.Errorf("no DHCP server is configured on %s", testAccLiveDHCPInterface)
		}
		if enable := getBool(obj, "enable"); enable == nil || *enable {
			return fmt.Errorf("the DHCP server on %s is enabled; the live tests never enable one", testAccLiveDHCPInterface)
		}
		return nil
	}
}

// testAccCheckLiveDHCPDaemonStopped asserts the box is not running a DHCP
// daemon. Nothing in this batch may start one on the management network.
func testAccCheckLiveDHCPDaemonStopped() resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := testAccClient()
		if err != nil {
			return fmt.Errorf("building verification client: %w", err)
		}
		items, err := client.List(context.Background(), servicesStatusPlural, nil)
		if err != nil {
			return fmt.Errorf("reading the service inventory: %w", err)
		}
		for _, item := range items {
			service, err := decodeObject(item)
			if err != nil {
				return fmt.Errorf("decoding a service inventory entry: %w", err)
			}
			name := objectKey(service, "name")
			if (name == "dhcpd" || name == "kea-dhcp4") && objectKey(service, "status") == "true" {
				return fmt.Errorf("%s is running on the box; the live tests never enable DHCP", name)
			}
		}
		return nil
	}
}

// servicesStatusPlural is the box's service inventory, used by the DHCP checks
// to prove no DHCP daemon was started.
const servicesStatusPlural = "/api/v2/status/services"

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_static_mapping
// ---------------------------------------------------------------------------

// A static mapping is keyed by the parent interface and a MAC address. MACs are
// hex, so the `tftest_` marker lives in `descr`; the address itself is a
// throwaway and the reserved IP sits in the parent's subnet but outside its
// (never served) range.
const (
	testAccLiveDHCPMappingMAC      = "00:11:22:33:44:55"
	testAccLiveDHCPMappingIP       = "10.99.0.220"
	testAccLiveDHCPMappingHostname = "tftest-live-mapping"
	testAccLiveDHCPMappingDescr    = "tftest_live_static_mapping"
)

// testAccLiveDHCPMappingChild locates the mapping on the box for the guards.
var testAccLiveDHCPMappingChild = testAccLiveChild{
	kind:          "DHCP static mapping",
	parentPlural:  dhcpParentPlural,
	parentFilters: map[string]string{"interface": testAccLiveDHCPInterface},
	childPlural:   dhcpStaticMappingPlural,
	keyField:      "mac",
	keyValue:      testAccLiveDHCPMappingMAC,
}

// testAccServicesDHCPStaticMappingLiveConfig renders the mapping and the
// disabled DHCP server it hangs off. The parent ignores changes to `staticmap`:
// the API reports this very child inside the parent's own collection, so a
// parent that managed the field would fight the child resource for it.
func testAccServicesDHCPStaticMappingLiveConfig(ipaddr, descr string) string {
	return testAccProviderConfig() +
		testAccServicesDHCPServerBlock("parent", 7200, `
  lifecycle {
    ignore_changes = [staticmap]
  }
`) +
		fmt.Sprintf(`
resource "pfsense_services_dhcp_static_mapping" "live" {
  parent_id              = pfsense_services_dhcp_server.parent.interface
  mac                    = %q
  ipaddr                 = %q
  hostname               = %q
  descr                  = %q
  domain                 = ""
  gateway                = ""
  domainsearchlist       = []
  defaultleasetime       = 7200
  maxleasetime           = 86400
  arp_table_static_entry = false
}
`, testAccLiveDHCPMappingMAC, ipaddr, testAccLiveDHCPMappingHostname, descr)
}

func TestAccServicesDHCPStaticMappingResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface)
			testAccLiveDHCPMappingChild.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccLiveDHCPMappingChild.checkAbsent(),
			testAccCheckLiveObjectAbsent("DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface),
			testAccCheckLiveDHCPDaemonStopped(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDHCPStaticMappingLiveConfig(testAccLiveDHCPMappingIP, testAccLiveDHCPMappingDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "id", testAccLiveDHCPInterface+"|"+testAccLiveDHCPMappingMAC),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "parent_id", testAccLiveDHCPInterface),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "mac", testAccLiveDHCPMappingMAC),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "ipaddr", testAccLiveDHCPMappingIP),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "hostname", testAccLiveDHCPMappingHostname),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "descr", testAccLiveDHCPMappingDescr),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// Both key components (the parent interface and the MAC) are
				// unchanged, so this is an in-place update; the reserved address and
				// the description move.
				Config: testAccServicesDHCPStaticMappingLiveConfig("10.99.0.221", testAccLiveDHCPMappingDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "id", testAccLiveDHCPInterface+"|"+testAccLiveDHCPMappingMAC),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "ipaddr", "10.99.0.221"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_static_mapping.live", "descr", testAccLiveDHCPMappingDescr+" (updated)"),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// The import ID is the parent interface and the MAC, joined by a pipe.
				ResourceName:      "pfsense_services_dhcp_static_mapping.live",
				ImportState:       true,
				ImportStateId:     testAccLiveDHCPInterface + "|" + testAccLiveDHCPMappingMAC,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_address_pool
// ---------------------------------------------------------------------------

// An address pool is keyed by the parent interface and its starting address.
// Neither can carry the `tftest_` marker, so the guards below make sure the test
// only ever manages a pool it created itself. The range sits above the parent's
// own (never served) range.
const (
	testAccLiveDHCPPoolRangeFrom = "10.99.0.230"
	testAccLiveDHCPPoolRangeTo   = "10.99.0.235"
)

var testAccLiveDHCPPoolChild = testAccLiveChild{
	kind:          "DHCP address pool",
	parentPlural:  dhcpParentPlural,
	parentFilters: map[string]string{"interface": testAccLiveDHCPInterface},
	childPlural:   dhcpAddressPoolPlural,
	keyField:      "range_from",
	keyValue:      testAccLiveDHCPPoolRangeFrom,
}

// testAccServicesDHCPAddressPoolLiveConfig renders the pool and the disabled
// DHCP server it hangs off. The parent needs no lifecycle block here: pools live
// in the parent's `pool` collection, which the DHCP server resource does not
// manage.
func testAccServicesDHCPAddressPoolLiveConfig(rangeTo string) string {
	return testAccProviderConfig() +
		testAccServicesDHCPServerBlock("parent", 7200, "") +
		fmt.Sprintf(`
resource "pfsense_services_dhcp_address_pool" "live" {
  parent_id        = pfsense_services_dhcp_server.parent.interface
  range_from       = %q
  range_to         = %q
  domain           = ""
  gateway          = ""
  mac_allow        = []
  mac_deny         = []
  domainsearchlist = []
  defaultleasetime = 7200
  maxleasetime     = 86400
  ignorebootp      = false
  ignoreclientuids = false
}
`, testAccLiveDHCPPoolRangeFrom, rangeTo)
}

func TestAccServicesDHCPAddressPoolResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface)
			testAccLiveDHCPPoolChild.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccLiveDHCPPoolChild.checkAbsent(),
			testAccCheckLiveObjectAbsent("DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface),
			testAccCheckLiveDHCPDaemonStopped(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDHCPAddressPoolLiveConfig(testAccLiveDHCPPoolRangeTo),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_address_pool.live", "id", testAccLiveDHCPInterface+"|"+testAccLiveDHCPPoolRangeFrom),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_address_pool.live", "parent_id", testAccLiveDHCPInterface),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_address_pool.live", "range_from", testAccLiveDHCPPoolRangeFrom),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_address_pool.live", "range_to", testAccLiveDHCPPoolRangeTo),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// Both key components (the parent interface and the starting address)
				// are unchanged, so this is an in-place update; the end of the range
				// moves.
				Config: testAccServicesDHCPAddressPoolLiveConfig("10.99.0.236"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_address_pool.live", "id", testAccLiveDHCPInterface+"|"+testAccLiveDHCPPoolRangeFrom),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_address_pool.live", "range_to", "10.99.0.236"),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// The import ID is the parent interface and the starting address,
				// joined by a pipe.
				ResourceName:      "pfsense_services_dhcp_address_pool.live",
				ImportState:       true,
				ImportStateId:     testAccLiveDHCPInterface + "|" + testAccLiveDHCPPoolRangeFrom,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dhcp_custom_option
// ---------------------------------------------------------------------------

// Custom option 114 is the captive-portal URL DHCP option (RFC 8910). It is
// keyed by the parent interface and the option number, so the `tftest_` marker
// lives in the value, which is a documentation-range URL that is never handed
// out (the parent DHCP server is disabled).
const (
	testAccLiveDHCPOptionNumber = 114
	testAccLiveDHCPOptionValue  = "http://192.0.2.1/tftest_live_custom_option"
)

var testAccLiveDHCPOptionChild = testAccLiveChild{
	kind:          "DHCP custom option",
	parentPlural:  dhcpParentPlural,
	parentFilters: map[string]string{"interface": testAccLiveDHCPInterface},
	childPlural:   dhcpCustomOptionPlural,
	keyField:      "number",
	keyValue:      "114",
}

func testAccServicesDHCPCustomOptionLiveConfig(value string) string {
	return testAccProviderConfig() +
		testAccServicesDHCPServerBlock("parent", 7200, "") +
		fmt.Sprintf(`
resource "pfsense_services_dhcp_custom_option" "live" {
  parent_id = pfsense_services_dhcp_server.parent.interface
  number    = %d
  type      = "text"
  value     = %q
}
`, testAccLiveDHCPOptionNumber, value)
}

func TestAccServicesDHCPCustomOptionResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface)
			testAccLiveDHCPOptionChild.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccLiveDHCPOptionChild.checkAbsent(),
			testAccCheckLiveObjectAbsent("DHCP server", dhcpServerPlural, "interface", testAccLiveDHCPInterface),
			testAccCheckLiveDHCPDaemonStopped(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDHCPCustomOptionLiveConfig(testAccLiveDHCPOptionValue),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "id", testAccLiveDHCPInterface+"|114"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "parent_id", testAccLiveDHCPInterface),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "number", "114"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "type", "text"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "value", testAccLiveDHCPOptionValue),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// Both key components (the parent interface and the option number)
				// are unchanged, so this is an in-place update; only the value moves.
				Config: testAccServicesDHCPCustomOptionLiveConfig(testAccLiveDHCPOptionValue + "/updated"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "id", testAccLiveDHCPInterface+"|114"),
					resource.TestCheckResourceAttr("pfsense_services_dhcp_custom_option.live", "value", testAccLiveDHCPOptionValue+"/updated"),
					testAccCheckLiveDHCPDaemonStopped(),
				),
			},
			{
				// The import ID is the parent interface and the option number, joined
				// by a pipe.
				ResourceName:      "pfsense_services_dhcp_custom_option.live",
				ImportState:       true,
				ImportStateId:     testAccLiveDHCPInterface + "|114",
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_host_override
// ---------------------------------------------------------------------------

// A host override is keyed by host and domain. The override resolves a
// `tftest_`-marked hostname under example.com (RFC 2606) to a documentation
// address (RFC 5737), so it can only ever shadow a name that does not resolve to
// anything real. The resolver's own settings are never touched.
const (
	testAccLiveResolverOverrideHost   = "tftest-live-resolver"
	testAccLiveResolverOverrideDomain = "example.com"
	testAccLiveResolverOverrideIP     = "192.0.2.10"
	testAccLiveResolverOverrideDescr  = "tftest_live_resolver_host_override"
)

var testAccLiveResolverOverrideObject = testAccLiveComposite{
	kind:    "DNS resolver host override",
	plural:  dnsResolverHostOverridePlural,
	filters: map[string]string{"host": testAccLiveResolverOverrideHost, "domain": testAccLiveResolverOverrideDomain},
	label:   testAccLiveResolverOverrideHost + "|" + testAccLiveResolverOverrideDomain,
}

// testAccLiveAliasIgnoreLifecycle is the lifecycle block the two alias tests put
// on their parent override. The API reports an alias inside its parent's own
// `aliases` collection, so a parent that managed the field would plan a diff
// against the child resource on every refresh.
const testAccLiveAliasIgnoreLifecycle = `
  lifecycle {
    ignore_changes = [aliases]
  }
`

// testAccServicesDNSResolverHostOverrideBlock renders one host override.
// `aliases` is deliberately left out of the configuration: the API reports it as
// null while the override has none, and a resource that managed it would fight
// the alias child resource for the same collection. `extra` carries per-test
// additions such as a lifecycle block.
func testAccServicesDNSResolverHostOverrideBlock(label, host, ip, descr, extra string) string {
	return fmt.Sprintf(`
resource "pfsense_services_dns_resolver_host_override" %q {
  host   = %q
  domain = %q
  ip     = [%q]
  descr  = %q
%s}
`, label, host, testAccLiveResolverOverrideDomain, ip, descr, extra)
}

func testAccServicesDNSResolverHostOverrideLiveConfig(ip, descr string) string {
	return testAccProviderConfig() +
		testAccServicesDNSResolverHostOverrideBlock("live", testAccLiveResolverOverrideHost, ip, descr, "")
}

func TestAccServicesDNSResolverHostOverrideResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccLiveResolverOverrideObject.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccLiveResolverOverrideObject.checkAbsent(),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDNSResolverHostOverrideLiveConfig(testAccLiveResolverOverrideIP, testAccLiveResolverOverrideDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "id", testAccLiveResolverOverrideObject.label),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "host", testAccLiveResolverOverrideHost),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "domain", testAccLiveResolverOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "ip.#", "1"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "ip.0", testAccLiveResolverOverrideIP),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "descr", testAccLiveResolverOverrideDescr),
				),
			},
			{
				// Both key components (host and domain) are unchanged, so this is an
				// in-place update; the address and the description move.
				Config: testAccServicesDNSResolverHostOverrideLiveConfig("192.0.2.11", testAccLiveResolverOverrideDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "id", testAccLiveResolverOverrideObject.label),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "ip.0", "192.0.2.11"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override.live", "descr", testAccLiveResolverOverrideDescr+" (updated)"),
				),
			},
			{
				// The import ID is the host and the domain, joined by a pipe.
				ResourceName:      "pfsense_services_dns_resolver_host_override.live",
				ImportState:       true,
				ImportStateId:     testAccLiveResolverOverrideObject.label,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_host_override_alias
// ---------------------------------------------------------------------------

// The alias hangs off its own throwaway host override rather than the one the
// test above manages, so the two tests cannot collide.
const (
	testAccLiveResolverAliasParentHost = "tftest-live-resolver-parent"
	testAccLiveResolverAliasHost       = "tftest-live-resolver-alias"
	testAccLiveResolverAliasDescr      = "tftest_live_resolver_host_override_alias"
)

var (
	testAccLiveResolverAliasParentKey = testAccLiveResolverAliasParentHost + "|" + testAccLiveResolverOverrideDomain

	testAccLiveResolverAliasParentObject = testAccLiveComposite{
		kind:    "DNS resolver host override",
		plural:  dnsResolverHostOverridePlural,
		filters: map[string]string{"host": testAccLiveResolverAliasParentHost, "domain": testAccLiveResolverOverrideDomain},
		label:   testAccLiveResolverAliasParentKey,
	}

	testAccLiveResolverAliasChild = testAccLiveChild{
		kind:          "DNS resolver host override alias",
		parentPlural:  dnsResolverHostOverrideParentPlural,
		parentFilters: map[string]string{"host": testAccLiveResolverAliasParentHost, "domain": testAccLiveResolverOverrideDomain},
		childPlural:   dnsResolverHostOverrideAliasPlural,
		keyField:      "host",
		keyValue:      testAccLiveResolverAliasHost,
	}
)

func testAccServicesDNSResolverHostOverrideAliasLiveConfig(descr string) string {
	return testAccProviderConfig() +
		testAccServicesDNSResolverHostOverrideBlock(
			"alias_parent",
			testAccLiveResolverAliasParentHost,
			testAccLiveResolverOverrideIP,
			"tftest_live_resolver_alias_parent",
			testAccLiveAliasIgnoreLifecycle,
		) +
		fmt.Sprintf(`
resource "pfsense_services_dns_resolver_host_override_alias" "live" {
  parent_id = "${pfsense_services_dns_resolver_host_override.alias_parent.host}|${pfsense_services_dns_resolver_host_override.alias_parent.domain}"
  host      = %q
  domain    = %q
  descr     = %q
}
`, testAccLiveResolverAliasHost, testAccLiveResolverOverrideDomain, descr)
}

func TestAccServicesDNSResolverHostOverrideAliasResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccLiveResolverAliasParentObject.preCheckAbsent(t)
			testAccLiveResolverAliasChild.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccLiveResolverAliasChild.checkAbsent(),
			testAccLiveResolverAliasParentObject.checkAbsent(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDNSResolverHostOverrideAliasLiveConfig(testAccLiveResolverAliasDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"pfsense_services_dns_resolver_host_override_alias.live", "id",
						testAccLiveResolverAliasParentKey+"|"+testAccLiveResolverAliasHost,
					),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override_alias.live", "parent_id", testAccLiveResolverAliasParentKey),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override_alias.live", "host", testAccLiveResolverAliasHost),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override_alias.live", "domain", testAccLiveResolverOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override_alias.live", "descr", testAccLiveResolverAliasDescr),
				),
			},
			{
				// The parent override and the alias hostname are unchanged, so this is
				// an in-place update; only the description moves.
				Config: testAccServicesDNSResolverHostOverrideAliasLiveConfig(testAccLiveResolverAliasDescr + " (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"pfsense_services_dns_resolver_host_override_alias.live", "id",
						testAccLiveResolverAliasParentKey+"|"+testAccLiveResolverAliasHost,
					),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_host_override_alias.live", "descr", testAccLiveResolverAliasDescr+" (updated)"),
				),
			},
			{
				// The import ID is the parent's own key and the alias hostname, joined
				// by pipes.
				ResourceName:      "pfsense_services_dns_resolver_host_override_alias.live",
				ImportState:       true,
				ImportStateId:     testAccLiveResolverAliasParentKey + "|" + testAccLiveResolverAliasHost,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_domain_override
// ---------------------------------------------------------------------------

// The domain override sends queries for one `tftest_`-marked subdomain of
// example.com (RFC 2606) to a documentation-range nameserver (RFC 5737) that
// does not exist. Nothing the box itself resolves lives under that domain, and
// the resolver keeps answering everything else exactly as before.
const (
	testAccLiveDomainOverrideDomain = "tftest-live.example.com"
	testAccLiveDomainOverrideIP     = "192.0.2.53"
	testAccLiveDomainOverrideDescr  = "tftest_live_domain_override"
)

// testAccServicesDNSResolverDomainOverrideLiveConfig renders the override for
// one step. `forward_tls_upstream` is pinned to the API default because the API
// reports it on every read.
func testAccServicesDNSResolverDomainOverrideLiveConfig(ip, descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_services_dns_resolver_domain_override" "live" {
  domain               = %q
  ip                   = %q
  descr                = %q
  forward_tls_upstream = false
}
`, testAccLiveDomainOverrideDomain, ip, descr)
}

func TestAccServicesDNSResolverDomainOverrideResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveObjectAbsent(t, "DNS resolver domain override", dnsResolverDomainOverridePlural, "domain", testAccLiveDomainOverrideDomain)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckLiveObjectAbsent("DNS resolver domain override", dnsResolverDomainOverridePlural, "domain", testAccLiveDomainOverrideDomain),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDNSResolverDomainOverrideLiveConfig(testAccLiveDomainOverrideIP, testAccLiveDomainOverrideDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "id", testAccLiveDomainOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "domain", testAccLiveDomainOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "ip", testAccLiveDomainOverrideIP),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "descr", testAccLiveDomainOverrideDescr),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "forward_tls_upstream", "false"),
				),
			},
			{
				// `domain` (the natural key) is unchanged; the nameserver address and
				// the description move.
				Config: testAccServicesDNSResolverDomainOverrideLiveConfig("192.0.2.54", testAccLiveDomainOverrideDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "id", testAccLiveDomainOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "ip", "192.0.2.54"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_domain_override.live", "descr", testAccLiveDomainOverrideDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_services_dns_resolver_domain_override.live",
				ImportState:       true,
				ImportStateId:     testAccLiveDomainOverrideDomain,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_forwarder_host_override
// ---------------------------------------------------------------------------

// The forwarder (dnsmasq) is not the box's active resolver — unbound is — so
// these entries are pure configuration. The test neither starts nor stops the
// forwarder; it only adds and removes an override for a `tftest_`-marked
// hostname under example.com pointing at a documentation address.
const (
	testAccLiveForwarderOverrideHost  = "tftest-live-forwarder"
	testAccLiveForwarderOverrideIP    = "192.0.2.20"
	testAccLiveForwarderOverrideDescr = "tftest_live_forwarder_host_override"
)

var testAccLiveForwarderOverrideObject = testAccLiveComposite{
	kind:    "DNS forwarder host override",
	plural:  dnsForwarderHostOverridePlural,
	filters: map[string]string{"host": testAccLiveForwarderOverrideHost, "domain": testAccLiveResolverOverrideDomain},
	label:   testAccLiveForwarderOverrideHost + "|" + testAccLiveResolverOverrideDomain,
}

// testAccServicesDNSForwarderHostOverrideBlock renders one forwarder override.
// As with the resolver, `aliases` is left unmanaged so the alias child resource
// owns that collection on its own.
func testAccServicesDNSForwarderHostOverrideBlock(label, host, ip, descr, extra string) string {
	return fmt.Sprintf(`
resource "pfsense_services_dns_forwarder_host_override" %q {
  host   = %q
  domain = %q
  ip     = %q
  descr  = %q
%s}
`, label, host, testAccLiveResolverOverrideDomain, ip, descr, extra)
}

func testAccServicesDNSForwarderHostOverrideLiveConfig(ip, descr string) string {
	return testAccProviderConfig() +
		testAccServicesDNSForwarderHostOverrideBlock("live", testAccLiveForwarderOverrideHost, ip, descr, "")
}

func TestAccServicesDNSForwarderHostOverrideResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccLiveForwarderOverrideObject.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccLiveForwarderOverrideObject.checkAbsent(),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDNSForwarderHostOverrideLiveConfig(testAccLiveForwarderOverrideIP, testAccLiveForwarderOverrideDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "id", testAccLiveForwarderOverrideObject.label),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "host", testAccLiveForwarderOverrideHost),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "domain", testAccLiveResolverOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "ip", testAccLiveForwarderOverrideIP),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "descr", testAccLiveForwarderOverrideDescr),
				),
			},
			{
				// Both key components (host and domain) are unchanged, so this is an
				// in-place update; the address and the description move.
				Config: testAccServicesDNSForwarderHostOverrideLiveConfig("192.0.2.21", testAccLiveForwarderOverrideDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "id", testAccLiveForwarderOverrideObject.label),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "ip", "192.0.2.21"),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override.live", "descr", testAccLiveForwarderOverrideDescr+" (updated)"),
				),
			},
			{
				// The import ID is the host and the domain, joined by a pipe.
				ResourceName:      "pfsense_services_dns_forwarder_host_override.live",
				ImportState:       true,
				ImportStateId:     testAccLiveForwarderOverrideObject.label,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_forwarder_host_override_alias
// ---------------------------------------------------------------------------

// As with the resolver alias, the forwarder alias brings its own throwaway
// parent override so the two forwarder tests cannot collide.
const (
	testAccLiveForwarderAliasParentHost = "tftest-live-forwarder-parent"
	testAccLiveForwarderAliasHost       = "tftest-live-forwarder-alias"
	testAccLiveForwarderAliasDescr      = "tftest_live_forwarder_host_override_alias"
)

var (
	testAccLiveForwarderAliasParentKey = testAccLiveForwarderAliasParentHost + "|" + testAccLiveResolverOverrideDomain

	testAccLiveForwarderAliasParentObject = testAccLiveComposite{
		kind:    "DNS forwarder host override",
		plural:  dnsForwarderHostOverridePlural,
		filters: map[string]string{"host": testAccLiveForwarderAliasParentHost, "domain": testAccLiveResolverOverrideDomain},
		label:   testAccLiveForwarderAliasParentKey,
	}

	testAccLiveForwarderAliasChild = testAccLiveChild{
		kind:          "DNS forwarder host override alias",
		parentPlural:  dnsForwarderHostOverrideParentPlural,
		parentFilters: map[string]string{"host": testAccLiveForwarderAliasParentHost, "domain": testAccLiveResolverOverrideDomain},
		childPlural:   dnsForwarderHostOverrideAliasPlural,
		keyField:      "host",
		keyValue:      testAccLiveForwarderAliasHost,
	}
)

func testAccServicesDNSForwarderHostOverrideAliasLiveConfig(description string) string {
	return testAccProviderConfig() +
		testAccServicesDNSForwarderHostOverrideBlock(
			"alias_parent",
			testAccLiveForwarderAliasParentHost,
			testAccLiveForwarderOverrideIP,
			"tftest_live_forwarder_alias_parent",
			testAccLiveAliasIgnoreLifecycle,
		) +
		fmt.Sprintf(`
resource "pfsense_services_dns_forwarder_host_override_alias" "live" {
  parent_id   = "${pfsense_services_dns_forwarder_host_override.alias_parent.host}|${pfsense_services_dns_forwarder_host_override.alias_parent.domain}"
  host        = %q
  domain      = %q
  description = %q
}
`, testAccLiveForwarderAliasHost, testAccLiveResolverOverrideDomain, description)
}

func TestAccServicesDNSForwarderHostOverrideAliasResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccLiveForwarderAliasParentObject.preCheckAbsent(t)
			testAccLiveForwarderAliasChild.preCheckAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckLiveObjectsAbsent(
			testAccLiveForwarderAliasChild.checkAbsent(),
			testAccLiveForwarderAliasParentObject.checkAbsent(),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccServicesDNSForwarderHostOverrideAliasLiveConfig(testAccLiveForwarderAliasDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"pfsense_services_dns_forwarder_host_override_alias.live", "id",
						testAccLiveForwarderAliasParentKey+"|"+testAccLiveForwarderAliasHost,
					),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override_alias.live", "parent_id", testAccLiveForwarderAliasParentKey),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override_alias.live", "host", testAccLiveForwarderAliasHost),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override_alias.live", "domain", testAccLiveResolverOverrideDomain),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override_alias.live", "description", testAccLiveForwarderAliasDescr),
				),
			},
			{
				// The parent override and the alias hostname are unchanged, so this is
				// an in-place update; only the description moves.
				Config: testAccServicesDNSForwarderHostOverrideAliasLiveConfig(testAccLiveForwarderAliasDescr + " (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"pfsense_services_dns_forwarder_host_override_alias.live", "id",
						testAccLiveForwarderAliasParentKey+"|"+testAccLiveForwarderAliasHost,
					),
					resource.TestCheckResourceAttr("pfsense_services_dns_forwarder_host_override_alias.live", "description", testAccLiveForwarderAliasDescr+" (updated)"),
				),
			},
			{
				// The import ID is the parent's own key and the alias hostname, joined
				// by pipes.
				ResourceName:      "pfsense_services_dns_forwarder_host_override_alias.live",
				ImportState:       true,
				ImportStateId:     testAccLiveForwarderAliasParentKey + "|" + testAccLiveForwarderAliasHost,
				ImportStateVerify: true,
			},
		},
	})
}
