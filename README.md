# Terraform Provider for pfSense (REST API v2)

A [Terraform](https://www.terraform.io) provider for pfSense, built against the
**pfSense REST API v2** ([`pfrest/pfSense-pkg-RESTAPI`](https://github.com/pfrest/pfSense-pkg-RESTAPI)).

This provider supersedes the earlier `elacy/terraform-provider-pfsense` (v1 API,
`jaredhendrickson13/pfsense-api`). It targets the `pfrest` v2 package (latest
release **v2.10.0**) which exposes 200+ endpoints under `/api/v2`.

## Status

Code-complete: **62 resources + 4 data sources** covering firewall, NAT,
interfaces, routing, services (DHCP/DNS/NTP/cron/BIND/FreeRADIUS), system, and
VPN (IPsec/OpenVPN/WireGuard). Every resource has an acceptance test that runs
against an in-process mock server (no pfSense required).

Live acceptance testing against a pfSense CE 2.8.1 VM is the final milestone —
see [`docs/COVERAGE.md`](docs/COVERAGE.md) for the roadmap and current status.

## Requirements

- [Go](https://go.dev/) 1.26+
- [Terraform](https://www.terraform.io/) 1.15+
- pfSense CE 2.8.1 / Plus 25.11.1+ with the `pfSense-pkg-RESTAPI` package
  installed and enabled.

## Authentication

The API supports three authentication methods. Configure exactly one:

| Method | Provider attributes |
| --- | --- |
| Basic (local user) | `username`, `password` |
| API key | `api_key` |
| JWT (Basic exchange) | `username`, `password`, `use_jwt = true` |

```hcl
provider "pfsense" {
  url      = "https://192.168.1.1"
  username = "admin"
  password = var.pfsense_password

  # Self-signed certificate:
  skip_tls_verify = true
}
```

## Example

```hcl
resource "pfsense_firewall_alias" "webservers" {
  name    = "webservers"
  type    = "host"
  address = ["10.0.0.10", "10.0.0.11", "10.0.0.12"]
}

resource "pfsense_firewall_rule" "allow_https" {
  descr            = "allow-https"
  type             = "pass"
  interface        = ["lan"]
  ipprotocol       = "inet"
  protocol         = "tcp"
  source           = "any"
  destination      = "webservers"
  destination_port = "443"
}

resource "pfsense_system_user" "deploy" {
  name     = "deploy"
  password = var.deploy_password
  priv     = ["page-all"]
}
```

See [`examples/`](examples/) for more.

## Resources (62)

Firewall:

- `pfsense_firewall_alias`
- `pfsense_firewall_rule`
- `pfsense_firewall_schedule`
- `pfsense_firewall_nat_port_forward`
- `pfsense_firewall_nat_one_to_one`
- `pfsense_firewall_nat_outbound`
- `pfsense_firewall_virtual_ip`
- `pfsense_firewall_traffic_shaper`
- `pfsense_firewall_traffic_shaper_limiter`
- `pfsense_firewall_traffic_shaper_queue`
- `pfsense_firewall_traffic_shaper_limiter_queue`

Interfaces:

- `pfsense_network_interface`
- `pfsense_interface_vlan`
- `pfsense_interface_bridge`
- `pfsense_interface_gre`
- `pfsense_interface_lagg`
- `pfsense_interface_group`

Routing:

- `pfsense_routing_gateway`
- `pfsense_routing_gateway_group`
- `pfsense_routing_static_route`

Services (DHCP / DNS / NTP / misc):

- `pfsense_services_dhcp_server`
- `pfsense_services_dhcp_static_mapping`
- `pfsense_services_dhcp_address_pool`
- `pfsense_services_dhcp_custom_option`
- `pfsense_services_dns_resolver_host_override`
- `pfsense_services_dns_resolver_host_override_alias`
- `pfsense_services_dns_resolver_domain_override`
- `pfsense_services_dns_forwarder_host_override`
- `pfsense_services_dns_forwarder_host_override_alias`
- `pfsense_services_ntp_settings`
- `pfsense_services_ntp_time_server`
- `pfsense_services_cron_job`
- `pfsense_services_service_watchdog`
- `pfsense_services_bind_access_list`
- `pfsense_services_bind_view`
- `pfsense_services_bind_zone`
- `pfsense_services_freeradius_mac`
- `pfsense_services_freeradius_user`

System:

- `pfsense_system_user`
- `pfsense_system_group`
- `pfsense_system_ca`
- `pfsense_system_certificate`
- `pfsense_system_crl`
- `pfsense_system_crl_revoked_certificate`
- `pfsense_system_tunable`
- `pfsense_system_hostname`
- `pfsense_system_dns`
- `pfsense_system_timezone`
- `pfsense_system_package`
- `pfsense_user_auth_server`
- `pfsense_system_restapi_access_list_entry`

VPN:

- `pfsense_ipsec_phase1`
- `pfsense_ipsec_phase1_encryption`
- `pfsense_ipsec_phase2`
- `pfsense_ipsec_phase2_encryption`
- `pfsense_openvpn_client`
- `pfsense_openvpn_server`
- `pfsense_openvpn_cso`
- `pfsense_wireguard_tunnel`
- `pfsense_wireguard_tunnel_address`
- `pfsense_wireguard_peer`
- `pfsense_wireguard_peer_allowed_ip`

## Data sources

- `pfsense_firewall_aliases`
- `pfsense_interfaces`
- `pfsense_routing_gateways`
- `pfsense_system_certificates`

## Object identity (important)

pfSense does **not** use persistent object IDs: the REST API identifies objects
by array index, which shifts when objects are reordered or deleted. This
provider therefore identifies most resources by a **natural key** (a unique
field such as `name`, or a composite of fields), which is stored as the
Terraform state ID and re-resolved to the current index on every operation.

- Firewall rules require a **unique `descr`** (used as the natural key).
- Static routes are identified by `network` + `gateway`.
- Host overrides by `host` + `domain`; VLANs by parent `if` + `tag`.
- CAs and certificates are identified by their persistent `refid`.

## Development

```sh
make build       # build the provider binary
make test        # run unit tests
make testacc     # run acceptance tests (mock server; requires TF_ACC=1)
make fmt         # gofmt
make vet         # go vet
```

### Running acceptance tests

Acceptance tests run against an in-process mock server (no pfSense required):

```sh
TF_ACC=1 go test ./... -run TestAcc -v
```

Live acceptance testing against a real pfSense VM is planned; see
[`docs/COVERAGE.md`](docs/COVERAGE.md) for the roadmap and how to run it.

## License

The upstream `pfSense-pkg-RESTAPI` is Apache-2.0. This provider's license is to
be finalized before the first public release.
