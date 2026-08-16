# Resource coverage & roadmap

This file tracks coverage against the pfSense REST API v2
(`pfrest/pfSense-pkg-RESTAPI` v2.10.0).

## Implemented — 62 resources + 4 data sources

| Terraform resource | API model | ID strategy |
| --- | --- | --- |
| `pfsense_firewall_alias` | `FirewallAlias` | natural key: `name` |
| `pfsense_firewall_rule` | `FirewallRule` | natural key: unique `descr` |
| `pfsense_firewall_schedule` | `FirewallSchedule` | natural key: `name` |
| `pfsense_firewall_nat_port_forward` | `PortForward` | natural key: `descr` |
| `pfsense_firewall_nat_one_to_one` | `OneToOneNAT` | natural key: `descr` |
| `pfsense_firewall_nat_outbound` | `OutboundNAT` | natural key: `descr` |
| `pfsense_firewall_virtual_ip` | `VirtualIP` | natural key: `descr` |
| `pfsense_firewall_traffic_shaper` | `TrafficShaper` | natural key: `name` |
| `pfsense_firewall_traffic_shaper_limiter` | `Limiter` | natural key: `name` |
| `pfsense_firewall_traffic_shaper_queue` | `Queue` | natural key: `name` |
| `pfsense_firewall_traffic_shaper_limiter_queue` | `LimiterQueue` | natural key: `name` |
| `pfsense_network_interface` | `NetworkInterface` | natural key: `if` |
| `pfsense_interface_vlan` | `InterfaceVLAN` | composite: `if` + `tag` |
| `pfsense_interface_bridge` | `InterfaceBridge` | natural key: `bridgeif` |
| `pfsense_interface_gre` | `InterfaceGRE` | natural key: `greif` |
| `pfsense_interface_lagg` | `InterfaceLAGG` | natural key: `laggif` |
| `pfsense_interface_group` | `InterfaceGroup` | natural key: `name` |
| `pfsense_routing_gateway` | `RoutingGateway` | natural key: `name` |
| `pfsense_routing_gateway_group` | `RoutingGatewayGroup` | natural key: `name` |
| `pfsense_routing_static_route` | `StaticRoute` | composite: `network` + `gateway` |
| `pfsense_services_dhcp_server` | `DHCPServer` | natural key: `interface` |
| `pfsense_services_dhcp_static_mapping` | `DHCPStaticMapping` | parent `interface` + `mac` |
| `pfsense_services_dhcp_address_pool` | `DHCPAddressPool` | parent `interface` + range |
| `pfsense_services_dhcp_custom_option` | `DHCPCustomOption` | parent `interface` + `number` |
| `pfsense_services_dns_resolver_host_override` | `DNSResolverHostOverride` | composite: `host` + `domain` |
| `pfsense_services_dns_resolver_host_override_alias` | `DNSResolverHostOverrideAlias` | parent host/domain + `host` |
| `pfsense_services_dns_resolver_domain_override` | `DNSResolverDomainOverride` | natural key: `domain` |
| `pfsense_services_dns_forwarder_host_override` | `DNSForwarderHostOverride` | composite: `host` + `domain` |
| `pfsense_services_dns_forwarder_host_override_alias` | `DNSForwarderHostOverrideAlias` | parent host/domain + `host` |
| `pfsense_services_ntp_settings` | `NTPSettings` | singleton |
| `pfsense_services_ntp_time_server` | `NTPTimeServer` | natural key: `timeserver` |
| `pfsense_services_cron_job` | `CronJob` | composite: full schedule |
| `pfsense_services_service_watchdog` | `ServiceWatchdog` | natural key: `name` |
| `pfsense_services_bind_access_list` | `BINDAccessList` | natural key: `name` |
| `pfsense_services_bind_view` | `BINDView` | natural key: `name` |
| `pfsense_services_bind_zone` | `BINDZone` | natural key: `name` |
| `pfsense_services_freeradius_mac` | `FreeRADIUSMAC` | natural key: `mac` |
| `pfsense_services_freeradius_user` | `FreeRADIUSUser` | natural key: `username` |
| `pfsense_system_user` | `User` | natural key: `name` |
| `pfsense_system_group` | `UserGroup` | natural key: `name` |
| `pfsense_system_ca` | `CertificateAuthority` | persistent `refid` |
| `pfsense_system_certificate` | `Certificate` | persistent `refid` |
| `pfsense_system_crl` | `CertificateRevocationList` | natural key: `descr` |
| `pfsense_system_crl_revoked_certificate` | `CRLRevokedCertificate` | parent `descr` + `certref` |
| `pfsense_system_tunable` | `SystemTunable` | natural key: `tunable` |
| `pfsense_system_hostname` | `SystemHostname` | singleton |
| `pfsense_system_dns` | `SystemDNS` | singleton |
| `pfsense_system_timezone` | `SystemTimezone` | singleton |
| `pfsense_system_package` | `Package` | natural key: `name` |
| `pfsense_user_auth_server` | `AuthServer` | natural key: `name` |
| `pfsense_system_restapi_access_list_entry` | `RESTAPIAccessListEntry` | natural key: `network` |
| `pfsense_ipsec_phase1` | `IPsecPhase1` | natural key: `descr` |
| `pfsense_ipsec_phase1_encryption` | `IPsecPhase1Encryption` | parent `descr` + alg/hash/dhgroup/keylen |
| `pfsense_ipsec_phase2` | `IPsecPhase2` | natural key: `descr` |
| `pfsense_ipsec_phase2_encryption` | `IPsecPhase2Encryption` | parent `descr` + `name` |
| `pfsense_openvpn_client` | `OpenVPNClient` | natural key: `description` |
| `pfsense_openvpn_server` | `OpenVPNServer` | natural key: `description` |
| `pfsense_openvpn_cso` | `OpenVPNClientSpecificOverride` | natural key: `common_name` |
| `pfsense_wireguard_tunnel` | `WireGuardTunnel` | natural key: `name` |
| `pfsense_wireguard_tunnel_address` | `WireGuardTunnelAddress` | parent `name` + `address` |
| `pfsense_wireguard_peer` | `WireGuardPeer` | natural key: `descr` |
| `pfsense_wireguard_peer_allowed_ip` | `WireGuardPeerAllowedIP` | parent `descr` + `address` |

## Data sources

`pfsense_firewall_aliases`, `pfsense_interfaces`, `pfsense_routing_gateways`,
`pfsense_system_certificates`.

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

In progress. A pfSense test VM (`pfsensetest`) exists on the TrueNAS NAS
(TrueNAS SCALE). Status as of the latest session:

- pfSense CE **2.7.2** is installed and reachable (web GUI + SSH; admin password
  reset to the default `pfsense`, SSH enabled).
- The target version is **CE 2.8.1**, but Netgate no longer publishes CE 2.8.x
  ISO/memstick images — the only install path is the online **Netgate Installer**
  (which the VM now boots from a CD-ROM) or an in-place upgrade from 2.7.2.
- The VM's network is **isolated** (bridge `br0`, 10.99.0.0/24) with no internet
  path; the second NIC (physical LAN `enp131s0f0`) also gets no DHCP. Giving the
  VM internet (NAS outbound NAT, or a proxy on the Hermes container at
  `172.16.26.2:18081`) is the open item.

Run against the VM once reachable:

```sh
TF_ACC=1 PFSENSE_URL=https://<vm-ip> PFSENSE_USERNAME=admin PFSENSE_PASSWORD=... \
  go test ./... -run TestAcc -v
```
