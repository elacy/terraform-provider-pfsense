package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live coverage for the interface family. Everything here runs against a real
// pfSense box and is gated by testAccPreCheck, so `go test ./...` without
// TF_ACC stays hermetic.
//
// SAFETY. The reference box has exactly one physical NIC, `vtnet0`, which
// carries the `wan` assignment and the management address the tests themselves
// connect to. Nothing below may disable, unassign, bridge or otherwise disturb
// it:
//
//   - VLAN, GRE and interface group create brand new logical objects and are
//     exercised with full CRUD.
//   - LAGG requires at least one member; it is given a throwaway VLAN created
//     in the same config, so the aggregation never covers `vtnet0` itself.
//   - Bridge cannot be tested here at all — see the comment above
//     TestAccInterfaceBridgeResourceLive.
//   - pfsense_network_interface is exercised read-only — see the comment above
//     TestAccNetworkInterfaceResourceLive.

// ---------------------------------------------------------------------------
// pfsense_interface_vlan
// ---------------------------------------------------------------------------

// VLANs are keyed on if|tag, which is also the Terraform ID and the import ID.
// The parent is the physical NIC: adding a VLAN clone to it does not touch the
// parent's own assignment or address. Tag 4090 sits at the very top of the
// usable range, well away from anything a real deployment would allocate.
const (
	testAccLiveVLANParent = "vtnet0"
	testAccLiveVLANTag    = "4090"
	testAccLiveVLANID     = testAccLiveVLANParent + "|" + testAccLiveVLANTag
	testAccLiveVLANDescr  = "tftest_live_vlan"
)

func testAccInterfaceVLANExists(parent, tag string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKeys(context.Background(), c, interfaceVLANPlural, map[string]string{
		"if":  parent,
		"tag": tag,
	})
	if err != nil {
		return false, fmt.Errorf("looking up VLAN %q: %w", parent+"|"+tag, err)
	}
	return found, nil
}

func testAccPreCheckLiveInterfaceVLANAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccInterfaceVLANExists(testAccLiveVLANParent, testAccLiveVLANTag)
	if err != nil {
		t.Fatalf("checking VLAN %q does not already exist: %v", testAccLiveVLANID, err)
	}
	if found {
		t.Fatalf("VLAN %q already exists on the box; remove it before running the live test", testAccLiveVLANID)
	}
}

func testAccCheckInterfaceVLANAbsent(parent, tag string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccInterfaceVLANExists(parent, tag)
		if err != nil {
			return fmt.Errorf("checking VLAN %q was destroyed: %w", parent+"|"+tag, err)
		}
		if found {
			return fmt.Errorf("VLAN %q still exists after destroy", parent+"|"+tag)
		}
		return nil
	}
}

func testAccInterfaceVLANLiveConfig(pcp int, descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_interface_vlan" "live" {
  if    = %q
  tag   = %s
  pcp   = %d
  descr = %q
}
`, testAccLiveVLANParent, testAccLiveVLANTag, pcp, descr)
}

func TestAccInterfaceVLANResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveInterfaceVLANAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInterfaceVLANAbsent(testAccLiveVLANParent, testAccLiveVLANTag),
		Steps: []resource.TestStep{
			{
				Config: testAccInterfaceVLANLiveConfig(1, testAccLiveVLANDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "id", testAccLiveVLANID),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "if", testAccLiveVLANParent),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "tag", testAccLiveVLANTag),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "pcp", "1"),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "descr", testAccLiveVLANDescr),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "vlanif",
						testAccLiveVLANParent+"."+testAccLiveVLANTag),
				),
			},
			{
				// Both key components (`if` and `tag`) are unchanged, so this is
				// an in-place update; only the priority and the description move.
				Config: testAccInterfaceVLANLiveConfig(5, testAccLiveVLANDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "id", testAccLiveVLANID),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "if", testAccLiveVLANParent),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "tag", testAccLiveVLANTag),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "pcp", "5"),
					resource.TestCheckResourceAttr("pfsense_interface_vlan.live", "descr", testAccLiveVLANDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_interface_vlan.live",
				ImportState:       true,
				ImportStateId:     testAccLiveVLANID,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_interface_gre
// ---------------------------------------------------------------------------

// A GRE tunnel is keyed on the interface it binds to, so this is one of the
// live objects that cannot carry a `tftest_` identity; the description does
// instead. Creating it only adds a `gre0` clone — the bound interface keeps its
// assignment and address — and both tunnel endpoints are documentation-range
// addresses that no traffic can reach.
const (
	testAccLiveGREIface = "wan"
	testAccLiveGREDescr = "tftest_live_gre"
)

func testAccInterfaceGREExists(iface string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, interfaceGREPlural, "if", iface)
	if err != nil {
		return false, fmt.Errorf("looking up GRE interface on %q: %w", iface, err)
	}
	return found, nil
}

func testAccPreCheckLiveInterfaceGREAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccInterfaceGREExists(testAccLiveGREIface)
	if err != nil {
		t.Fatalf("checking no GRE interface exists on %q: %v", testAccLiveGREIface, err)
	}
	if found {
		t.Fatalf("a GRE interface already exists on %q; remove it before running the live test", testAccLiveGREIface)
	}
}

func testAccCheckInterfaceGREAbsent(iface string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccInterfaceGREExists(iface)
		if err != nil {
			return fmt.Errorf("checking GRE interface on %q was destroyed: %w", iface, err)
		}
		if found {
			return fmt.Errorf("GRE interface on %q still exists after destroy", iface)
		}
		return nil
	}
}

// testAccInterfaceGRELiveConfig renders the GRE config for one step. The whole
// IPv6 tunnel triple is set because the API only stores `tunnel_remote_addr6`
// and `tunnel_remote_net6` when `tunnel_local_addr6` accompanies them; setting
// the remote half alone leaves them null on read and the plan never settles.
func testAccInterfaceGRELiveConfig(remoteAddr, descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_interface_gre" "live" {
  if                  = %q
  descr               = %q
  add_static_route    = false
  remote_addr         = %q
  tunnel_local_addr   = "192.0.2.1"
  tunnel_remote_addr  = "192.0.2.2"
  tunnel_remote_net   = 30
  tunnel_local_addr6  = "2001:db8::1"
  tunnel_remote_addr6 = "2001:db8::2"
  tunnel_remote_net6  = 64
}
`, testAccLiveGREIface, descr, remoteAddr)
}

func TestAccInterfaceGREResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveInterfaceGREAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInterfaceGREAbsent(testAccLiveGREIface),
		Steps: []resource.TestStep{
			{
				Config: testAccInterfaceGRELiveConfig("198.51.100.50", testAccLiveGREDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "id", testAccLiveGREIface),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "if", testAccLiveGREIface),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "descr", testAccLiveGREDescr),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "remote_addr", "198.51.100.50"),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "tunnel_remote_addr", "192.0.2.2"),
					resource.TestCheckResourceAttrSet("pfsense_interface_gre.live", "greif"),
				),
			},
			{
				// `if` (the natural key) is unchanged; only the far-end address
				// and the description move.
				Config: testAccInterfaceGRELiveConfig("198.51.100.51", testAccLiveGREDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "id", testAccLiveGREIface),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "if", testAccLiveGREIface),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "remote_addr", "198.51.100.51"),
					resource.TestCheckResourceAttr("pfsense_interface_gre.live", "descr", testAccLiveGREDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_interface_gre.live",
				ImportState:       true,
				ImportStateId:     testAccLiveGREIface,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_interface_lagg
// ---------------------------------------------------------------------------

// `members` is required and the API rejects an empty list, so the LAGG needs a
// real member interface. The only physical NIC on the box carries management
// traffic, so instead the config creates a throwaway VLAN clone and aggregates
// that: `vtnet0` keeps its own assignment and address throughout, and Terraform
// tears the LAGG down before the VLAN it depends on.
const (
	testAccLiveLAGGDescr     = "tftest_live_lagg"
	testAccLiveLAGGVLANTag   = "4091"
	testAccLiveLAGGVLANID    = testAccLiveVLANParent + "|" + testAccLiveLAGGVLANTag
	testAccLiveLAGGVLANDescr = "tftest_live_lagg_member"
	testAccLiveLAGGMember    = testAccLiveVLANParent + "." + testAccLiveLAGGVLANTag
)

func testAccInterfaceLAGGExists(descr string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, interfaceLAGGPlural, "descr", descr)
	if err != nil {
		return false, fmt.Errorf("looking up LAGG interface %q: %w", descr, err)
	}
	return found, nil
}

func testAccPreCheckLiveInterfaceLAGGAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccInterfaceLAGGExists(testAccLiveLAGGDescr)
	if err != nil {
		t.Fatalf("checking LAGG interface %q does not already exist: %v", testAccLiveLAGGDescr, err)
	}
	if found {
		t.Fatalf("LAGG interface %q already exists on the box; remove it before running the live test", testAccLiveLAGGDescr)
	}

	found, err = testAccInterfaceVLANExists(testAccLiveVLANParent, testAccLiveLAGGVLANTag)
	if err != nil {
		t.Fatalf("checking member VLAN %q does not already exist: %v", testAccLiveLAGGVLANID, err)
	}
	if found {
		t.Fatalf("member VLAN %q already exists on the box; remove it before running the live test", testAccLiveLAGGVLANID)
	}
}

func testAccCheckInterfaceLAGGAbsent(descr string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		found, err := testAccInterfaceLAGGExists(descr)
		if err != nil {
			return fmt.Errorf("checking LAGG interface %q was destroyed: %w", descr, err)
		}
		if found {
			return fmt.Errorf("LAGG interface %q still exists after destroy", descr)
		}
		return testAccCheckInterfaceVLANAbsent(testAccLiveVLANParent, testAccLiveLAGGVLANTag)(s)
	}
}

// testAccInterfaceLAGGLiveConfig renders the LAGG (and the throwaway VLAN it
// aggregates) for one step. `lacptimeout` and `lagghash` are API defaults that
// come back on every read, so they are pinned to keep the plan empty.
func testAccInterfaceLAGGLiveConfig(lacptimeout string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_interface_vlan" "lagg_member" {
  if    = %q
  tag   = %s
  pcp   = 0
  descr = %q
}

resource "pfsense_interface_lagg" "live" {
  descr       = %q
  members     = [pfsense_interface_vlan.lagg_member.vlanif]
  proto       = "lacp"
  lacptimeout = %q
  lagghash    = "l2,l3,l4"
}
`, testAccLiveVLANParent, testAccLiveLAGGVLANTag, testAccLiveLAGGVLANDescr, testAccLiveLAGGDescr, lacptimeout)
}

func TestAccInterfaceLAGGResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveInterfaceLAGGAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInterfaceLAGGAbsent(testAccLiveLAGGDescr),
		Steps: []resource.TestStep{
			{
				Config: testAccInterfaceLAGGLiveConfig("slow"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "id", testAccLiveLAGGDescr),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "descr", testAccLiveLAGGDescr),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "proto", "lacp"),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "lacptimeout", "slow"),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "members.#", "1"),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "members.0", testAccLiveLAGGMember),
					resource.TestCheckResourceAttrSet("pfsense_interface_lagg.live", "laggif"),
				),
			},
			{
				// `descr` (the natural key) is unchanged; only the LACP timeout
				// moves.
				Config: testAccInterfaceLAGGLiveConfig("fast"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "id", testAccLiveLAGGDescr),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "descr", testAccLiveLAGGDescr),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "lacptimeout", "fast"),
					resource.TestCheckResourceAttr("pfsense_interface_lagg.live", "members.0", testAccLiveLAGGMember),
				),
			},
			{
				ResourceName:      "pfsense_interface_lagg.live",
				ImportState:       true,
				ImportStateId:     testAccLiveLAGGDescr,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_interface_group
// ---------------------------------------------------------------------------

// Group names are constrained to a short alphanumeric token by pfSense, so the
// `tftest` marker has to lose its underscore here; the description carries the
// full `tftest_` identity. The group is created empty, which the API accepts,
// so no existing interface is drawn into it.
const (
	testAccLiveInterfaceGroupName  = "tftestgrp"
	testAccLiveInterfaceGroupDescr = "tftest_live_group"
)

func testAccInterfaceGroupExists(ifname string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, interfaceGroupPlural, "ifname", ifname)
	if err != nil {
		return false, fmt.Errorf("looking up interface group %q: %w", ifname, err)
	}
	return found, nil
}

func testAccPreCheckLiveInterfaceGroupAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccInterfaceGroupExists(testAccLiveInterfaceGroupName)
	if err != nil {
		t.Fatalf("checking interface group %q does not already exist: %v", testAccLiveInterfaceGroupName, err)
	}
	if found {
		t.Fatalf("interface group %q already exists on the box; remove it before running the live test", testAccLiveInterfaceGroupName)
	}
}

func testAccCheckInterfaceGroupAbsent(ifname string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccInterfaceGroupExists(ifname)
		if err != nil {
			return fmt.Errorf("checking interface group %q was destroyed: %w", ifname, err)
		}
		if found {
			return fmt.Errorf("interface group %q still exists after destroy", ifname)
		}
		return nil
	}
}

// testAccInterfaceGroupLiveConfig renders the group config for one step. The
// empty `members` list is explicit: the API always reports the collection, so
// omitting it would leave a null-versus-empty diff after every apply.
func testAccInterfaceGroupLiveConfig(descr string) string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_interface_group" "live" {
  ifname  = %q
  members = []
  descr   = %q
}
`, testAccLiveInterfaceGroupName, descr)
}

func TestAccInterfaceGroupResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveInterfaceGroupAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckInterfaceGroupAbsent(testAccLiveInterfaceGroupName),
		Steps: []resource.TestStep{
			{
				Config: testAccInterfaceGroupLiveConfig(testAccLiveInterfaceGroupDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "id", testAccLiveInterfaceGroupName),
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "ifname", testAccLiveInterfaceGroupName),
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "members.#", "0"),
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "descr", testAccLiveInterfaceGroupDescr),
				),
			},
			{
				// `ifname` (the natural key) is unchanged; only the description
				// moves.
				Config: testAccInterfaceGroupLiveConfig(testAccLiveInterfaceGroupDescr + " (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "id", testAccLiveInterfaceGroupName),
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "ifname", testAccLiveInterfaceGroupName),
					resource.TestCheckResourceAttr("pfsense_interface_group.live", "descr", testAccLiveInterfaceGroupDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_interface_group.live",
				ImportState:       true,
				ImportStateId:     testAccLiveInterfaceGroupName,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_interface_bridge
// ---------------------------------------------------------------------------

// NOT TESTED LIVE — needs a multi-NIC box.
//
// `members` is required and the API rejects an empty list
// (LENGTH_VALIDATOR_MINIMUM_CONSTRAINT: "Field `members` must contain a minimum
// of 1 entries"). Unlike the LAGG model, which accepts a raw interface name and
// can therefore aggregate a throwaway VLAN clone, the bridge model restricts
// `members` to *assigned* interfaces and answers a VLAN clone with
// FIELD_INVALID_CHOICE: "Field `members` must be one of [wan]".
//
// `wan` is the single physical NIC on the reference box and carries the
// management address these tests connect over, so the only member the API would
// accept is exactly the one that must never be bridged. There is no safe
// configuration to test here until a box with a spare assignable NIC is
// available.

// ---------------------------------------------------------------------------
// pfsense_network_interface
// ---------------------------------------------------------------------------

// Exercised READ-ONLY — full CRUD needs a multi-NIC box.
//
// The resource's Delete calls the API delete with apply=true, which unassigns
// the interface. On the reference box the only assigned interface is `wan` on
// the single physical NIC, and unassigning it drops the management address the
// tests connect over. The usual escape hatch — mutate a cosmetic field and put
// it back — is closed too: the API refuses to write `WAN` back into `descr`
// (FILTER_NAME_VALIDATOR_RESERVED_NAME_USED: "Field `descr` cannot be `WAN`
// because it is a name reserved by the system"), so a description change could
// never be reverted.
//
// What is left is still worth covering: import resolution and the Read mapping.
// The test below therefore has a single ImportState step, which the testing
// framework runs in a throwaway working directory that is discarded without a
// destroy — the test case's own state stays empty, so the post-test destroy is
// a no-op and `wan` is never written to. ImportStateVerify is not usable
// without a prior applied step, so ImportStateCheck compares the imported
// attributes against the live API object instead.

// testAccLiveNetworkInterfaceKey is the natural key the resource resolves on:
// the interfaces model keys on `if`, which holds the physical device name
// (`vtnet0`), while the pfSense assignment name (`wan`) is the model's `id`.
const testAccLiveNetworkInterfaceKey = "vtnet0"

func testAccPreCheckLiveNetworkInterfacePresent(t *testing.T) {
	t.Helper()

	c, err := testAccClient()
	if err != nil {
		t.Fatalf("building verification client: %v", err)
	}
	_, _, found, err := findByKey(context.Background(), c, networkInterfacePlural, "if", testAccLiveNetworkInterfaceKey)
	if err != nil {
		t.Fatalf("looking up network interface %q: %v", testAccLiveNetworkInterfaceKey, err)
	}
	if !found {
		t.Fatalf("network interface %q is not present on the box; the live import test needs it", testAccLiveNetworkInterfaceKey)
	}
}

// testAccCheckNetworkInterfaceImported asserts the imported attributes match
// what the API reports for the same interface, so the check does not hardcode
// this particular box's description or addressing.
func testAccCheckNetworkInterfaceImported(iface string) resource.ImportStateCheckFunc {
	return func(states []*terraform.InstanceState) error {
		if len(states) != 1 {
			return fmt.Errorf("expected 1 imported network interface, got %d", len(states))
		}
		imported := states[0]

		c, err := testAccClient()
		if err != nil {
			return fmt.Errorf("building verification client: %w", err)
		}
		_, obj, found, err := findByKey(context.Background(), c, networkInterfacePlural, "if", iface)
		if err != nil {
			return fmt.Errorf("looking up network interface %q: %w", iface, err)
		}
		if !found {
			return fmt.Errorf("network interface %q disappeared during import", iface)
		}

		if imported.ID != iface {
			return fmt.Errorf("imported id = %q, want %q", imported.ID, iface)
		}
		for _, attr := range []string{"if", "descr", "typev4", "ipaddr", "subnet"} {
			want := objectKey(obj, attr)
			if got := imported.Attributes[attr]; got != want {
				return fmt.Errorf("imported %s = %q, want %q", attr, got, want)
			}
		}
		return nil
	}
}

// testAccNetworkInterfaceLiveConfig declares the resource so `terraform import`
// has an address to import into. It is never applied: the only step is an
// import step, and every required attribute is filled from the live object's
// own values so the block could not drift the box even if it were.
func testAccNetworkInterfaceLiveConfig() string {
	return testAccProviderConfig() + fmt.Sprintf(`
resource "pfsense_network_interface" "live" {
  if                = %q
  descr             = "WAN"
  typev4            = "static"
  ipaddr            = "10.99.0.2"
  subnet            = 24
  ipaddrv6          = "2001:db8:1::1"
  subnetv6          = 64
  prefix_6rd        = "2001:db8:2::/32"
  gateway_6rd       = "192.0.2.254"
  prefix_6rd_v4plen = 0
  track6_interface  = "wan"
}
`, testAccLiveNetworkInterfaceKey)
}

func TestAccNetworkInterfaceResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveNetworkInterfacePresent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:           testAccNetworkInterfaceLiveConfig(),
				ResourceName:     "pfsense_network_interface.live",
				ImportState:      true,
				ImportStateId:    testAccLiveNetworkInterfaceKey,
				ImportStateCheck: testAccCheckNetworkInterfaceImported(testAccLiveNetworkInterfaceKey),
			},
		},
	})
}
