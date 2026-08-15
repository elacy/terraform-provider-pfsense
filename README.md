# Terraform Provider for pfSense (REST API v2)

A [Terraform](https://www.terraform.io) provider for pfSense, built against the
**pfSense REST API v2** ([`pfrest/pfSense-pkg-RESTAPI`](https://github.com/pfrest/pfSense-pkg-RESTAPI)).

This provider supersedes the earlier `elacy/terraform-provider-pfsense` (v1 API,
`jaredhendrickson13/pfsense-api`). It targets the `pfrest` v2 package (latest
release **v2.10.0**) which exposes 200+ endpoints under `/api/v2`.

## Status

Early development. A working provider with a core set of resources, an internal
v2 API client, and offline (mock-server) acceptance tests. Live acceptance
testing against a pfSense VM is the next milestone (pending VM provisioning).

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
  descr      = "allow-https"
  type       = "pass"
  interface  = ["lan"]
  ipprotocol = "inet"
  protocol   = "tcp"
  source     = "any"
  destination_port = "443"
  destination     = "webservers"
}

resource "pfsense_system_user" "deploy" {
  name     = "deploy"
  password = var.deploy_password
  priv     = ["page-all"]
}
```

See [`examples/`](examples/) for more.

## Implemented resources

- `pfsense_firewall_alias`
- `pfsense_firewall_rule`
- `pfsense_firewall_schedule`
- `pfsense_interface_vlan`
- `pfsense_routing_gateway`
- `pfsense_routing_gateway_group`
- `pfsense_routing_static_route`
- `pfsense_services_dhcp_server`
- `pfsense_services_dns_resolver_host_override`
- `pfsense_services_cron_job`
- `pfsense_services_ntp_settings`
- `pfsense_system_user`
- `pfsense_system_group`
- `pfsense_system_ca`
- `pfsense_system_certificate`
- `pfsense_system_tunable`
- `pfsense_system_hostname`
- `pfsense_system_dns`
- `pfsense_system_timezone`

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

License is to be decided as part of the repository strategy (see the open
questions below). The upstream `pfSense-pkg-RESTAPI` is Apache-2.0.

## Open questions

1. **Repository**: rewrite `elacy/terraform-provider-pfsense` in place (major
   version bump) vs. a new repository.
2. **VM**: go-ahead to provision a pfSense CE 2.8.1 VM on the TrueNAS NAS for
   live acceptance testing (isolated network to avoid rogue DHCP).
