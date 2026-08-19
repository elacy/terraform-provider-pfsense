package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_interface (v0, SDKv2 provider v1) -> pfsense_network_interface (v1)
// ---------------------------------------------------------------------------

// networkInterfaceModelV0 is the schema-version-0 (SDKv2-era) state model for
// pfsense_interface. The tfsdk tags use the OLD attribute names so that
// req.State.Get decodes the prior state verbatim. The implicit SDKv2 `id`
// attribute is intentionally absent: the version-1 id is the interface key
// (`if`), which is Required in version 0 (see upgradeStateV0To1).
type networkInterfaceModelV0 struct {
	AdvDhcpConfigAdvanced         types.Bool   `tfsdk:"adv_dhcp_config_advanced"`
	AdvDhcpConfigFileOverride     types.Bool   `tfsdk:"adv_dhcp_config_file_override"`
	AdvDhcpConfigFileOverrideFile types.String `tfsdk:"adv_dhcp_config_file_override_file"`
	AdvDhcpOptionModifiers        types.String `tfsdk:"adv_dhcp_option_modifiers"`
	AdvDhcpPtBackoffCutoff        types.Int64  `tfsdk:"adv_dhcp_pt_backoff_cutoff"`
	AdvDhcpPtInitialInterval      types.Int64  `tfsdk:"adv_dhcp_pt_initial_interval"`
	AdvDhcpPtReboot               types.Int64  `tfsdk:"adv_dhcp_pt_reboot"`
	AdvDhcpPtRetry                types.Int64  `tfsdk:"adv_dhcp_pt_retry"`
	AdvDhcpPtSelectTimeout        types.Int64  `tfsdk:"adv_dhcp_pt_select_timeout"`
	AdvDhcpPtTimeout              types.Int64  `tfsdk:"adv_dhcp_pt_timeout"`
	AdvDhcpRequestOptions         types.String `tfsdk:"adv_dhcp_request_options"`
	AdvDhcpRequiredOptions        types.String `tfsdk:"adv_dhcp_required_options"`
	AdvDhcpSendOptions            types.String `tfsdk:"adv_dhcp_send_options"`
	AliasAddress                  types.String `tfsdk:"alias_address"`
	AliasSubnet                   types.Int64  `tfsdk:"alias_subnet"`
	BlockBogons                   types.Bool   `tfsdk:"block_bogons"`
	BlockPrivate                  types.Bool   `tfsdk:"block_private"`
	Description                   types.String `tfsdk:"description"`
	DhcpCvPt                      types.Int64  `tfsdk:"dhcp_cv_pt"`
	DhcpHostname                  types.String `tfsdk:"dhcp_hostname"`
	DhcpRejectFrom                types.List   `tfsdk:"dhcp_reject_from"`
	DhcpVlanEnable                types.Bool   `tfsdk:"dhcp_vlan_enable"`
	Enable                        types.Bool   `tfsdk:"enable"`
	Gateway                       types.String `tfsdk:"gateway"`
	Gateway6Rd                    types.String `tfsdk:"gateway_6_rd"`
	GatewayV6                     types.String `tfsdk:"gateway_v6"`
	If                            types.String `tfsdk:"if"`
	IpAddress                     types.String `tfsdk:"ip_address"`
	IpAddressV6                   types.String `tfsdk:"ip_address_v6"`
	IpV6UseV4Iface                types.Bool   `tfsdk:"ip_v6_use_v4_iface"`
	Media                         types.String `tfsdk:"media"`
	Mss                           types.String `tfsdk:"mss"`
	Mtu                           types.Int64  `tfsdk:"mtu"`
	PrefixV6Rd                    types.String `tfsdk:"prefix_v6_rd"`
	Prefix6RdV4Plen               types.Int64  `tfsdk:"prefix_6_rd_v4_plen"`
	SpoofMac                      types.String `tfsdk:"spoof_mac"`
	Subnet                        types.Int64  `tfsdk:"subnet"`
	SubnetV6                      types.String `tfsdk:"subnet_v6"`
	TrackV6Interface              types.String `tfsdk:"track_v6_interface"`
	TrackV6PrefixIdHex            types.String `tfsdk:"track_v6_prefix_id_hex"`
	Type                          types.String `tfsdk:"type"`
	TypeV6                        types.String `tfsdk:"type_v6"`
}

var _ resource.ResourceWithUpgradeState = (*networkInterfaceResource)(nil)

// networkInterfacePriorSchemaV0 is the SDKv2 pfsense_interface schema (from the
// provider v1 properties map) translated to framework attributes. It mirrors
// the v0 state exactly: same attribute names, same required/optional flags,
// SDKv2 types mapped per TYPE TRANSLATION rules. The implicit SDKv2 `id`
// attribute is deliberately excluded (the version-1 id is derived from `if`).
var networkInterfacePriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"adv_dhcp_config_advanced":           schema.BoolAttribute{Optional: true},
		"adv_dhcp_config_file_override":      schema.BoolAttribute{Optional: true},
		"adv_dhcp_config_file_override_file": schema.StringAttribute{Optional: true},
		"adv_dhcp_option_modifiers":          schema.StringAttribute{Optional: true},
		"adv_dhcp_pt_backoff_cutoff":         schema.Int64Attribute{Optional: true},
		"adv_dhcp_pt_initial_interval":       schema.Int64Attribute{Optional: true},
		"adv_dhcp_pt_reboot":                 schema.Int64Attribute{Optional: true},
		"adv_dhcp_pt_retry":                  schema.Int64Attribute{Optional: true},
		"adv_dhcp_pt_select_timeout":         schema.Int64Attribute{Optional: true},
		"adv_dhcp_pt_timeout":                schema.Int64Attribute{Optional: true},
		"adv_dhcp_request_options":           schema.StringAttribute{Optional: true},
		"adv_dhcp_required_options":          schema.StringAttribute{Optional: true},
		"adv_dhcp_send_options":              schema.StringAttribute{Optional: true},
		"alias_address":                      schema.StringAttribute{Optional: true},
		"alias_subnet":                       schema.Int64Attribute{Optional: true},
		"block_bogons":                       schema.BoolAttribute{Optional: true},
		"block_private":                      schema.BoolAttribute{Optional: true},
		"description":                        schema.StringAttribute{Required: true},
		"dhcp_cv_pt":                         schema.Int64Attribute{Optional: true},
		"dhcp_hostname":                      schema.StringAttribute{Optional: true},
		"dhcp_reject_from":                   schema.ListAttribute{ElementType: types.StringType, Optional: true},
		"dhcp_vlan_enable":                   schema.BoolAttribute{Optional: true},
		"enable":                             schema.BoolAttribute{Optional: true},
		"gateway":                            schema.StringAttribute{Optional: true},
		"gateway_6_rd":                       schema.StringAttribute{Optional: true},
		"gateway_v6":                         schema.StringAttribute{Optional: true},
		"if":                                 schema.StringAttribute{Required: true},
		"ip_address":                         schema.StringAttribute{Optional: true},
		"ip_address_v6":                      schema.StringAttribute{Optional: true},
		"ip_v6_use_v4_iface":                 schema.BoolAttribute{Optional: true},
		"media":                              schema.StringAttribute{Optional: true},
		"mss":                                schema.StringAttribute{Optional: true},
		"mtu":                                schema.Int64Attribute{Optional: true},
		"prefix_6_rd_v4_plen":                schema.Int64Attribute{Optional: true},
		"prefix_v6_rd":                       schema.StringAttribute{Optional: true},
		"spoof_mac":                          schema.StringAttribute{Optional: true},
		"subnet":                             schema.Int64Attribute{Optional: true},
		"subnet_v6":                          schema.StringAttribute{Optional: true},
		"track_v6_interface":                 schema.StringAttribute{Optional: true},
		"track_v6_prefix_id_hex":             schema.StringAttribute{Optional: true},
		"type":                               schema.StringAttribute{Optional: true},
		"type_v6":                            schema.StringAttribute{Optional: true},
	},
}

// UpgradeState implements resource.ResourceWithUpgradeState so that existing
// pfsense_interface state (schema version 0) is migrated in-place to
// pfsense_network_interface (schema version 1) with no recreation.
func (r *networkInterfaceResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &networkInterfacePriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 decodes the v0 state, maps every user-configurable value
// to its new home, derives the resource id, and writes the v1 state.
func (r *networkInterfaceResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior networkInterfaceModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cur, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The v1 contract is id == if: Read, Update and Delete all look the
	// interface up by the `if` key. The v0 id was the interface's descriptive
	// name, so carrying it over would put state out of contract until the
	// first Read self-heals it. `if` is Required in v0 and therefore always
	// populated, so use it directly.
	cur.ID = prior.If

	resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
}

// toCurrent maps every v0 value to its v1 home (renames, retypes, dropped
// fields, and v1-only fields left null). Optional attributes go through the
// zero-value normalisers — emptyToNull for strings, falseToNull for bools,
// zeroToNull for integers — so the SDKv2 "" / false / 0 zero values do not
// land in v1 state where the framework means null. Required v1 attributes are
// never normalised (see the `subnet` note below), and `type`/`type_v6` are
// remapped onto the v1 value domains.
func (m networkInterfaceModelV0) toCurrent(ctx context.Context) (networkInterfaceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var cur networkInterfaceModel

	cur.If = m.If
	cur.Enable = falseToNull(m.Enable)
	cur.Descr = m.Description
	cur.Spoofmac = emptyToNull(m.SpoofMac)
	cur.MTU = zeroToNull(m.Mtu)
	cur.Mss = upgradeStringToInt64(m.Mss, "mss", "pfsense_interface", &diags)
	cur.Media = emptyToNull(m.Media)
	// cur.Mediaopt: no v0 equivalent; left null.
	cur.Blockpriv = falseToNull(m.BlockPrivate)
	cur.Blockbogons = falseToNull(m.BlockBogons)
	cur.Typev4 = networkInterfaceTypev4ToCurrent(m.Type, &diags)
	cur.Ipaddr = emptyToNull(m.IpAddress)
	// `subnet` is normalised the same way as `ipaddr`: both were Optional in
	// v0 and Required in v2, so an unset v0 value (SDKv2 zero value) becomes
	// null. A null in a Required attribute fails the next plan loudly rather
	// than silently carrying a semantically wrong 0 (/0 subnet). See the
	// "optional in v0, required in v2" note in upgrade-analysis.md.
	cur.Subnet = zeroToNull(m.Subnet)
	cur.Gateway = emptyToNull(m.Gateway)
	cur.Dhcphostname = emptyToNull(m.DhcpHostname)
	cur.AliasAddress = emptyToNull(m.AliasAddress)
	cur.AliasSubnet = zeroToNull(m.AliasSubnet)
	cur.Dhcprejectfrom = m.DhcpRejectFrom
	cur.AdvDHCPConfigAdvanced = falseToNull(m.AdvDhcpConfigAdvanced)
	// cur.AdvDHCPPtValues: no v0 equivalent; left null.
	cur.AdvDHCPPtTimeout = zeroToNull(m.AdvDhcpPtTimeout)
	cur.AdvDHCPPtRetry = zeroToNull(m.AdvDhcpPtRetry)
	cur.AdvDHCPPtSelectTimeout = zeroToNull(m.AdvDhcpPtSelectTimeout)
	cur.AdvDHCPPtReboot = zeroToNull(m.AdvDhcpPtReboot)
	cur.AdvDHCPPtBackoffCutoff = zeroToNull(m.AdvDhcpPtBackoffCutoff)
	cur.AdvDHCPPtInitialInterval = zeroToNull(m.AdvDhcpPtInitialInterval)
	cur.AdvDHCPSendOptions = emptyToNull(m.AdvDhcpSendOptions)
	cur.AdvDHCPRequestOptions = emptyToNull(m.AdvDhcpRequestOptions)
	cur.AdvDHCPRequiredOptions = emptyToNull(m.AdvDhcpRequiredOptions)
	cur.AdvDHCPOptionModifiers = emptyToNull(m.AdvDhcpOptionModifiers)
	cur.AdvDHCPConfigFileOverride = falseToNull(m.AdvDhcpConfigFileOverride)
	cur.AdvDHCPConfigFileOverridePath = emptyToNull(m.AdvDhcpConfigFileOverrideFile)
	cur.Typev6 = networkInterfaceTypev6ToCurrent(m.TypeV6)
	cur.Ipaddrv6 = emptyToNull(m.IpAddressV6)
	cur.Subnetv6 = upgradeStringToInt64(m.SubnetV6, "subnet_v6", "pfsense_interface", &diags)
	cur.Gatewayv6 = emptyToNull(m.GatewayV6)
	cur.Ipv6usev4iface = falseToNull(m.IpV6UseV4Iface)
	// cur.Slaacusev4iface: no v0 equivalent; left null.
	cur.Prefix6rd = emptyToNull(m.PrefixV6Rd)
	cur.Gateway6rd = emptyToNull(m.Gateway6Rd)
	cur.Prefix6rdV4plen = zeroToNull(m.Prefix6RdV4Plen)
	cur.Track6Interface = emptyToNull(m.TrackV6Interface)
	cur.Track6PrefixIDHex = emptyToNull(m.TrackV6PrefixIdHex)

	return cur, diags
}

// networkInterfaceTypev4ToCurrent maps the v0 `type` value onto the v1
// `typev4` domain.
//
// v0 validated `type` with StringInSlice(["staticv4", "dhcp"]), so the only
// values that can appear in prior state are "staticv4", "dhcp" and "" (the
// SDKv2 zero value for an unset optional string). v1 makes `typev4` Required
// and validates it with OneOf("static", "dhcp", "none"), so "staticv4" has to
// be renamed and the unset case has to become the explicit "none":
//
//	"staticv4" -> "static"
//	"dhcp"     -> "dhcp"
//	""         -> "none"
//
// emptyToNull is deliberately NOT used here: `typev4` is Required in v1 and
// must never be null. Anything outside the v0 domain is reported as a warning
// (as with the v0-only ip_protocol = "inet46" in firewall_rule_upgrade.go)
// and mapped to "none", because carrying it over verbatim would only fail the
// OneOf validator on the next plan.
func networkInterfaceTypev4ToCurrent(v types.String, diags *diag.Diagnostics) types.String {
	switch typev4 := v.ValueString(); {
	case v.IsNull() || v.IsUnknown() || typev4 == "":
		return types.StringValue("none")
	case typev4 == "staticv4":
		return types.StringValue("static")
	case typev4 == "dhcp":
		return types.StringValue("dhcp")
	default:
		diags.AddWarning(
			"Unsupported interface type value replaced during state upgrade",
			"The v0 interface set `type = \""+typev4+"\"`, but the v2 `typev4` attribute only accepts "+
				"\"static\", \"dhcp\" or \"none\". The value was migrated as \"none\" so the upgraded state "+
				"stays valid. Set `typev4` explicitly in your configuration to the addressing mode this "+
				"interface actually uses, then re-apply.",
		)
		return types.StringValue("none")
	}
}

// networkInterfaceTypev6ToCurrent maps the v0 `type_v6` value onto the v1
// `typev6` domain.
//
// The v1 OneOf ("staticv6", "dhcp6", "slaac", "6rd", "track6", "6to4",
// "none") is a superset of the v0 StringInSlice, so every configured value
// carries over unchanged. Only the SDKv2 "" zero value needs mapping: `typev6`
// is Optional, but "none" (not null) is its explicit "no IPv6" value and is
// what pfSense reports back for such an interface, so "" becomes "none" to
// match what the first Read would write.
func networkInterfaceTypev6ToCurrent(v types.String) types.String {
	if v.IsUnknown() {
		return v
	}
	if v.IsNull() || v.ValueString() == "" {
		return types.StringValue("none")
	}
	return v
}
