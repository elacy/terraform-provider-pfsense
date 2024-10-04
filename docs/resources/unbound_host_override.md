---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "pfsense_unbound_host_override Resource - terraform-provider-pfsense"
subcategory: ""
description: |-
  Unbound Host Override
---

# pfsense_unbound_host_override (Resource)

Unbound Host Override



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `dns` (String) Hostname of the host override.
- `ip_addresses` (List of String) IPv4 or IPv6 of the host override.

### Optional

- `aliases` (Block List) Host override aliases to associate with this host override. For more information on alias object fields, see documentation for /api/v1/services/dnsmasq/host_override/alias. (see [below for nested schema](#nestedblock--aliases))
- `description` (String) Description of the host override.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--aliases"></a>
### Nested Schema for `aliases`

Required:

- `domain_name` (String) Domnain Name of the host override alias.
- `host_name` (String) Hostname of the host override alias.

Optional:

- `description` (String) Description of the host override alias.