package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dhcpStaticMappingV0 is the schema-version-0 state shape of the old SDKv2
// resource pfsense_dhcp_static_mapping (git ref origin/v1). The tfsdk tags use
// the OLD attribute names so req.State.Get can decode prior state directly.
type dhcpStaticMappingV0 struct {
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
// 1 and have no version-0 source, so they are left null. The computed "id"
// (interface|mac) is set by the StateUpgrader, not here.
func (v *dhcpStaticMappingV0) toCurrent(ctx context.Context) (dhcpStaticMappingModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	return dhcpStaticMappingModel{
		ParentID:            v.Interface,
		MAC:                 v.MAC,
		Ipaddr:              v.IPAddress,
		CID:                 v.ClientIdentifier,
		Hostname:            v.HostName,
		Domain:              v.Domain,
		Domainsearchlist:    v.DomainSearchList,
		Gateway:             v.Gateway,
		DNSServer:           v.DNSServers,
		WINSServer:          types.ListNull(types.StringType),
		NTPServer:           types.ListNull(types.StringType),
		ARPTableStaticEntry: v.ARPTableStaticEntry,
		Descr:               v.Description,
	}, diags
}

// UpgradeState returns the version 0 → 1 state upgrader for the renamed
// resource pfsense_dhcp_static_mapping → pfsense_services_dhcp_static_mapping.
func (r *dhcpStaticMappingResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &dhcpStaticMappingPriorSchemaV0,
			StateUpgrader: r.upgradeStateV0toV1,
		},
	}
}

var _ resource.ResourceWithUpgradeState = (*dhcpStaticMappingResource)(nil)

// upgradeStateV0toV1 migrates v1-provider state in place. The prior resource id
// is "<interface>.<mac>" (partition "<interface>", natural key "<mac>"); the
// version-1 id is "<interface>|<mac>".
func (r *dhcpStaticMappingResource) upgradeStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior dhcpStaticMappingV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	iface, mac := splitPriorStaticMappingID(priorResourceID(req.RawState))
	if iface == "" || mac == "" {
		// Fall back to the natural-key attributes (both Required in v0).
		iface = prior.Interface.ValueString()
		mac = prior.MAC.ValueString()
	}
	if iface == "" || mac == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_services_dhcp_static_mapping",
			"unable to derive the resource id from the prior state: both \"interface\" and \"mac\" are empty",
		)
		return
	}
	current.ID = types.StringValue(r.key(iface, mac))

	resp.Diagnostics.Append(resp.State.Set(ctx, &current)...)
}

// splitPriorStaticMappingID splits the v0 resource id ("<interface>.<mac>")
// into its natural-key parts.
func splitPriorStaticMappingID(id string) (iface, mac string) {
	parts := strings.SplitN(id, ".", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
