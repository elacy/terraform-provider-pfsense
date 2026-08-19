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
| `pfsense_network_interface` *(was `pfsense_interface`)* | `description`→`descr`, `spoof_mac`→`spoofmac`, `block_private`→`blockpriv`, `block_bogons`→`blockbogons`, `type`→`typev4` (values remapped, see section 4.4), `type_v6`→`typev6`, `ip_address`→`ipaddr`, `ip_address_v6`→`ipaddrv6`, `gateway_v6`→`gatewayv6`, `dhcp_hostname`→`dhcphostname`, `dhcp_reject_from`→`dhcprejectfrom`, `subnet_v6`→`subnetv6`, `prefix_v6_rd`→`prefix_6rd`, `gateway_6_rd`→`gateway_6rd`, `prefix_6_rd_v4_plen`→`prefix_6rd_v4plen`, `track_v6_interface`→`track6_interface`, `track_v6_prefix_id_hex`→`track6_prefix_id_hex`, `ip_v6_use_v4_iface`→`ipv6usev4iface`, `adv_dhcp_config_file_override_file`→`adv_dhcp_config_file_override_path`. `mss` and `subnet_v6` are retyped from string to number. `dhcp_cv_pt` and `dhcp_vlan_enable` have no v2 equivalent and are dropped. `id` becomes the interface key (`if`) — the v1 id was the interface's descriptive name. |
| `pfsense_services_dhcp_server` *(was `pfsense_dhcp_server`)* | `default_lease_time`→`defaultleasetime`, `max_lease_time`→`maxleasetime` (retyped string→number), `dns_server`→`dnsserver`, `deny_unknown` (bool) →`denyunknown` (string: `true`→`"enabled"`, `false`→null — the zero value is ambiguous and is normalised away). `id` is the interface. **`domain_search_list`, `mac_allow_list`, `mac_deny_list` and `ignore_bootp` are dropped — see the note below.** |
| `pfsense_services_dhcp_static_mapping` *(was `pfsense_dhcp_static_mapping`)* | `interface`→`parent_id`, `client_identifier`→`cid`, `ip_address`→`ipaddr`, `host_name`→`hostname`, `description`→`descr`, `domain_search_list`→`domainsearchlist`, `dns_servers`→`dnsserver`. `id` becomes `"<interface>\|<mac>"` (was `"<interface>.<mac>"`). |
| `pfsense_services_dns_resolver_host_override` *(was `pfsense_unbound_host_override`)* | `dns` (an FQDN) split at the first dot into `host` + `domain`; `ip_addresses`→`ip`; `description`→`descr`; `aliases[].host_name`→`aliases[].host`, `aliases[].domain_name`→`aliases[].domain`, `aliases[].description`→`aliases[].descr`. `id` becomes `"<host>\|<domain>"` (was the FQDN). |

### Zero-value normalisation (`""`, `false`, `0` → null)

Optional attributes that were never configured are normalised from their SDKv2
zero value back to null, so `terraform plan -refresh=false` does not report a
spurious diff on every attribute the practitioner never set:

| SDKv2 type       | Persisted zero value | Migrated to |
| ---------------- | -------------------- | ----------- |
| `TypeString`     | `""`                 | null        |
| `TypeBool`       | `false`              | null        |
| `TypeInt`        | `0`                  | null        |
| `TypeList`       | `[]`                 | null        |

The SDKv2 stored an unset optional attribute and an explicitly configured zero
value identically, so this mapping is **ambiguous and one-way**: an attribute
that was deliberately set to `false` or `0` is also migrated to null. That is
the safe direction — the attribute is optional, so null carries no meaning of
its own, and the first `Read` after the upgrade re-reads the real value from
the pfSense API, so the **state** self-heals on the first refresh. (The plan
does not: because these attributes are Optional and not Computed, a config
that omits one may still show a persistent diff against the API's value on
later refreshes — a pre-existing resource-level behaviour the upgrader cannot
change.)

Normalisation is applied to every attribute that was **optional in v0** — even
when it became required in v2 (see section 4.5): a zero value there is still
"unset" data, and mapping it to null fails the next plan loudly rather than
silently carrying a wrong value. Two attributes are deliberate exceptions,
each carrying a comment in `internal/provider/*_upgrade.go` explaining why:

- `pfsense_firewall_alias.detail` — when at least one target has a non-empty
  description, missing descriptions are padded with `""` to stay index-aligned
  with `address`. When every description is empty the whole list is still
  normalised to null (see `firewall_alias_upgrade.go`).
- `pfsense_network_interface.typev4` — an unset v0 `type` is synthesised as
  `"none"` rather than null because `typev4` is required and can never be
  null; `"none"` is an honest "unknown", not a retained zero value. The
  upgrader also emits a **warning** in this case so the unset mode is not
  silently lost (see section 4.4 and `network_interface_upgrade.go`).

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
# Only non-empty lists (or ignore_bootp=true) trigger the warning — SDKv2
# persists every DHCP server with these keys set to zero values, so matching
# the bare key name would flag 100% of servers and identify nothing.
terraform state pull | grep -E '"(domain_search_list|mac_allow_list|mac_deny_list)": *\[[^]]' 
terraform state pull | grep -E '"ignore_bootp": *true'
```

The full old→new field mappings are encoded in
`internal/provider/*_upgrade.go` and are exercised by the accompanying
`*_upgrade_test.go` unit tests.

---

## 4. Breaking changes

Sections 4.1–4.3 cover `pfsense_firewall_rule`, which is a rewrite against the
pfSense `/api/v2/firewall/rule` endpoint. Section 4.4 covers the
`pfsense_network_interface` addressing-type values, section 4.5 covers
attributes that were optional in v1 and are required in v2 across both
resources, and section 4.6 covers `pfsense_network_interface` attributes that
are no longer modelled. Section 4.7 covers a `pfsense_services_dhcp_server`
value that cannot be represented in v2, and sections 4.8–4.9 cover the two
remaining upgrade-blocking errors (a `pfsense_unbound_host_override` host
override with no domain, and a non-numeric `mss`/`subnet_v6`).

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
# Empty descriptions block the upgrade on firewall RULES only. The grep below
# also matches pfsense_firewall_alias, pfsense_dhcp_static_mapping and
# pfsense_unbound_host_override, whose optional description SDKv2 also persisted
# as "" — those are harmless. Cross-check each hit against the resource type
# (or filter: terraform state list | grep pfsense_firewall_rule).
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

### 4.4 `pfsense_network_interface` — `type` values are remapped to `typev4`

The v1 `type` attribute was optional and validated against
`["staticv4", "dhcp"]`. The v2 `typev4` attribute is **required** and validates
against `["static", "dhcp", "none"]`, so the value has to be translated rather
than copied. The StateUpgrader applies this mapping:

| v1 `type`      | v2 `typev4` | Notes                                              |
| -------------- | ----------- | -------------------------------------------------- |
| `staticv4`     | `static`    | renamed                                             |
| `dhcp`         | `dhcp`      | unchanged                                           |
| `""` (unset)   | `none`      | **warning** — `typev4` is required, so it can never be left null  |
| anything else  | `none`      | **warning** — should not occur; see below           |

Nothing outside that set can be produced by the v1 provider (its validator
rejected it), but if hand-edited state contains one the upgrader emits a
**warning** and writes `none`, so the upgraded state stays valid instead of
failing the v2 `OneOf` validator on the next plan. An unset (`""`) `type` —
the SDKv2 zero value for an omitted optional string — emits the same warning,
because it means the interface's real IPv4 mode was never recorded in Terraform.

**Update your configuration to match**: set `typev4` to the addressing mode the
interface *actually* uses — check the pfSense UI, or run `terraform state pull`
after a refresh. An interface that had `type = "staticv4"` must say
`typev4 = "static"`. An interface that never set `type` in the v1 config is the
dangerous case: its real mode is still configured in pfSense (often `static`),
but the upgrader only knows the v1 state was empty, so it writes `typev4 =
"none"` as the honest "unknown". **Do not leave `typev4 = "none"` on an
interface that actually has IPv4** — that drops its address on the next apply.
Set it to the real mode, and reserve `none` only for interfaces with no IPv4.

`type_v6` → `typev6` needs no value translation — the v2 domain
(`staticv6`, `dhcp6`, `slaac`, `6rd`, `track6`, `6to4`, `none`) is a superset
of the v1 one. An unset (`""`) v1 `type_v6` is normalised to null (typev6 is
Optional and not Computed), and the first Read re-populates what pfSense reports
back for an interface with no
IPv6 configuration. A value outside that set (hand-edited state) is carried over
verbatim with a **warning** naming it, since it will otherwise fail the v2
`OneOf` validator on the next plan.

Check which interfaces are affected before upgrading:

```bash
terraform state pull | grep -E '"type": *"(staticv4|dhcp)?"'
```

This grep also matches every other resource that carries a bare `type` key —
`pfsense_firewall_alias`, `pfsense_firewall_rule`,
`pfsense_services_dhcp_custom_option` and `pfsense_services_ntp_time_server` —
so cross-check the resource address (`terraform state list`) before acting.
Acting on a wrong hit here is the most dangerous of the triage steps: it can
leave `typev4 = "none"` on an interface that actually has IPv4, which drops its
address on the next apply.

### 4.5 Attributes that were optional in v1 and are required in v2

Some attributes that the v1 provider allowed you to omit are **required** in
v2. The StateUpgrader carries over whatever v1 held (normalising the SDKv2 zero
value to null for the string attributes, per section 3), but Terraform will
reject the *configuration* until you add the attribute by hand — the next plan
fails with `The argument "<name>" is required, but no definition was found`.

| Resource                    | Attribute (v1 → v2)          | v1        | v2       |
| --------------------------- | ---------------------------- | --------- | -------- |
| `pfsense_firewall_rule`     | `ip_protocol` → `ipprotocol` | optional  | required |
| `pfsense_firewall_rule`     | `source` → `source`          | optional  | required |
| `pfsense_firewall_rule`     | `destination` → `destination`| optional  | required |
| `pfsense_firewall_rule`     | `description` → `descr`      | optional  | required (and the resource identity — see 4.2, this one **blocks** the upgrade) |
| `pfsense_network_interface` | `type` → `typev4`            | optional  | required (value remapped — see 4.4) |
| `pfsense_network_interface` | `ip_address` → `ipaddr`      | optional  | required |
| `pfsense_network_interface` | `subnet` → `subnet`          | optional  | required |

Every attribute in this table **except `type` → `typev4`** (whose remap to
`"none"` is described in section 4.4) was optional in v0, so its SDKv2 zero
value (`""` for strings, `0` for `subnet`) is normalised to null in exactly the
same way as the optional-in-v2 attributes in section 3. An interface that never set
`ipaddr` or `subnet` therefore arrives in v2 state with those attributes null,
which makes the next `terraform plan` fail with a clear "missing required
argument" — add the real values to your configuration (the commands below find
them).

Find the resources you have to hand-edit before upgrading:

```bash
# firewall rules missing ip_protocol / source / destination
terraform state pull | grep -E '"(ip_protocol|source|destination)": *""'

# interfaces missing type / ip_address / subnet (the "type"/"ip_address" grep
# also catches dhcp/unbound host entries on some API versions — cross-check the
# surrounding resource address, e.g. via terraform state list)
terraform state pull | grep -E '"(type|ip_address)": *""'
terraform state pull | grep -E '"subnet": *0'
```

Add the missing arguments to your configuration (using the values pfSense
actually has, visible in the UI or via `terraform state pull`) before running
`terraform plan`.

### 4.6 `pfsense_network_interface` — `dhcp_cv_pt` and `dhcp_vlan_enable` are dropped

The v2 `pfsense_network_interface` resource does not model `dhcp_cv_pt` or
`dhcp_vlan_enable`. They are functional DHCP settings, so when either is set
the StateUpgrader emits a **warning** and drops the value from state. The
pfSense configuration itself is unchanged — only Terraform's tracking of it is
lost — and the setting has to be re-applied by hand (in the pfSense UI or a
future provider resource).

Check for affected interfaces before upgrading:

```bash
terraform state pull | grep -E '"(dhcp_cv_pt|dhcp_vlan_enable)": *(true|[1-9])'
```

### 4.7 `pfsense_services_dhcp_server` — non-numeric `max_lease_time` is dropped

The v1 `max_lease_time` was an unvalidated optional *string* that accepted both
a number of seconds and the literal `"infinite"` (no lease-expiry cap). The v2
`maxleasetime` is an integer with no representation for `"infinite"`, so the
StateUpgrader maps any non-numeric value to null and emits a **warning** naming
the value.

The pfSense configuration itself is untouched — only Terraform's tracking is
lost — and null simply means "leave it to the server default". If the server
requires a finite maximum lease time, set `maxleasetime` explicitly in your
configuration.

Check for affected servers before upgrading:

```bash
terraform state pull | grep -E '"max_lease_time": *"[^"]*[^0-9"][^"]*"'
```

### 4.8 `pfsense_unbound_host_override` — a `dns` value with no dot blocks the upgrade

The v1 `dns` was a host override written as `host.domain`. The StateUpgrader
splits it at the first `.` into the v2 `host` and `domain` attributes, which are
what the v2 `id` (`host|domain`) is derived from. A `dns` value with no `.` has
no domain component, so the split fails and the upgrader raises an **error**.
This is the blocking counterpart to section 4.2: unlike `max_lease_time`
(section 4.7), there is no null fallback here because the natural key itself
is the unparseable value.

Check for affected overrides before upgrading:

```bash
terraform state pull | grep -E '"dns": *"[^."]+"'
```

If a value has no domain, correct it in the pfSense UI (host overrides should
be FQDNs like `alarm.lan`) and refresh the v1 state so a proper `host.domain`
is recorded, then re-run the upgrade.

### 4.9 `pfsense_network_interface` — a non-numeric `mss` / `subnet_v6` blocks the upgrade

`mss` and `subnet_v6` were unvalidated optional strings in v1 and are retyped
to integers in v2. Unlike `max_lease_time` (section 4.7) — where `"infinite"`
is a legitimate value — there is no valid non-numeric `mss` or `subnet_v6`, so
the upgrader raises an **error** rather than dropping it silently. (The
`"infinite"` lenient path in section 4.7 is specific to `max_lease_time`; do
not assume the same behaviour here.)

Check for affected interfaces before upgrading:

```bash
terraform state pull | grep -E '"(mss|subnet_v6)": *"[^"]*[^0-9"][^"]*"'
```

If one is non-numeric, canonicalise the burst size / prefix length in the
pfSense UI (or hand-edit the backed-up state file) so the attribute holds a
plain integer, then re-run the upgrade.

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
   `pfsense_services_dhcp_address_pool` resources that will replace them. Also
   grep for a non-numeric `max_lease_time` (section 4.7) and set a finite
   `maxleasetime` where the server needs a cap.
7. Rewrite `type` as `typev4` on every `pfsense_network_interface`
   (`staticv4` → `static`). For interfaces that never set `type`, set `typev4`
   to the mode the interface actually uses (check the pfSense UI or
   `terraform state pull`) — do **not** blindly use `none` unless the interface
   genuinely has no IPv4 — section 4.4.
8. Add the arguments that became required in v2 (section 4.5): `ipprotocol`,
   `source` and `destination` on firewall rules; `typev4`, `ipaddr` and
   `subnet` on network interfaces.
9. Check for the two remaining upgrade-blocking conditions (sections 4.8–4.9):
   a `pfsense_unbound_host_override` whose `dns` has no dot, and a
   non-numeric `mss` / `subnet_v6` on a `pfsense_network_interface`.
10. `terraform plan` — expect no *replacements* and no unexplained changes
   (StateUpgraders migrate in place). Attributes that were normalised to null
   (section 3) may still show a small in-place diff against the API's value on
   later refreshes; that is expected and harmless. Any remaining "forces
   replacement" on a migrated resource, or any in-place `typev4` change, is a
   red flag to investigate before applying.
11. `terraform apply`.

If a resource does force replacement after following this guide, file an issue
with the resource type and a redacted snippet of the `plan` diff — do not apply.
