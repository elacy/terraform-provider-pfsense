# Resource expansion spec — terraform-provider-pfsense v2

You are extending the pfsense v2 provider at:
`/opt/data/kanban/workspaces/t_8227b5da/terraform-provider-pfsense`

## Environment (run before any go command)
```
export PATH="/opt/data/toolchain/go/bin:/opt/data/bin:$PATH"
export GOPATH=/opt/data/work/gopath GOMODCACHE=/opt/data/work/gopath/pkg/mod GOCACHE=/opt/data/work/gocache
cd /opt/data/kanban/workspaces/t_8227b5da/terraform-provider-pfsense
```

## Pattern (READ THESE FIRST, they are authoritative)
- `internal/provider/firewall_alias_resource.go` — the canonical resource template (model struct, Schema, Create/Read/Update/Delete/ImportState, natural-key resolution).
- `internal/provider/helpers.go` — all shared helpers: `findByKey`, `findByKeys`, `formatID`, `setString/setBool/setInt/setStringList/setStringSet`, `getString/getBool/getInt/getStringSlice/getSliceMap`, `strValue/boolValue/intValue/strListValue`, `ptrString/ptrBool/ptrInt`, `applyNow`, `decodeObject`.
- `internal/provider/configure.go` — `resourceClient(ctx, req, resp)`.
- `internal/client/client.go` — `c.List/Get/Create/Update/Delete/Apply`, `client.Query{}.Set(...)`.
- `internal/provider/schema_helpers.go` and `nested_helpers.go` — extra helpers (read them).
- Field names/types/enums: `/opt/data/work/tfp-recon/schema_meta.json` (per model). API URL paths + parent info: `/opt/data/work/tfp-recon/pfrest/pfSense-pkg-RESTAPI/files/usr/local/pkg/RESTAPI/Endpoints/*.inc`.

## Rules
1. Write ONE new file per category (see below). Do NOT edit provider.go, helpers.go, or any existing file. I will register the new resources in provider.go myself.
2. Terraform type name = `pfsense_<snake>` (e.g. `pfsense_nat_port_forward`, `pfsense_interface_bridge`, `pfsense_wireguard_tunnel`).
3. Natural key: use `name` if the model has one; else `descr`; else the composite of the identity fields. For parent-child resources (parent_id in the POST body + `parent_id` query param on GET/DELETE), the resource also has a required `parent_id` attribute (the parent's natural-key value) and the natural key is scoped within the parent.
4. Create/Update payloads are `map[string]any` built with the `set*` helpers, plus `"id": <resolved index>` on Update, plus `applyNow(payload)` for firewall/interface/routing/dhcp/dns/vpn models. Parent-child payloads must include `"parent_id": <parent value>`.
5. Read/Delete resolve the object with `findByKey` (or `findByKeys` for composite keys) and use `formatID(id)` for the `?id=` query.
6. Register each new resource constructor in the file, following `NewFirewallAliasResource` naming.
7. After EACH file: `gofmt -w <file> && go build ./...` must be clean. Do not leave the package broken.

## Categories + resources (write these files)

### 1. internal/provider/nat_resources.go
- pfsense_nat_port_forward  (model PortForward)  /api/v2/firewall/nat/port_forward  (singular+plural same). Natural key: composite `interface|protocol|destination|target` — use findByKeys.
- pfsense_nat_one_to_one    (OneToOneNATMapping)  /api/v2/firewall/nat/one_to_one/mapping
- pfsense_nat_outbound      (OutboundNATMapping)  /api/v2/firewall/nat/outbound/mapping

### 2. internal/provider/shaper_resources.go
- pfsense_firewall_virtual_ip (VirtualIP) /api/v2/firewall/virtual_ip — key: `descr`.
- pfsense_firewall_traffic_shaper (TrafficShaper) /api/v2/firewall/traffic_shaper
- pfsense_firewall_traffic_shaper_limiter (TrafficShaperLimiter) /api/v2/firewall/traffic_shaper/limiter — key: `name`.
- pfsense_firewall_traffic_shaper_queue (TrafficShaperQueue) /api/v2/firewall/traffic_shaper/queue — parent_id (limiter name), key: `name`.
- pfsense_firewall_traffic_shaper_limiter_queue (TrafficShaperLimiterQueue) /api/v2/firewall/traffic_shaper/limiter/queue — parent_id, key: `name`.

### 3. internal/provider/interface_resources.go
- pfsense_network_interface (NetworkInterface) /api/v2/interface — id is a STRING (interface name, persistent). Natural key = the `if` field value (the interface ID itself). Use `if` as the identity.
- pfsense_interface_bridge (InterfaceBridge) /api/v2/interface/bridge — key: `descr` (or `bridgeif`).
- pfsense_interface_gre (InterfaceGRE) /api/v2/interface/gre — key: `if`.
- pfsense_interface_lagg (InterfaceLAGG) /api/v2/interface/lagg — key: `laggif`/`descr`.
- pfsense_interface_group (InterfaceGroup) /api/v2/interface/group — key: `ifname`.

### 4. internal/provider/services_extra_resources.go
- pfsense_dhcp_static_mapping (DHCPServerStaticMapping) /api/v2/services/dhcp_server/static_mapping — parent_id = dhcp server interface name; key: `mac`.
- pfsense_dhcp_address_pool (DHCPServerAddressPool) /api/v2/services/dhcp_server/address_pool — parent_id; key: `range_from`.
- pfsense_dhcp_custom_option (DHCPServerCustomOption) /api/v2/services/dhcp_server/custom_option — parent_id; key: `number`.
- pfsense_dns_resolver_domain_override (DNSResolverDomainOverride) /api/v2/services/dns_resolver/domain_override — key: `domain`.
- pfsense_dns_resolver_host_override_alias (DNSResolverHostOverrideAlias) /api/v2/services/dns_resolver/host_override/alias — parent_id = host override domain; key: `host`.
- pfsense_dns_forwarder_host_override (DNSForwarderHostOverride) /api/v2/services/dns_forwarder/host_override — key: `domain`.
- pfsense_dns_forwarder_host_override_alias (DNSForwarderHostOverrideAlias) /api/v2/services/dns_forwarder/host_override/alias — parent_id; key: `host`.
- pfsense_ntp_time_server (NTPTimeServer) /api/v2/services/ntp/time_server — key: `timeserver`.
- pfsense_service_watchdog (ServiceWatchdog) /api/v2/services/service_watchdog — key: `name`.

### 5. internal/provider/system_extra_resources.go
- pfsense_system_crl (CertificateRevocationList) /api/v2/system/crl — key: `descr`.
- pfsense_system_crl_revoked_certificate (CertificateRevocationListRevokedCertificate) /api/v2/system/crl/revoked_certificate — parent_id = CRL descr; key: `certref`.
- pfsense_system_package (Package) /api/v2/system/package — key: `name`.
- pfsense_user_auth_server (AuthServer) /api/v2/user/auth_server — key: `name`.
- pfsense_system_restapi_access_list_entry (RESTAPIAccessListEntry) /api/v2/system/restapi/access_list/entry — key: `network`.

### 6. internal/provider/vpn_resources.go
- pfsense_ipsec_phase1 (IPsecPhase1) /api/v2/vpn/ipsec/phase1 — key: `descr`.
- pfsense_ipsec_phase1_encryption (IPsecPhase1Encryption) /api/v2/vpn/ipsec/phase1/encryption — parent_id = phase1 descr; key: composite `encryption_algorithm_name|hash_algorithm|dhgroup`.
- pfsense_ipsec_phase2 (IPsecPhase2) /api/v2/vpn/ipsec/phase2 — key: `descr`.
- pfsense_ipsec_phase2_encryption (IPsecPhase2Encryption) /api/v2/vpn/ipsec/phase2/encryption — parent_id = phase2 descr; key: `name`.
- pfsense_openvpn_client (OpenVPNClient) /api/v2/vpn/openvpn/client — key: `descr`.
- pfsense_openvpn_server (OpenVPNServer) /api/v2/vpn/openvpn/server — key: `descr`.
- pfsense_openvpn_cso (OpenVPNClientSpecificOverride) /api/v2/vpn/openvpn/cso — key: `common_name`.
- pfsense_wireguard_tunnel (WireGuardTunnel) /api/v2/vpn/wireguard/tunnel — key: `name`.
- pfsense_wireguard_tunnel_address (WireGuardTunnelAddress) /api/v2/vpn/wireguard/tunnel/address — parent_id = tunnel name; key: `address`.
- pfsense_wireguard_peer (WireGuardPeer) /api/v2/vpn/wireguard/peer — key: `descr`.
- pfsense_wireguard_peer_allowed_ip (WireGuardPeerAllowedIP) /api/v2/vpn/wireguard/peer/allowed_ip — parent_id = peer descr; key: `address`.

### 7. internal/provider/bind_freeradius_resources.go  (only if time permits — lower priority)
- BIND (access_list, view, zone, zone/record, access_list/entry, sync/remote_host) + FreeRADIUS (client, interface, mac, user). Use `name`/`descr` keys. /api/v2/services/bind/* and /api/v2/services/freeradius/*.

## Do NOT implement (action/export endpoints that don't map to a resource):
- /services/acme/certificate/action, /vpn/openvpn/client_export/config, /services/haproxy/*/action (HAProxy action rules are out of scope for now), ACME account_key/certificate (acme is a plugin, low value).

## Verification
- After all files: `gofmt -l .` should be empty; `go build ./...` clean; `go test ./...` passes (existing tests). Report the list of resource type names you added + the file each lives in + build/test results. Do NOT edit provider.go — I register them.
