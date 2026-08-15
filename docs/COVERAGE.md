# Resource coverage & roadmap

This file tracks coverage against the pfSense REST API v2 (`pfrest/pfSense-pkg-RESTAPI`
v2.10.0) and the plan to reach full coverage.

## Implemented (this milestone)

| Terraform resource | API model | ID strategy |
| --- | --- | --- |
| `pfsense_firewall_alias` | `FirewallAlias` | natural key: `name` |
| `pfsense_firewall_rule` | `FirewallRule` | natural key: unique `descr` |
| `pfsense_firewall_schedule` | `FirewallSchedule` | natural key: `name` |
| `pfsense_interface_vlan` | `InterfaceVLAN` | composite: `if` + `tag` |
| `pfsense_routing_gateway` | `RoutingGateway` | natural key: `name` |
| `pfsense_routing_gateway_group` | `RoutingGatewayGroup` | natural key: `name` |
| `pfsense_routing_static_route` | `StaticRoute` | composite: `network` + `gateway` |
| `pfsense_services_dhcp_server` | `DHCPServer` | natural key: `interface` |
| `pfsense_services_dns_resolver_host_override` | `DNSResolverHostOverride` | composite: `host` + `domain` |
| `pfsense_services_cron_job` | `CronJob` | composite: full schedule |
| `pfsense_services_ntp_settings` | `NTPSettings` | singleton |
| `pfsense_system_user` | `User` | natural key: `name` |
| `pfsense_system_group` | `UserGroup` | natural key: `name` |
| `pfsense_system_ca` | `CertificateAuthority` | persistent `refid` |
| `pfsense_system_certificate` | `Certificate` | persistent `refid` |
| `pfsense_system_tunable` | `SystemTunable` | natural key: `tunable` |
| `pfsense_system_hostname` | `SystemHostname` | singleton |
| `pfsense_system_dns` | `SystemDNS` | singleton |
| `pfsense_system_timezone` | `SystemTimezone` | singleton |

## Data sources

`pfsense_firewall_aliases`, `pfsense_interfaces`, `pfsense_routing_gateways`,
`pfsense_system_certificates`.

## Next up (mapped, not yet implemented)

- `pfsense_network_interface` (`NetworkInterface`) — string ID (`wan`, `lan`, …).
- `pfsense_interface_bridge` (`InterfaceBridge`).
- `pfsense_interface_gre`, `pfsense_interface_lagg`, `pfsense_interface_group`.
- `pfsense_firewall_nat_port_forward` (`PortForward`), 1:1, outbound mappings + mode.
- `pfsense_firewall_virtual_ip` (`VirtualIP`).
- `pfsense_firewall_traffic_shaper` + limiters/queues (`TrafficShaper*`).
- `pfsense_services_dhcp_relay` (`DHCPRelay`), DHCP static mapping / pool /
  custom option as top-level resources.
- `pfsense_services_dns_resolver_*` remaining (domain override, ACL, settings),
  DNS forwarder overrides.
- `pfsense_services_ntp_time_server` (`NTPTimeServer`), service watchdog.
- `pfsense_system_crl` + revoked cert, `pfsense_system_csr`, auth servers,
  console, webgui settings, packages.
- VPN: IPsec phase1/phase2, OpenVPN client/server/CSO, WireGuard tunnel/peer.

## Design notes

- **ID strategy**: array-index IDs are not persistent; each resource uses a
  natural key (or the persistent `refid` for certs) resolved via the plural
  endpoint's query filters (`field__exact`).
- **Apply semantics**: mutations carry the `apply` control parameter in the JSON
  body so changes are pushed to the running system (filter reload, service
  restart) within the same request. Models that always apply ignore it.
- **Auth**: basic, `X-API-Key`, and JWT (with transparent 401-refresh) are
  implemented in `internal/client`.
- **Content types**: POST/PATCH use JSON (id + control params in the body);
  GET/DELETE use the URL query string (the API defaults to
  `x-www-form-urlencoded` for those methods).

## Live acceptance testing

Planned: provision pfSense CE 2.8.1 on the TrueNAS NAS with an **isolated**
network (no rogue DHCP), install `pfSense-pkg-RESTAPI`, then run
`TF_ACC=1 go test ./... -run TestAcc` with real credentials.
