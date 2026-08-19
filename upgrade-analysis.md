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

Every v1 resource is migrated automatically by a framework `StateUpgrader`
(`schema.Version 0 → 1`) on the first `plan`/`apply`. This is independent of
section 2: `pfsense_firewall_alias`, `pfsense_firewall_rule` and
`pfsense_interface_vlan` keep their type name and need no `state mv`, while the
four renamed resources run the same upgrader once the `state mv` has been done.

| Resource                                    | Field-level changes (old → new) |
| ------------------------------------------- | ------------------------------- |
| `pfsense_firewall_alias`                    | `description` → `descr`; `target` (list of `{address, description}`) flattened to `address` (list of string) + `detail` (list of string). `id` is the alias name. |
| `pfsense_firewall_rule`                     | `ack_queue`→`ackqueue`, `default_queue`→`defaultqueue`, `description`→`descr`, `dn_pipe`→`dnpipe`, `pdn_pipe`→`pdnpipe`, `ip_protocol`→`ipprotocol`, `schedule`→`sched`, `state_type`→`statetype`, `icmp_type`→`icmptype`; `tcp_flag` split into `tcp_flags_set` / `tcp_flags_out_of` / `tcp_flags_any`. `id` becomes the rule description. |
| `pfsense_interface_vlan`                    | `description` → `descr`; `if`, `tag` and `pcp` carry over unchanged; the new computed `vlanif` is populated from the old `id` (which was the generated VLAN interface name). `id` becomes `"<if>\|<tag>"`. |
| `pfsense_network_interface` *(was `pfsense_interface`)* | `description`→`descr`, `spoof_mac`→`spoofmac`, `block_private`→`blockpriv`, `block_bogons`→`blockbogons`, `type`→`typev4`, `type_v6`→`typev6`, `ip_address`→`ipaddr`, `ip_address_v6`→`ipaddrv6`, `gateway_v6`→`gatewayv6`, `dhcp_hostname`→`dhcphostname`, `dhcp_reject_from`→`dhcprejectfrom`, `subnet_v6`→`subnetv6`, `prefix_v6_rd`→`prefix_6rd`, `gateway_6_rd`→`gateway_6rd`, `prefix_6_rd_v4_plen`→`prefix_6rd_v4plen`, `track_v6_interface`→`track6_interface`, `track_v6_prefix_id_hex`→`track6_prefix_id_hex`, `ip_v6_use_v4_iface`→`ipv6usev4iface`, `adv_dhcp_config_file_override_file`→`adv_dhcp_config_file_override_path`. `mss` and `subnet_v6` are retyped from string to number. `dhcp_cv_pt` and `dhcp_vlan_enable` have no v2 equivalent and are dropped. `id` becomes the interface key (`if`) — the v1 id was the interface's descriptive name. |
| `pfsense_services_dhcp_server` *(was `pfsense_dhcp_server`)* | `default_lease_time`→`defaultleasetime`, `max_lease_time`→`maxleasetime` (retyped string→number), `dns_server`→`dnsserver`, `deny_unknown` (bool) →`denyunknown` (string: `true`→`"enabled"`, `false`/unset→ unset). `id` is the interface. **`domain_search_list`, `mac_allow_list`, `mac_deny_list` and `ignore_bootp` are dropped — see the note below.** |
| `pfsense_services_dhcp_static_mapping` *(was `pfsense_dhcp_static_mapping`)* | `interface`→`parent_id`, `client_identifier`→`cid`, `ip_address`→`ipaddr`, `host_name`→`hostname`, `description`→`descr`, `domain_search_list`→`domainsearchlist`, `dns_servers`→`dnsserver`. `id` becomes `"<interface>\|<mac>"` (was `"<interface>.<mac>"`). |
| `pfsense_services_dns_resolver_host_override` *(was `pfsense_unbound_host_override`)* | `dns` (an FQDN) split at the first dot into `host` + `domain`; `ip_addresses`→`ip`; `description`→`descr`; `aliases[].host_name`→`aliases[].host`, `aliases[].domain_name`→`aliases[].domain`, `aliases[].description`→`aliases[].descr`. `id` becomes `"<host>\|<domain>"` (was the FQDN). |

Optional string attributes that were never configured are normalised from the
SDKv2 empty-string zero value (`""`) back to null, so `terraform plan
-refresh=false` does not report a spurious `"" → null` diff on them.

### `pfsense_services_dhcp_server` — attributes not carried over

The v2 `pfsense_services_dhcp_server` resource does not model
`domain_search_list`, `mac_allow_list`, `mac_deny_list` or `ignore_bootp`.
When any of them is set, the StateUpgrader emits a **warning** and drops the
value from state. The pfSense configuration itself is untouched — only
Terraform's tracking of it is lost.

Re-create them as a `pfsense_services_dhcp_address_pool`, which exposes the
equivalent fields:

| v1 `pfsense_dhcp_server` attribute | `pfsense_services_dhcp_address_pool` attribute |
| --------------------------------- | --------------------------------------------- |
| `domain_search_list`              | `domainsearchlist`                             |
| `mac_allow_list`                  | `mac_allow`                                    |
| `mac_deny_list`                   | `mac_deny`                                     |
| `ignore_bootp`                    | `ignorebootp`                                  |

Check for affected servers before upgrading:

```bash
terraform state pull | grep -E '"(domain_search_list|mac_allow_list|mac_deny_list|ignore_bootp)"'
```

The full old→new field mappings are encoded in
`internal/provider/*_upgrade.go` and are exercised by the accompanying
`*_upgrade_test.go` unit tests.

---

## 4. Breaking changes — `pfsense_firewall_rule`

The v2 `pfsense_firewall_rule` resource is a rewrite against the pfSense
`/api/v2/firewall/rule` endpoint. Three v1 behaviours do not survive the
rewrite.

### 4.1 Floating rules are no longer modelled

There is no `floating` attribute in v2.

- `floating = false` (the v1 default) → migrated normally.
- `floating = true` → the StateUpgrader emits a **warning** and the floating
  attribute is dropped. You must re-create these rules outside of Terraform, or
  as a separate resource type if one is added later.

Check for affected rules before upgrading:

```bash
terraform state pull | grep -A3 '"floating": *true'
```

### 4.2 `description` is now the resource identity — and is mandatory

v2 identifies a firewall rule solely by its `descr`: `Read`, `Update` and
`Delete` look the rule up by that value. The v1 `description` was *optional*,
and the SDKv2 persisted an unset optional string as `""`.

A rule with an empty description therefore has no usable identity — an empty
`descr` would match the **first** unrelated rule that also has no description
and silently `PATCH` or `DELETE` it. The StateUpgrader **fails with an error**
in that case rather than warning, because a warning does not stop `apply`.

To fix an affected rule: give it a unique `description` (in the pfSense UI and
in your v1 configuration), refresh the v1 state so the description is recorded,
then re-run the upgrade.

Check for affected rules before upgrading:

```bash
terraform state pull | grep -E '"description": *""'
```

### 4.3 `ip_protocol = "inet46"` is no longer accepted

The v1 `ip_protocol` accepted `inet`, `inet6` and `inet46` (dual stack). The v2
`ipprotocol` attribute validates against `inet` / `inet6` only.

The StateUpgrader carries the value over as-is and emits a **warning** when it
is neither `inet` nor `inet6`, so you can see which rules need attention; the
next `plan` will fail validation on those rules until they are fixed. Set
`ipprotocol` to `inet` or `inet6`, splitting the rule into two if it genuinely
needs to cover both address families.

Check for affected rules before upgrading:

```bash
terraform state pull | grep '"ip_protocol": *"inet46"'
```

---

## 5. Recommended migration order

1. `terraform state pull > backup.tfstate` (back up).
2. Pin the provider to `v2.0.0` in `required_providers`.
3. Hand-edit the `provider` block (section 1).
4. Run all `terraform state mv` commands (section 2).
5. Triage the firewall rules (section 4): grep for `floating = true`, for
   rules with an empty `description` (these **block** the upgrade), and for
   `ip_protocol = "inet46"`.
6. Triage the DHCP servers (section 3): grep for `domain_search_list`,
   `mac_allow_list`, `mac_deny_list` and `ignore_bootp`, and plan the
   `pfsense_services_dhcp_address_pool` resources that will replace them.
7. `terraform plan` — expect **no** diffs (StateUpgraders migrate in place);
   any remaining "forces replacement" on a migrated resource is a bug to report.
8. `terraform apply`.

If a resource does force replacement after following this guide, file an issue
with the resource type and a redacted snippet of the `plan` diff — do not apply.
