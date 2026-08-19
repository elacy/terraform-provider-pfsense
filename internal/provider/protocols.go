package provider

// firewallProtocols is the single source of truth for the protocol values the
// pfSense firewall accepts. Filter rules and NAT rules share the same enum in
// the API, so both resources validate against this one slice rather than
// maintaining divergent hand-written allowlists (which previously caused one
// resource to reject values the other accepted).
var firewallProtocols = []string{
	"any", "tcp", "udp", "tcp/udp", "icmp", "esp", "ah", "gre", "igmp",
	"pim", "ospf", "sctp", "ipv6", "carp", "pfsync",
}
