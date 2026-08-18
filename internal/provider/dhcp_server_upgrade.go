package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
// attribute is intentionally absent: it is carried over from req.RawState
// instead (see dhcpServerUpgradeStateV0To1).
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

// dhcpServerPriorSchema is the SDKv2 pfsense_dhcp_server schema (from the
// provider v1 properties map) translated to framework attributes. It mirrors
// the v0 state exactly: same attribute names, same required/optional flags,
// SDKv2 types mapped per TYPE TRANSLATION rules (including max_lease_time as
// a String, since that is how it was stored in v0). The implicit SDKv2 `id`
// attribute is deliberately excluded (it is carried over from RawState).
func dhcpServerPriorSchema() schema.Schema {
	return schema.Schema{
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
}

// UpgradeState implements resource.ResourceWithUpgradeState so that existing
// pfsense_dhcp_server state (schema version 0) is migrated in-place to
// pfsense_services_dhcp_server (schema version 1) with no recreation.
func (r *dhcpServerResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	priorSchema := dhcpServerPriorSchema()
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &priorSchema,
			StateUpgrader: dhcpServerUpgradeStateV0To1,
		},
	}
}

// dhcpServerUpgradeStateV0To1 decodes the v0 state, maps every
// user-configurable value to its new home, restores the resource id, and
// writes the v1 state.
func dhcpServerUpgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
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

	cur.ID = dhcpServerPriorID(req, prior)

	resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
}

// dhcpServerPriorID returns the v0 resource id read from the raw prior state,
// falling back to the natural key (`interface`) when it is absent. In the v0
// provider the resource id IS the interface value, so this is a pure
// round-trip of the old id.
func dhcpServerPriorID(req resource.UpgradeStateRequest, prior dhcpServerModelV0) types.String {
	if id := priorResourceID(req.RawState); id != "" {
		return types.StringValue(id)
	}
	return prior.Interface
}

// toCurrent maps every v0 value to its v1 home (renames, retypes, dropped
// fields, and v1-only fields left null).
func (m dhcpServerModelV0) toCurrent(ctx context.Context) (dhcpServerModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var cur dhcpServerModel

	cur.Interface = m.Interface
	cur.Enable = m.Enable
	cur.RangeFrom = m.RangeFrom
	cur.RangeTo = m.RangeTo
	cur.Domain = m.Domain
	cur.DefaultLease = m.DefaultLeaseTime
	cur.MaxLease = dhcpServerStringToInt64(m.MaxLeaseTime, "max_lease_time", &diags)
	cur.Gateway = m.Gateway
	cur.DNSServer = m.DNSServer
	// WINSServer, NTPServer and StaticMap have no v0 equivalent; keep them
	// null but with their real element types so the state can be written.
	cur.WINSServer = types.ListNull(types.StringType)
	cur.NTPServer = types.ListNull(types.StringType)
	cur.DenyUnknown = dhcpServerDenyUnknownToCurrent(m.DenyUnknown)
	cur.StaticMap = types.ListNull(types.ObjectType{AttrTypes: dhcpStaticMapAttrTypes})

	return cur, diags
}

// dhcpServerDenyUnknownToCurrent converts the v0 bool to the v1 string value
// the pfSense API stores for the denyunknown field.
func dhcpServerDenyUnknownToCurrent(v types.Bool) types.String {
	if v.IsNull() {
		return types.StringNull()
	}
	if v.ValueBool() {
		return types.StringValue("enabled")
	}
	return types.StringValue("disabled")
}

// dhcpServerStringToInt64 converts the v0 max_lease_time string attribute
// (which held a number) into the v1 integer type. An empty string (the SDKv2
// zero value for an unset optional string) maps to null; a non-numeric value
// yields a diagnostic rather than silently corrupting state.
func dhcpServerStringToInt64(v types.String, attr string, diags *diag.Diagnostics) types.Int64 {
	if v.IsNull() || v.ValueString() == "" {
		return types.Int64Null()
	}
	n, err := strconv.ParseInt(v.ValueString(), 10, 64)
	if err != nil {
		diags.AddAttributeError(
			path.Root(attr),
			"Invalid Value During State Upgrade",
			"An error was encountered when upgrading pfsense_dhcp_server state "+
				"from schema version 0 to version 1: the value of attribute "+
				"\""+attr+"\" (\""+v.ValueString()+"\") could not be parsed as an integer. "+
				"Set a valid integer value or remove the attribute from the resource configuration, then re-apply.",
		)
		return types.Int64Null()
	}
	return types.Int64Value(n)
}
