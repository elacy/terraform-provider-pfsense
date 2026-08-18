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

// dnsHostOverrideAliasV0 is the version-0 state shape of one element of the
// old "aliases" list (TypeList of Resource with host_name/domain_name/
// description, from the SDKv2 pfsense_unbound_host_override resource).
type dnsHostOverrideAliasV0 struct {
	HostName    types.String `tfsdk:"host_name"`
	DomainName  types.String `tfsdk:"domain_name"`
	Description types.String `tfsdk:"description"`
}

// dnsResolverHostOverrideV0 is the schema-version-0 state shape of the old
// SDKv2 resource pfsense_unbound_host_override (git ref origin/v1). The tfsdk
// tags use the OLD attribute names so req.State.Get can decode prior state
// directly.
type dnsResolverHostOverrideV0 struct {
	DNS         types.String             `tfsdk:"dns"`
	IPAddresses types.List               `tfsdk:"ip_addresses"`
	Description types.String             `tfsdk:"description"`
	Aliases     []dnsHostOverrideAliasV0 `tfsdk:"aliases"`
}

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

// toCurrent maps version-0 state into the version-1 model
// (dnsResolverHostOverrideModel). Attribute renames:
//
//	dns           → host + domain (split at the first dot)
//	ip_addresses  → ip
//	description   → descr
//	aliases       → aliases (each element: host_name→host, domain_name→domain,
//	                          description→descr)
//
// The computed "id" (host|domain) is set by the StateUpgrader, not here.
func (v *dnsResolverHostOverrideV0) toCurrent(ctx context.Context) (dnsResolverHostOverrideModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	current := dnsResolverHostOverrideModel{
		IP:    v.IPAddresses,
		Descr: v.Description,
	}

	aliasType := types.ObjectType{AttrTypes: dnsAliasAttrTypes}
	if v.Aliases != nil {
		elements := make([]attr.Value, 0, len(v.Aliases))
		for _, alias := range v.Aliases {
			obj, d := types.ObjectValue(dnsAliasAttrTypes, map[string]attr.Value{
				"host":   alias.HostName,
				"domain": alias.DomainName,
				"descr":  alias.Description,
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
		// Null (unset) prior list stays null; the zero value of types.List is
		// not a valid state value.
		current.Aliases = types.ListNull(aliasType)
	}

	return current, diags
}

// UpgradeState returns the version 0 → 1 state upgrader for the renamed
// resource pfsense_unbound_host_override →
// pfsense_services_dns_resolver_host_override.
func (r *dnsResolverHostOverrideResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &dnsResolverHostOverridePriorSchemaV0,
			StateUpgrader: r.upgradeStateV0toV1,
		},
	}
}

var _ resource.ResourceWithUpgradeState = (*dnsResolverHostOverrideResource)(nil)

// upgradeStateV0toV1 migrates v1-provider state in place. The prior resource id
// is the FQDN ("<host>.<domain>"); the version-1 id is "<host>|<domain>".
func (r *dnsResolverHostOverrideResource) upgradeStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior dnsResolverHostOverrideV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	fqdn := priorResourceID(req.RawState)
	if fqdn == "" {
		// Fall back to the natural-key attribute ("dns" is Required in v0).
		fqdn = prior.DNS.ValueString()
	}
	if fqdn == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_services_dns_resolver_host_override",
			"unable to derive the resource id from the prior state: \"dns\" is empty",
		)
		return
	}
	host, domain := splitDNSHostDomain(fqdn)
	current.Host = types.StringValue(host)
	current.Domain = types.StringValue(domain)
	current.ID = types.StringValue(r.key(host, domain))

	resp.Diagnostics.Append(resp.State.Set(ctx, &current)...)
}

// splitDNSHostDomain splits a host-override FQDN into host and domain at the
// first dot, mirroring the SDKv2 splitDns helper.
func splitDNSHostDomain(fqdn string) (host, domain string) {
	if i := strings.Index(fqdn, "."); i >= 0 {
		return fqdn[:i], fqdn[i+1:]
	}
	return fqdn, ""
}
