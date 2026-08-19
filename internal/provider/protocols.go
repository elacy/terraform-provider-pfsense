package provider

// Protocol value sets for the firewall resources. pfSense builds these from
// get_ipprotocols() in src/etc/inc/filter.inc, and the set differs by
// endpoint, so each resource keeps its own list rather than sharing one:
//
//   - Filter rules (pfsense_firewall_rule) accept the full set, including
//     "any" and the tunnel/keepalive protocols sctp, carp and pfsync.
//   - NAT port forwards accept "any" but not the tunnel/keepalive protocols.
//   - Outbound NAT accepts neither "any" nor the tunnel/keepalive protocols.

// firewallRuleProtocols is the protocol set for pfsense_firewall_rule.
var firewallRuleProtocols = []string{
	"any", "tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "igmp",
	"pim", "ospf", "sctp", "ipv6", "carp", "pfsync",
}

// natPortForwardProtocols is the protocol set for
// pfsense_firewall_nat_port_forward.
var natPortForwardProtocols = []string{
	"any", "tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "ipv6",
	"igmp", "pim", "ospf",
}

// natOutboundProtocols is the protocol set for pfsense_firewall_nat_outbound.
var natOutboundProtocols = []string{
	"tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "ipv6",
	"igmp", "pim", "ospf",
}
