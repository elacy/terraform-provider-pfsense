package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_dhcp_static_mapping (v0, SDKv2 provider v1) ->
// pfsense_services_dhcp_static_mapping (v1)
// ---------------------------------------------------------------------------

// dhcpStaticMappingModelV0 is the schema-version-0 state shape of the old
// SDKv2 resource pfsense_dhcp_static_mapping (git ref origin/v1). The tfsdk
// tags use the OLD attribute names so req.State.Get can decode prior state
// directly. The implicit SDKv2 `id` attribute is intentionally absent: it is
// read from req.RawState instead (see upgradeStateV0To1).
type dhcpStaticMappingModelV0 struct {
	Interface           types.String `tfsdk:"interface"`
	MAC                 types.String `tfsdk:"mac"`
	ClientIdentifier    types.String `tfsdk:"client_identifier"`
	IPAddress           types.String `tfsdk:"ip_address"`
	HostName            types.String `tfsdk:"host_name"`
	Description         types.String `tfsdk:"description"`
	Gateway             types.String `tfsdk:"gateway"`
	Domain              types.String `tfsdk:"domain"`
	DomainSearchList    types.List   `tfsdk:"domain_search_list"`
	DNSServers          types.List   `tfsdk:"dns_servers"`
	ARPTableStaticEntry types.Bool   `tfsdk:"arp_table_static_entry"`
}

var _ resource.ResourceWithUpgradeState = (*dhcpStaticMappingResource)(nil)

// dhcpStaticMappingPriorSchemaV0 is the PriorSchema for the version 0 → 1
// state upgrade. It contains exactly the old SDKv2 properties translated to
// framework attributes — no "id", which is implicit in both providers.
var dhcpStaticMappingPriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"interface":              schema.StringAttribute{Required: true},
		"mac":                    schema.StringAttribute{Required: true},
		"client_identifier":      schema.StringAttribute{Optional: true},
		"ip_address":             schema.StringAttribute{Optional: true},
		"host_name":              schema.StringAttribute{Optional: true},
		"description":            schema.StringAttribute{Optional: true},
		"gateway":                schema.StringAttribute{Optional: true},
		"domain":                 schema.StringAttribute{Optional: true},
		"domain_search_list":     schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"dns_servers":            schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"arp_table_static_entry": schema.BoolAttribute{Optional: true},
	},
}

// UpgradeState returns the version 0 → 1 state upgrader for the renamed
// resource pfsense_dhcp_static_mapping → pfsense_services_dhcp_static_mapping.
func (r *dhcpStaticMappingResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &dhcpStaticMappingPriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 migrates v1-provider state in place. The prior resource id
// is "<interface>.<mac>" (partition "<interface>", natural key "<mac>"); the
// version-1 id is "<interface>|<mac>".
func (r *dhcpStaticMappingResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior dhcpStaticMappingModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// `interface` and `mac` are both Required in v0, so the prior-state
	// attributes are always populated and are the authoritative source for
	// the natural key. (Parsing the v0 composite id "<interface>.<mac>" is
	// strictly weaker: a dotted interface name would split at the wrong dot.)
	iface := prior.Interface.ValueString()
	mac := prior.MAC.ValueString()
	if iface == "" || mac == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_services_dhcp_static_mapping",
			"unable to derive the resource id from the prior state: \"interface\" and/or \"mac\" is empty",
		)
		return
	}
	current.ID = types.StringValue(r.key(iface, mac))

	resp.Diagnostics.Append(resp.State.Set(ctx, &current)...)
}

// toCurrent maps version-0 state into the version-1 model
// (dhcpStaticMappingModel). Attribute renames:
//
//	interface          → parent_id
//	mac                → mac
//	client_identifier  → cid
//	ip_address         → ipaddr
//	host_name          → hostname
//	description        → descr
//	gateway            → gateway
//	domain             → domain
//	domain_search_list → domainsearchlist
//	dns_servers        → dnsserver
//	arp_table_static_entry → arp_table_static_entry
//
// defaultleasetime, maxleasetime, winsserver and ntpserver are new in version
// 1 and have no version-0 source, so they are left null. Optional attributes
// go through the zero-value normalisers — emptyToNull for strings,
// falseToNull for the `arp_table_static_entry` bool — so the SDKv2 "" / false
// zero values do not land in version-1 state where the framework means null.
// The computed "id" (interface|mac) is set by the StateUpgrader, not here.
func (m dhcpStaticMappingModelV0) toCurrent(ctx context.Context) (dhcpStaticMappingModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	return dhcpStaticMappingModel{
		ParentID:            m.Interface,
		MAC:                 m.MAC,
		Ipaddr:              emptyToNull(m.IPAddress),
		CID:                 emptyToNull(m.ClientIdentifier),
		Hostname:            emptyToNull(m.HostName),
		Domain:              emptyToNull(m.Domain),
		Domainsearchlist:    emptyListToNull(ctx, m.DomainSearchList),
		Gateway:             emptyToNull(m.Gateway),
		DNSServer:           emptyListToNull(ctx, m.DNSServers),
		WINSServer:          types.ListNull(types.StringType),
		NTPServer:           types.ListNull(types.StringType),
		ARPTableStaticEntry: falseToNull(m.ARPTableStaticEntry),
		Descr:               emptyToNull(m.Description),
	}, diags
}
