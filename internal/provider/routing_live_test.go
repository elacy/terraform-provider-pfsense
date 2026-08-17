package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// Live CRUD coverage for the routing family (gateways, gateway groups and
// static routes). Everything here runs against a real pfSense box and is gated
// by testAccPreCheck, so `go test ./...` without TF_ACC stays hermetic.
//
// SAFETY. The reference box reaches the network through a single gateway,
// `WANGW` (10.99.0.1 on `wan`), which is also the configured IPv4 default
// gateway and carries the management traffic these tests connect over. Nothing
// below may touch it:
//
//   - Every gateway created here is brand new, `tftest_`-prefixed, and points at
//     a documentation-range address (192.0.2.0/24, 198.51.100.0/24) that no
//     traffic can reach. Monitoring is disabled so dpinger never probes it.
//   - Gateway groups and static routes only ever reference those throwaway
//     gateways, never `WANGW`.
//   - The one static route created covers 198.51.100.0/24, so it can neither
//     shadow 10.99.0.0/24 nor replace the default route.

// ---------------------------------------------------------------------------
// pfsense_routing_gateway
// ---------------------------------------------------------------------------

// testAccLiveGatewayName is the natural key (and Terraform ID) of the throwaway
// gateway exercised with full CRUD.
const (
	testAccLiveGatewayName  = "tftest_gw"
	testAccLiveGatewayDescr = "tftest_live_gateway"
)

func testAccRoutingGatewayExists(name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, routingGatewayPlural, "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up gateway %q: %w", name, err)
	}
	return found, nil
}

func testAccPreCheckLiveRoutingGatewayAbsent(t *testing.T, name string) {
	t.Helper()

	found, err := testAccRoutingGatewayExists(name)
	if err != nil {
		t.Fatalf("checking gateway %q does not already exist: %v", name, err)
	}
	if found {
		t.Fatalf("gateway %q already exists on the box; remove it before running the live test", name)
	}
}

func testAccCheckRoutingGatewayAbsent(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		found, err := testAccRoutingGatewayExists(name)
		if err != nil {
			return fmt.Errorf("checking gateway %q was destroyed: %w", name, err)
		}
		if found {
			return fmt.Errorf("gateway %q still exists after destroy", name)
		}
		return nil
	}
}

// testAccRoutingGatewayBlock renders one throwaway gateway. It is shared by all
// three tests below: the gateway-group and static-route configs need a gateway
// to reference, and only a purpose-built one is safe to point them at.
//
// Every field the API echoes back is pinned explicitly. They are all Optional
// (never Computed), so an omitted one would be null in the config but populated
// with the API default on the next read, leaving a diff that never settles.
// `monitor` is the exception: with `monitor_disable = true` the API reports it
// as null, so leaving it unset is what keeps the plan empty — and it also stops
// dpinger from probing an address that does not exist. `nonlocalgateway` is
// required because the gateway address is a documentation-range address outside
// `wan`'s own subnet.
func testAccRoutingGatewayBlock(label, name, address, descr string) string {
	return fmt.Sprintf(`
resource "pfsense_routing_gateway" %q {
  name                          = %q
  descr                         = %q
  interface                     = "wan"
  ipprotocol                    = "inet"
  gateway                       = %q
  disabled                      = false
  monitor_disable               = true
  action_disable                = false
  force_down                    = false
  dpinger_dont_add_static_route = false
  gw_down_kill_states           = ""
  nonlocalgateway               = true
  weight                        = 1
  data_payload                  = 1
  latencylow                    = 200
  latencyhigh                   = 500
  losslow                       = 10
  losshigh                      = 20
  interval                      = 500
  loss_interval                 = 2000
  time_period                   = 60000
  alert_interval                = 1000
}
`, label, name, descr, address)
}

func testAccRoutingGatewayLiveConfig(address, descr string) string {
	return testAccProviderConfig() + testAccRoutingGatewayBlock("live", testAccLiveGatewayName, address, descr)
}

func TestAccRoutingGatewayResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveRoutingGatewayAbsent(t, testAccLiveGatewayName)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoutingGatewayAbsent(testAccLiveGatewayName),
		Steps: []resource.TestStep{
			{
				Config: testAccRoutingGatewayLiveConfig("192.0.2.1", testAccLiveGatewayDescr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "id", testAccLiveGatewayName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "name", testAccLiveGatewayName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "descr", testAccLiveGatewayDescr),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "interface", "wan"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "ipprotocol", "inet"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "gateway", "192.0.2.1"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "nonlocalgateway", "true"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "monitor_disable", "true"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "weight", "1"),
				),
			},
			{
				// `name` (the natural key) is unchanged, so this is an in-place
				// update; only the far-end address and the description move.
				Config: testAccRoutingGatewayLiveConfig("198.51.100.1", testAccLiveGatewayDescr+" (updated)"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "id", testAccLiveGatewayName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "name", testAccLiveGatewayName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "gateway", "198.51.100.1"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway.live", "descr", testAccLiveGatewayDescr+" (updated)"),
				),
			},
			{
				ResourceName:      "pfsense_routing_gateway.live",
				ImportState:       true,
				ImportStateId:     testAccLiveGatewayName,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_routing_gateway_group
// ---------------------------------------------------------------------------

// `priorities` is required and the API rejects an empty list, so the group needs
// a real gateway to tier. `WANGW` is off limits, so the config creates its own
// throwaway gateway and tiers that instead; Terraform tears the group down
// before the gateway it depends on.
const (
	testAccLiveGatewayGroupName    = "tftest_gwgroup"
	testAccLiveGatewayGroupDescr   = "tftest_live_gateway_group"
	testAccLiveGatewayGroupGateway = "tftest_grp_gw"
)

func testAccRoutingGatewayGroupExists(name string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKey(context.Background(), c, routingGatewayGroupPlural, "name", name)
	if err != nil {
		return false, fmt.Errorf("looking up gateway group %q: %w", name, err)
	}
	return found, nil
}

func testAccPreCheckLiveRoutingGatewayGroupAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccRoutingGatewayGroupExists(testAccLiveGatewayGroupName)
	if err != nil {
		t.Fatalf("checking gateway group %q does not already exist: %v", testAccLiveGatewayGroupName, err)
	}
	if found {
		t.Fatalf("gateway group %q already exists on the box; remove it before running the live test", testAccLiveGatewayGroupName)
	}

	testAccPreCheckLiveRoutingGatewayAbsent(t, testAccLiveGatewayGroupGateway)
}

func testAccCheckRoutingGatewayGroupAbsent(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		found, err := testAccRoutingGatewayGroupExists(name)
		if err != nil {
			return fmt.Errorf("checking gateway group %q was destroyed: %w", name, err)
		}
		if found {
			return fmt.Errorf("gateway group %q still exists after destroy", name)
		}
		return testAccCheckRoutingGatewayAbsent(testAccLiveGatewayGroupGateway)(s)
	}
}

// testAccRoutingGatewayGroupLiveConfig renders the group (and the throwaway
// gateway it tiers) for one step. `virtual_ip` is pinned to the API default
// `address` because the API reports it on every read.
func testAccRoutingGatewayGroupLiveConfig(trigger, descr string, tier int) string {
	return testAccProviderConfig() +
		testAccRoutingGatewayBlock("group_member", testAccLiveGatewayGroupGateway, "192.0.2.10", "tftest_live_group_member") +
		fmt.Sprintf(`
resource "pfsense_routing_gateway_group" "live" {
  name    = %q
  trigger = %q
  descr   = %q

  priorities = [
    {
      gateway    = pfsense_routing_gateway.group_member.name
      tier       = %d
      virtual_ip = "address"
    },
  ]
}
`, testAccLiveGatewayGroupName, trigger, descr, tier)
}

func TestAccRoutingGatewayGroupResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveRoutingGatewayGroupAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoutingGatewayGroupAbsent(testAccLiveGatewayGroupName),
		Steps: []resource.TestStep{
			{
				Config: testAccRoutingGatewayGroupLiveConfig("down", testAccLiveGatewayGroupDescr, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "id", testAccLiveGatewayGroupName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "name", testAccLiveGatewayGroupName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "trigger", "down"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "descr", testAccLiveGatewayGroupDescr),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "priorities.#", "1"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "priorities.0.gateway", testAccLiveGatewayGroupGateway),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "priorities.0.tier", "1"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "priorities.0.virtual_ip", "address"),
				),
			},
			{
				// `name` (the natural key) is unchanged; the trigger level, the
				// description and the member's tier move.
				Config: testAccRoutingGatewayGroupLiveConfig("downloss", testAccLiveGatewayGroupDescr+" (updated)", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "id", testAccLiveGatewayGroupName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "name", testAccLiveGatewayGroupName),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "trigger", "downloss"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "descr", testAccLiveGatewayGroupDescr+" (updated)"),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "priorities.0.gateway", testAccLiveGatewayGroupGateway),
					resource.TestCheckResourceAttr("pfsense_routing_gateway_group.live", "priorities.0.tier", "2"),
				),
			},
			{
				ResourceName:      "pfsense_routing_gateway_group.live",
				ImportState:       true,
				ImportStateId:     testAccLiveGatewayGroupName,
				ImportStateVerify: true,
			},
		},
	})
}

// ---------------------------------------------------------------------------
// pfsense_routing_static_route
// ---------------------------------------------------------------------------

// Static routes have no unique name in the API; the resource keys them on
// network|gateway, which is also the Terraform ID and the import ID. Neither
// component can carry the `tftest_` marker (one is a CIDR, the other a gateway
// name resolved by the API), so the description carries it instead.
//
// The route covers a documentation range and points at its own throwaway
// gateway, so it can neither shadow the management network nor divert the
// default route.
const (
	testAccLiveStaticRouteNetwork = "198.51.100.0/24"
	testAccLiveStaticRouteGateway = "tftest_rt_gw"
	testAccLiveStaticRouteID      = testAccLiveStaticRouteNetwork + "|" + testAccLiveStaticRouteGateway
	testAccLiveStaticRouteDescr   = "tftest_live_static_route"
)

func testAccRoutingStaticRouteExists(network, gateway string) (bool, error) {
	c, err := testAccClient()
	if err != nil {
		return false, fmt.Errorf("building verification client: %w", err)
	}
	_, _, found, err := findByKeys(context.Background(), c, routingStaticRoutePlural, map[string]string{
		"network": network,
		"gateway": gateway,
	})
	if err != nil {
		return false, fmt.Errorf("looking up static route %q: %w", network+"|"+gateway, err)
	}
	return found, nil
}

func testAccPreCheckLiveRoutingStaticRouteAbsent(t *testing.T) {
	t.Helper()

	found, err := testAccRoutingStaticRouteExists(testAccLiveStaticRouteNetwork, testAccLiveStaticRouteGateway)
	if err != nil {
		t.Fatalf("checking static route %q does not already exist: %v", testAccLiveStaticRouteID, err)
	}
	if found {
		t.Fatalf("static route %q already exists on the box; remove it before running the live test", testAccLiveStaticRouteID)
	}

	testAccPreCheckLiveRoutingGatewayAbsent(t, testAccLiveStaticRouteGateway)
}

func testAccCheckRoutingStaticRouteAbsent(network, gateway string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		found, err := testAccRoutingStaticRouteExists(network, gateway)
		if err != nil {
			return fmt.Errorf("checking static route %q was destroyed: %w", network+"|"+gateway, err)
		}
		if found {
			return fmt.Errorf("static route %q still exists after destroy", network+"|"+gateway)
		}
		return testAccCheckRoutingGatewayAbsent(testAccLiveStaticRouteGateway)(s)
	}
}

// testAccRoutingStaticRouteLiveConfig renders the route (and the throwaway
// gateway it points at) for one step.
func testAccRoutingStaticRouteLiveConfig(descr string, disabled bool) string {
	return testAccProviderConfig() +
		testAccRoutingGatewayBlock("route_gw", testAccLiveStaticRouteGateway, "192.0.2.20", "tftest_live_route_gateway") +
		fmt.Sprintf(`
resource "pfsense_routing_static_route" "live" {
  network  = %q
  gateway  = pfsense_routing_gateway.route_gw.name
  descr    = %q
  disabled = %t
}
`, testAccLiveStaticRouteNetwork, descr, disabled)
}

func TestAccRoutingStaticRouteResourceLive(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckLiveRoutingStaticRouteAbsent(t)
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRoutingStaticRouteAbsent(testAccLiveStaticRouteNetwork, testAccLiveStaticRouteGateway),
		Steps: []resource.TestStep{
			{
				Config: testAccRoutingStaticRouteLiveConfig(testAccLiveStaticRouteDescr, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "id", testAccLiveStaticRouteID),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "network", testAccLiveStaticRouteNetwork),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "gateway", testAccLiveStaticRouteGateway),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "descr", testAccLiveStaticRouteDescr),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "disabled", "false"),
				),
			},
			{
				// Both key components (`network` and `gateway`) are unchanged, so
				// this is an in-place update; only the description and the
				// disabled flag move.
				Config: testAccRoutingStaticRouteLiveConfig(testAccLiveStaticRouteDescr+" (updated)", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "id", testAccLiveStaticRouteID),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "network", testAccLiveStaticRouteNetwork),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "gateway", testAccLiveStaticRouteGateway),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "descr", testAccLiveStaticRouteDescr+" (updated)"),
					resource.TestCheckResourceAttr("pfsense_routing_static_route.live", "disabled", "true"),
				),
			},
			{
				ResourceName:      "pfsense_routing_static_route.live",
				ImportState:       true,
				ImportStateId:     testAccLiveStaticRouteID,
				ImportStateVerify: true,
			},
		},
	})
}
