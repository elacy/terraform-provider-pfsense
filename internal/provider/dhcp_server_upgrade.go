package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_dhcp_server (v0, SDKv2 provider v1) -> pfsense_services_dhcp_server
// (v1)
// ---------------------------------------------------------------------------

// dhcpServerModelV0 is the schema-version-0 (SDKv2-era) state model for
// pfsense_dhcp_server. The tfsdk tags use the OLD attribute names so that
// req.State.Get decodes the prior state verbatim. The implicit SDKv2 `id`
// attribute is intentionally absent: the v1 id is derived from the natural key
// `interface` instead (see upgradeStateV0To1).
type dhcpServerModelV0 struct {
	DefaultLeaseTime types.Int64  `tfsdk:"default_lease_time"`
	DenyUnknown      types.Bool   `tfsdk:"deny_unknown"`
	DNSServer        types.List   `tfsdk:"dns_server"`
	Domain           types.String `tfsdk:"domain"`
	DomainSearchList types.List   `tfsdk:"domain_search_list"`
	Enable           types.Bool   `tfsdk:"enable"`
	Gateway          types.String `tfsdk:"gateway"`
	IgnoreBootp      types.Bool   `tfsdk:"ignore_bootp"`
	Interface        types.String `tfsdk:"interface"`
	MacAllowList     types.List   `tfsdk:"mac_allow_list"`
	MacDenyList      types.List   `tfsdk:"mac_deny_list"`
	MaxLeaseTime     types.String `tfsdk:"max_lease_time"`
	RangeFrom        types.String `tfsdk:"range_from"`
	RangeTo          types.String `tfsdk:"range_to"`
}

var _ resource.ResourceWithUpgradeState = (*dhcpServerResource)(nil)

// dhcpServerPriorSchemaV0 is the SDKv2 pfsense_dhcp_server schema (from the
// provider v1 properties map) translated to framework attributes. It mirrors
// the v0 state exactly: same attribute names, same required/optional flags,
// SDKv2 types mapped per TYPE TRANSLATION rules. The type asymmetry between
// the two lease-time attributes is real v1 behaviour, not a transcription
// bug: `default_lease_time` was already an integer in v1, while
// `max_lease_time` was a string in v1 (it can be "infinite") and is retyped to
// the v2 integer in toCurrent via dhcpServerMaxLeaseTimeUpgrade, which maps a
// non-numeric value such as "infinite" to null with a warning. The implicit
// SDKv2 `id` attribute is deliberately excluded (the v1 id is derived from the
// natural key `interface`).
var dhcpServerPriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"default_lease_time": schema.Int64Attribute{Optional: true},
		"deny_unknown":       schema.BoolAttribute{Optional: true},
		"dns_server":         schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"domain":             schema.StringAttribute{Optional: true},
		"domain_search_list": schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"enable":             schema.BoolAttribute{Optional: true},
		"gateway":            schema.StringAttribute{Optional: true},
		"ignore_bootp":       schema.BoolAttribute{Optional: true},
		"interface":          schema.StringAttribute{Required: true},
		"mac_allow_list":     schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"mac_deny_list":      schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"max_lease_time":     schema.StringAttribute{Optional: true},
		"range_from":         schema.StringAttribute{Optional: true},
		"range_to":           schema.StringAttribute{Optional: true},
	},
}

// UpgradeState implements resource.ResourceWithUpgradeState so that existing
// pfsense_dhcp_server state (schema version 0) is migrated in-place to
// pfsense_services_dhcp_server (schema version 1) with no recreation.
func (r *dhcpServerResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &dhcpServerPriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 decodes the v0 state, maps every user-configurable value
// to its new home, restores the resource id, and writes the v1 state.
func (r *dhcpServerResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior dhcpServerModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cur, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// `interface` is Required in v0, so it is always populated, but guard it
	// anyway (defense-in-depth): an empty id would make Read/Update/Delete
	// target the first unrelated DHCP server via findByKey(..., "interface", "").
	if prior.Interface.ValueString() == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_dhcp_server",
			"unable to derive the resource id from the prior state: \"interface\" is empty",
		)
		return
	}
	// The v1 id is the interface (`id == interface`, set by dhcpServerResource.Read);
	// it is derived from the natural key, not the raw v0 id.
	cur.ID = prior.Interface

	resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
}

// dhcpServerMaxLeaseTimeUpgrade converts the v0 `max_lease_time` string to the
// v1 `maxleasetime` integer. v0 held it as an unvalidated optional string, and
// the pfSense API accepts not just a number of seconds but the literal
// "infinite" (no lease-expiry cap). The v1 `maxleasetime` is an Int64 with no
// representation for "infinite", so a non-numeric value maps to null with a
// WARNING naming the value rather than an ERROR — an error would abort
// UpgradeResourceState and brick the resource permanently, whereas null simply
// means "leave it to the server default" and the first Read repopulates the
// real value. A numeric string parses normally.
func dhcpServerMaxLeaseTimeUpgrade(v types.String, diags *diag.Diagnostics) types.Int64 {
	if v.IsNull() || v.ValueString() == "" {
		return types.Int64Null()
	}
	n, err := strconv.ParseInt(v.ValueString(), 10, 64)
	if err != nil {
		diags.AddWarning(
			"max_lease_time Dropped During State Upgrade",
			"An error was encountered when upgrading pfsense_dhcp_server state "+
				"from schema version 0 to version 1: the value of attribute "+
				"\"max_lease_time\" (\""+v.ValueString()+"\") is not a number of seconds "+
				"(the API also accepts \"infinite\"), which the v1 integer \"maxleasetime\" "+
				"cannot represent. It has been dropped (migrated to null); configure a "+
				"finite \"maxleasetime\" if the server requires one.",
		)
		return types.Int64Null()
	}
	return types.Int64Value(n)
}

// toCurrent maps every v0 value to its v1 home (renames, retypes, dropped
// fields, and v1-only fields left null). Optional attributes go through the
// zero-value normalisers — emptyToNull for strings, falseToNull for bools,
// zeroToNull for integers — so the SDKv2 "" / false / 0 zero values do not
// land in v1 state where the framework means null.
func (m dhcpServerModelV0) toCurrent(ctx context.Context) (dhcpServerModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var cur dhcpServerModel

	cur.Interface = m.Interface
	cur.Enable = falseToNull(m.Enable)
	cur.RangeFrom = emptyToNull(m.RangeFrom)
	cur.RangeTo = emptyToNull(m.RangeTo)
	cur.Domain = emptyToNull(m.Domain)
	cur.DefaultLease = zeroToNull(m.DefaultLeaseTime)
	cur.MaxLease = dhcpServerMaxLeaseTimeUpgrade(m.MaxLeaseTime, &diags)
	cur.Gateway = emptyToNull(m.Gateway)
	cur.DNSServer = emptyListToNull(ctx, m.DNSServer)
	// WINSServer, NTPServer and StaticMap have no v0 equivalent; keep them
	// null but with their real element types so the state can be written.
	cur.WINSServer = types.ListNull(types.StringType)
	cur.NTPServer = types.ListNull(types.StringType)
	cur.DenyUnknown = dhcpServerDenyUnknownToCurrent(m.DenyUnknown)
	cur.StaticMap = types.ListNull(types.ObjectType{AttrTypes: dhcpStaticMapAttrTypes})

	diags.Append(dhcpServerDroppedAttributeWarnings(m)...)

	return cur, diags
}

// dhcpServerDenyUnknownToCurrent converts the v0 `deny_unknown` bool to the v1
// `denyunknown` string.
//
// On pfsense_services_dhcp_server `denyunknown` is an unvalidated optional
// string; the sibling pfsense_services_dhcp_address_pool exposes the same API
// field with domain "enabled" / "class". Only true has an unambiguous
// spelling ("enabled"); false is the SDKv2 zero value for an unset optional
// bool and cannot be told apart from an explicit off, so it is normalised to
// null (the attribute is Optional) and the first Read re-populates the real
// value from the API. Mapping false to "disabled" would surface a spurious
// "disabled" -> null diff on every migrated server that never configured it.
func dhcpServerDenyUnknownToCurrent(v types.Bool) types.String {
	if v.IsNull() || v.IsUnknown() {
		return types.StringNull()
	}
	if v.ValueBool() {
		return types.StringValue("enabled")
	}
	return types.StringNull()
}

// dhcpServerDroppedAttributeWarnings reports the v0 attributes that the v1
// pfsense_services_dhcp_server resource does not model. They are access
// control / DHCP option data, so dropping them silently would quietly change
// what the DHCP server hands out and to whom; the practitioner has to
// re-create them as a pfsense_services_dhcp_address_pool, which exposes
// mac_allow, mac_deny, ignorebootp and domainsearchlist.
func dhcpServerDroppedAttributeWarnings(m dhcpServerModelV0) diag.Diagnostics {
	var diags diag.Diagnostics

	dropped := make([]string, 0, 4)
	for _, l := range []struct {
		name string
		list types.List
	}{
		{"domain_search_list", m.DomainSearchList},
		{"mac_allow_list", m.MacAllowList},
		{"mac_deny_list", m.MacDenyList},
	} {
		if !l.list.IsNull() && !l.list.IsUnknown() && len(l.list.Elements()) > 0 {
			dropped = append(dropped, l.name)
		}
	}
	if m.IgnoreBootp.ValueBool() {
		dropped = append(dropped, "ignore_bootp")
	}

	if len(dropped) == 0 {
		return diags
	}

	list := ""
	for i, name := range dropped {
		if i > 0 {
			list += ", "
		}
		list += "`" + name + "`"
	}

	diags.AddWarning(
		"DHCP server attributes not carried over",
		"The v2 pfsense_services_dhcp_server resource does not model "+list+", so "+
			"the value(s) were dropped from the upgraded state. These control which clients get "+
			"leases and what options they receive — the pfSense configuration itself is unchanged, "+
			"but Terraform no longer manages them. Re-create them as a "+
			"`pfsense_services_dhcp_address_pool` resource, which exposes `mac_allow`, `mac_deny`, "+
			"`ignorebootp` and `domainsearchlist`.",
	)

	return diags
}
