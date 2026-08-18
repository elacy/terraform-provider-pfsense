# v1 → v2 Upgrade & Migration Guide

This document describes how to migrate Terraform state and configuration from
`terraform-provider-pfsense` **v1.x** (terraform-plugin-sdk/v2) to **v2.x**
(terraform-plugin-framework).

The provider source address is unchanged — `registry.terraform.io/elacy/pfsense` —
so **no `state replace-provider` is required**. Migration is a mix of:

1. Manual edits to the `provider` block (authentication fields were renamed).
2. `terraform state mv` for four resources whose type names changed.
3. Automatic in-place state migration (framework `StateUpgrader`, schema version
   `0 → 1`) for the resources whose type names were kept.

> **Back up your state first.** Run `terraform state pull > backup.tfstate` and
> commit/keep that file before starting.

---

## 1. Provider block — manual edits required

The `provider` block's authentication attributes were renamed. Edit them by hand
before running `terraform plan`.

| v1 attribute          | v2 attribute        | Notes                                                        |
| --------------------- | ------------------- | ------------------------------------------------------------ |
| `url`                 | `url`               | unchanged (required)                                          |
| `user`                | `username`          | renamed                                                       |
| `password`            | `password`          | unchanged                                                      |
| `jwt_token`           | `use_jwt`           | changed from string token to `bool` (exchange user/password)  |
| `api_client_id`       | `api_key`           | merged into a single API key                                  |
| `api_client_token`    | `api_key`           | merged into a single API key                                  |
| `allow_insecure`      | `skip_tls_verify`   | renamed                                                       |
| `timeout`             | `timeout`           | unchanged                                                      |

Example migration:

```hcl
# v1
provider "pfsense" {
  url             = "https://10.99.0.2"
  user            = "admin"
  password        = "pfsense"
  allow_insecure  = true
}

# v2
provider "pfsense" {
  url             = "https://10.99.0.2"
  username        = "admin"
  password        = "pfsense"
  skip_tls_verify = true
}
```

---

## 2. Resource type renames — `terraform state mv` required

Four resources changed their Terraform type name. Run `terraform state mv` for
each before `terraform plan`, or Terraform will plan to destroy and recreate
them.

| v1 type                                   | v2 type                                         |
| ----------------------------------------- | ----------------------------------------------- |
| `pfsense_interface`                       | `pfsense_network_interface`                     |
| `pfsense_dhcp_server`                     | `pfsense_services_dhcp_server`                  |
| `pfsense_dhcp_static_mapping`             | `pfsense_services_dhcp_static_mapping`          |
| `pfsense_unbound_host_override`           | `pfsense_services_dns_resolver_host_override`   |

```bash
terraform state mv 'pfsense_interface.foo'                     'pfsense_network_interface.foo'
terraform state mv 'pfsense_dhcp_server.foo'                   'pfsense_services_dhcp_server.foo'
terraform state mv 'pfsense_dhcp_static_mapping.foo'           'pfsense_services_dhcp_static_mapping.foo'
terraform state mv 'pfsense_unbound_host_override.foo'         'pfsense_services_dns_resolver_host_override.foo'
```

> Repeat for every resource instance of each type. Use
> `terraform state list | grep <old_type>` to enumerate them.

After the `state mv`, the StateUpgrader (section 3) runs on the next
`plan`/`apply` and migrates the schema in place — no recreation, no config loss.

---

## 3. In-place state migration (automatic)

These resources keep their type name and are migrated automatically by a
framework `StateUpgrader` (`schema.Version 0 → 1`) on the first `apply`:

| Resource                                    | Field-level changes (old → new) |
| ------------------------------------------- | ------------------------------- |
| `pfsense_firewall_alias`                    | `description` → `descr`; `target` (list of `{address, description}`) flattened to `address` (list of string) + `detail` (list of string). `id` is the alias name. |
| `pfsense_firewall_rule`                     | `ack_queue`→`ackqueue`, `default_queue`→`defaultqueue`, `description`→`descr`, `dn_pipe`→`dnpipe`, `pdn_pipe`→`pdnpipe`, `ip_protocol`→`ipprotocol`, `schedule`→`sched`, `state_type`→`statetype`, `icmp_type`→`icmptype`; `tcp_flag` split into `tcp_flags_set` / `tcp_flags_out_of` / `tcp_flags_any`. `id` becomes the rule description. |
| `pfsense_network_interface` *(was `pfsense_interface`)* | field renames + list restructuring handled in-place |
| `pfsense_services_dhcp_server` *(was `pfsense_dhcp_server`)* | field renames handled in-place |
| `pfsense_services_dhcp_static_mapping` *(was `pfsense_dhcp_static_mapping`)* | field renames handled in-place |
| `pfsense_services_dns_resolver_host_override` *(was `pfsense_unbound_host_override`)* | `host_name`→`hostname`, `description`→`descr`, `domain_search_list`→`domainsearchlist` |

`pfsense_interface_vlan` is unchanged and requires no migration.

The full old→new field mappings are encoded in
`internal/provider/*_upgrade.go` and are exercised by the accompanying
`*_upgrade_test.go` unit tests.

---

## 4. Breaking change — floating firewall rules

The v2 `pfsense_firewall_rule` resource is a rewrite against the pfSense
`/api/v2/firewall/rule` endpoint and **no longer models floating rules**
(no `floating` attribute).

- `floating = false` (the v1 default) → migrated normally.
- `floating = true` → the StateUpgrader emits a **warning** and the floating
  attribute is dropped. You must re-create these rules outside of Terraform, or
  as a separate resource type if one is added later.

Check for affected rules before upgrading:

```bash
terraform state pull | grep -A3 '"floating": *true'
```

---

## 5. Recommended migration order

1. `terraform state pull > backup.tfstate` (back up).
2. Pin the provider to `v2.0.0` in `required_providers`.
3. Hand-edit the `provider` block (section 1).
4. Run all `terraform state mv` commands (section 2).
5. Grep for `floating = true` and triage (section 4).
6. `terraform plan` — expect **no** diffs (StateUpgraders migrate in place);
   any remaining "forces replacement" on a migrated resource is a bug to report.
7. `terraform apply`.

If a resource does force replacement after following this guide, file an issue
with the resource type and a redacted snippet of the `plan` diff — do not apply.
