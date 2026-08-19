package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_unbound_host_override (v0, SDKv2 provider v1) ->
// pfsense_services_dns_resolver_host_override (v1)
// ---------------------------------------------------------------------------

// dnsResolverHostOverrideAliasV0 is the version-0 state shape of one element
// of the old "aliases" list (TypeList of Resource with host_name/domain_name/
// description, from the SDKv2 pfsense_unbound_host_override resource).
type dnsResolverHostOverrideAliasV0 struct {
	HostName    types.String `tfsdk:"host_name"`
	DomainName  types.String `tfsdk:"domain_name"`
	Description types.String `tfsdk:"description"`
}

// dnsResolverHostOverrideModelV0 is the schema-version-0 state shape of the
// old SDKv2 resource pfsense_unbound_host_override (git ref origin/v1). The
// tfsdk tags use the OLD attribute names so req.State.Get can decode prior
// state directly. The implicit SDKv2 `id` attribute is intentionally absent:
// it is read from req.RawState instead (see upgradeStateV0To1).
type dnsResolverHostOverrideModelV0 struct {
	DNS         types.String                     `tfsdk:"dns"`
	IPAddresses types.List                       `tfsdk:"ip_addresses"`
	Description types.String                     `tfsdk:"description"`
	Aliases     []dnsResolverHostOverrideAliasV0 `tfsdk:"aliases"`
}

var _ resource.ResourceWithUpgradeState = (*dnsResolverHostOverrideResource)(nil)

// dnsResolverHostOverridePriorSchemaV0 is the PriorSchema for the version 0 →
// 1 state upgrade. It contains exactly the old SDKv2 properties translated to
// framework attributes — no "id", which is implicit in both providers.
var dnsResolverHostOverridePriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"dns": schema.StringAttribute{Required: true},
		"ip_addresses": schema.ListAttribute{
			ElementType: types.StringType,
			Required:    true,
		},
		"description": schema.StringAttribute{Optional: true},
		"aliases": schema.ListAttribute{
			Optional: true,
			ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
				"host_name":   types.StringType,
				"domain_name": types.StringType,
				"description": types.StringType,
			}},
		},
	},
}

// UpgradeState returns the version 0 → 1 state upgrader for the renamed
// resource pfsense_unbound_host_override →
// pfsense_services_dns_resolver_host_override.
func (r *dnsResolverHostOverrideResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &dnsResolverHostOverridePriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 migrates v1-provider state in place. The prior resource id
// is the FQDN ("<host>.<domain>"); the version-1 id is "<host>|<domain>".
func (r *dnsResolverHostOverrideResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior dnsResolverHostOverrideModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The v1 id is derived from the natural key "dns" (Required in v0), never
	// from the raw v0 id: SDKv2 sets the id once on Create and does not
	// rewrite it when the host is edited in place, so a stale raw id would
	// map the override to the wrong host after an in-place rename.
	fqdn := prior.DNS.ValueString()
	if fqdn == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_services_dns_resolver_host_override",
			"unable to derive the resource id from the prior state: \"dns\" is empty",
		)
		return
	}
	host, domain := splitDNSHostDomain(fqdn)
	if domain == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_services_dns_resolver_host_override",
			"the host override \""+fqdn+"\" has no domain component: a v1 FQDN must contain "+
				"at least one dot (host.domain). Add a domain to the override in pfSense, "+
				"refresh the v1 state, then re-run the upgrade",
		)
		return
	}
	current.Host = types.StringValue(host)
	current.Domain = types.StringValue(domain)
	current.ID = types.StringValue(r.key(host, domain))

	resp.Diagnostics.Append(resp.State.Set(ctx, &current)...)
}

// toCurrent maps version-0 state into the version-1 model
// (dnsResolverHostOverrideModel). Attribute renames:
//
//	dns           → host + domain (split at the first dot)
//	ip_addresses  → ip
//	description   → descr
//	aliases       → aliases (each element: host_name→host, domain_name→domain,
//	                          description→descr)
//
// Optional strings go through emptyToNull so the SDKv2 "" zero value does not
// land in version-1 state as an empty string where the framework means null.
// The computed "id" (host|domain) is set by the StateUpgrader, not here.
func (m dnsResolverHostOverrideModelV0) toCurrent(ctx context.Context) (dnsResolverHostOverrideModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current := dnsResolverHostOverrideModel{
		IP:    m.IPAddresses,
		Descr: emptyToNull(m.Description),
	}

	aliasType := types.ObjectType{AttrTypes: dnsAliasAttrTypes}
	// SDKv2 persisted an unset optional list as [] (its zero value), which is
	// indistinguishable from an explicitly empty list; both become null (the
	// list counterpart of emptyToNull) and self-heal on the first Read.
	if len(m.Aliases) > 0 {
		elements := make([]attr.Value, 0, len(m.Aliases))
		for _, alias := range m.Aliases {
			obj, d := types.ObjectValue(dnsAliasAttrTypes, map[string]attr.Value{
				"host":   alias.HostName,
				"domain": alias.DomainName,
				"descr":  emptyToNull(alias.Description),
			})
			diags.Append(d...)
			if diags.HasError() {
				return current, diags
			}
			elements = append(elements, obj)
		}
		list, d := types.ListValue(aliasType, elements)
		diags.Append(d...)
		if diags.HasError() {
			return current, diags
		}
		current.Aliases = list
	} else {
		// A null or empty prior list becomes null; an empty types.List is not
		// a valid state value and the first Read repopulates a non-empty one.
		current.Aliases = types.ListNull(aliasType)
	}

	return current, diags
}

// splitDNSHostDomain splits a host-override FQDN into host and domain at the
// first dot, mirroring the SDKv2 splitDns helper.
func splitDNSHostDomain(fqdn string) (host, domain string) {
	if i := strings.Index(fqdn, "."); i >= 0 {
		return fqdn[:i], fqdn[i+1:]
	}
	return fqdn, ""
}
