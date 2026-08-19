package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNetworkInterfaceModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	rejectFrom := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("10.0.0.5"),
		types.StringValue("10.0.0.6"),
	})

	prior := networkInterfaceModelV0{
		AdvDhcpConfigAdvanced:         types.BoolValue(true),
		AdvDhcpConfigFileOverride:     types.BoolValue(true),
		AdvDhcpConfigFileOverrideFile: types.StringValue("/cf/conf/dhcpd.conf"),
		AdvDhcpOptionModifiers:        types.StringValue("option domain-name-servers 1.1.1.1;"),
		AdvDhcpPtBackoffCutoff:        types.Int64Value(15),
		AdvDhcpPtInitialInterval:      types.Int64Value(2),
		AdvDhcpPtReboot:               types.Int64Value(10),
		AdvDhcpPtRetry:                types.Int64Value(5),
		AdvDhcpPtSelectTimeout:        types.Int64Value(0),
		AdvDhcpPtTimeout:              types.Int64Value(60),
		AdvDhcpRequestOptions:         types.StringValue("domain-name-servers"),
		AdvDhcpRequiredOptions:        types.StringValue("domain-name"),
		AdvDhcpSendOptions:            types.StringValue("host-name"),
		AliasAddress:                  types.StringValue("10.0.0.10"),
		AliasSubnet:                   types.Int64Value(24),
		BlockBogons:                   types.BoolValue(true),
		BlockPrivate:                  types.BoolValue(false),
		Description:                   types.StringValue("WAN"),
		DhcpCvPt:                      types.Int64Value(3),
		DhcpHostname:                  types.StringValue("wan-dhcp"),
		DhcpRejectFrom:                rejectFrom,
		DhcpVlanEnable:                types.BoolValue(true),
		Enable:                        types.BoolValue(true),
		Gateway:                       types.StringValue("GW_WAN"),
		Gateway6Rd:                    types.StringValue("2001:db8::1"),
		GatewayV6:                     types.StringValue("GW6_WAN"),
		If:                            types.StringValue("wan"),
		IpAddress:                     types.StringValue("10.0.0.1"),
		IpAddressV6:                   types.StringValue("2001:db8::1"),
		IpV6UseV4Iface:                types.BoolValue(true),
		Media:                         types.StringValue("1000baseT"),
		Mss:                           types.StringValue("1500"),
		Mtu:                           types.Int64Value(1500),
		PrefixV6Rd:                    types.StringValue("2001:db8:abcd::"),
		Prefix6RdV4Plen:               types.Int64Value(32),
		SpoofMac:                      types.StringValue("00:11:22:33:44:55"),
		Subnet:                        types.Int64Value(24),
		SubnetV6:                      types.StringValue("64"),
		TrackV6Interface:              types.StringValue("wan"),
		TrackV6PrefixIdHex:            types.StringValue("0"),
		Type:                          types.StringValue("staticv4"),
		TypeV6:                        types.StringValue("staticv6"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	// Identity pass-through.
	if cur.If.ValueString() != "wan" {
		t.Errorf("If = %q, want %q", cur.If.ValueString(), "wan")
	}
	if cur.Enable.ValueBool() != true {
		t.Errorf("Enable = %v, want true", cur.Enable.ValueBool())
	}
	if cur.MTU.ValueInt64() != 1500 {
		t.Errorf("MTU = %d, want 1500", cur.MTU.ValueInt64())
	}
	if cur.Media.ValueString() != "1000baseT" {
		t.Errorf("Media = %q, want 1000baseT", cur.Media.ValueString())
	}
	if cur.Gateway.ValueString() != "GW_WAN" {
		t.Errorf("Gateway = %q, want GW_WAN", cur.Gateway.ValueString())
	}
	if cur.AliasAddress.ValueString() != "10.0.0.10" {
		t.Errorf("AliasAddress = %q, want 10.0.0.10", cur.AliasAddress.ValueString())
	}
	if cur.AliasSubnet.ValueInt64() != 24 {
		t.Errorf("AliasSubnet = %d, want 24", cur.AliasSubnet.ValueInt64())
	}
	if !cur.Dhcprejectfrom.Equal(rejectFrom) {
		t.Errorf("Dhcprejectfrom = %v, want %v", cur.Dhcprejectfrom, rejectFrom)
	}
	if cur.Subnet.ValueInt64() != 24 {
		t.Errorf("Subnet = %d, want 24", cur.Subnet.ValueInt64())
	}
	if !cur.AdvDHCPConfigAdvanced.ValueBool() {
		t.Errorf("AdvDHCPConfigAdvanced = false, want true")
	}
	if cur.AdvDHCPPtTimeout.ValueInt64() != 60 {
		t.Errorf("AdvDHCPPtTimeout = %d, want 60", cur.AdvDHCPPtTimeout.ValueInt64())
	}
	if cur.AdvDHCPPtRetry.ValueInt64() != 5 {
		t.Errorf("AdvDHCPPtRetry = %d, want 5", cur.AdvDHCPPtRetry.ValueInt64())
	}
	// adv_dhcp_pt_select_timeout = 0 is the SDKv2 zero value for an unset
	// optional int, so it is normalised to null rather than carried over as 0.
	if !cur.AdvDHCPPtSelectTimeout.IsNull() {
		t.Errorf("AdvDHCPPtSelectTimeout = %v, want null (from adv_dhcp_pt_select_timeout = 0)", cur.AdvDHCPPtSelectTimeout)
	}
	if cur.AdvDHCPPtReboot.ValueInt64() != 10 {
		t.Errorf("AdvDHCPPtReboot = %d, want 10", cur.AdvDHCPPtReboot.ValueInt64())
	}
	if cur.AdvDHCPPtBackoffCutoff.ValueInt64() != 15 {
		t.Errorf("AdvDHCPPtBackoffCutoff = %d, want 15", cur.AdvDHCPPtBackoffCutoff.ValueInt64())
	}
	if cur.AdvDHCPPtInitialInterval.ValueInt64() != 2 {
		t.Errorf("AdvDHCPPtInitialInterval = %d, want 2", cur.AdvDHCPPtInitialInterval.ValueInt64())
	}
	if cur.AdvDHCPSendOptions.ValueString() != "host-name" {
		t.Errorf("AdvDHCPSendOptions = %q, want host-name", cur.AdvDHCPSendOptions.ValueString())
	}
	if cur.AdvDHCPRequestOptions.ValueString() != "domain-name-servers" {
		t.Errorf("AdvDHCPRequestOptions = %q, want domain-name-servers", cur.AdvDHCPRequestOptions.ValueString())
	}
	if cur.AdvDHCPRequiredOptions.ValueString() != "domain-name" {
		t.Errorf("AdvDHCPRequiredOptions = %q, want domain-name", cur.AdvDHCPRequiredOptions.ValueString())
	}
	if cur.AdvDHCPOptionModifiers.ValueString() != "option domain-name-servers 1.1.1.1;" {
		t.Errorf("AdvDHCPOptionModifiers = %q, want option domain-name-servers 1.1.1.1;", cur.AdvDHCPOptionModifiers.ValueString())
	}

	// Renamed attributes.
	if cur.Descr.ValueString() != "WAN" {
		t.Errorf("Descr = %q, want WAN (from description)", cur.Descr.ValueString())
	}
	if cur.Spoofmac.ValueString() != "00:11:22:33:44:55" {
		t.Errorf("Spoofmac = %q, want 00:11:22:33:44:55 (from spoof_mac)", cur.Spoofmac.ValueString())
	}
	// block_private = false is the SDKv2 zero value for an unset optional
	// bool, so it is normalised to null rather than carried over as false.
	if !cur.Blockpriv.IsNull() {
		t.Errorf("Blockpriv = %v, want null (from block_private = false)", cur.Blockpriv)
	}
	if cur.Blockbogons.ValueBool() != true {
		t.Errorf("Blockbogons = %v, want true (from block_bogons)", cur.Blockbogons.ValueBool())
	}
	if cur.Dhcphostname.ValueString() != "wan-dhcp" {
		t.Errorf("Dhcphostname = %q, want wan-dhcp (from dhcp_hostname)", cur.Dhcphostname.ValueString())
	}
	if cur.Ipaddr.ValueString() != "10.0.0.1" {
		t.Errorf("Ipaddr = %q, want 10.0.0.1 (from ip_address)", cur.Ipaddr.ValueString())
	}
	if cur.Ipaddrv6.ValueString() != "2001:db8::1" {
		t.Errorf("Ipaddrv6 = %q, want 2001:db8::1 (from ip_address_v6)", cur.Ipaddrv6.ValueString())
	}
	if cur.Gatewayv6.ValueString() != "GW6_WAN" {
		t.Errorf("Gatewayv6 = %q, want GW6_WAN (from gateway_v6)", cur.Gatewayv6.ValueString())
	}
	if cur.Ipv6usev4iface.ValueBool() != true {
		t.Errorf("Ipv6usev4iface = %v, want true (from ip_v6_use_v4_iface)", cur.Ipv6usev4iface.ValueBool())
	}
	if cur.Prefix6rd.ValueString() != "2001:db8:abcd::" {
		t.Errorf("Prefix6rd = %q, want 2001:db8:abcd:: (from prefix_v6_rd)", cur.Prefix6rd.ValueString())
	}
	if cur.Gateway6rd.ValueString() != "2001:db8::1" {
		t.Errorf("Gateway6rd = %q, want 2001:db8::1 (from gateway_6_rd)", cur.Gateway6rd.ValueString())
	}
	if cur.Prefix6rdV4plen.ValueInt64() != 32 {
		t.Errorf("Prefix6rdV4plen = %d, want 32 (from prefix_6_rd_v4_plen)", cur.Prefix6rdV4plen.ValueInt64())
	}
	if cur.Track6Interface.ValueString() != "wan" {
		t.Errorf("Track6Interface = %q, want wan (from track_v6_interface)", cur.Track6Interface.ValueString())
	}
	if cur.Track6PrefixIDHex.ValueString() != "0" {
		t.Errorf("Track6PrefixIDHex = %q, want 0 (from track_v6_prefix_id_hex)", cur.Track6PrefixIDHex.ValueString())
	}
	if cur.Typev4.ValueString() != "static" {
		t.Errorf("Typev4 = %q, want static (from type = staticv4)", cur.Typev4.ValueString())
	}
	if cur.Typev6.ValueString() != "staticv6" {
		t.Errorf("Typev6 = %q, want staticv6 (from type_v6)", cur.Typev6.ValueString())
	}
	if cur.AdvDHCPConfigFileOverride.ValueBool() != true {
		t.Errorf("AdvDHCPConfigFileOverride = %v, want true", cur.AdvDHCPConfigFileOverride.ValueBool())
	}
	if cur.AdvDHCPConfigFileOverridePath.ValueString() != "/cf/conf/dhcpd.conf" {
		t.Errorf("AdvDHCPConfigFileOverridePath = %q, want /cf/conf/dhcpd.conf (from adv_dhcp_config_file_override_file)", cur.AdvDHCPConfigFileOverridePath.ValueString())
	}

	// Retyped attributes (v0 string -> v1 integer).
	if cur.Mss.ValueInt64() != 1500 {
		t.Errorf("Mss = %d, want 1500 (parsed from string %q)", cur.Mss.ValueInt64(), prior.Mss.ValueString())
	}
	if cur.Subnetv6.ValueInt64() != 64 {
		t.Errorf("Subnetv6 = %d, want 64 (parsed from string %q)", cur.Subnetv6.ValueInt64(), prior.SubnetV6.ValueString())
	}

	// v1-only attributes must stay null.
	if !cur.Mediaopt.IsNull() {
		t.Errorf("Mediaopt = %v, want null (no v0 equivalent)", cur.Mediaopt)
	}
	if !cur.AdvDHCPPtValues.IsNull() {
		t.Errorf("AdvDHCPPtValues = %v, want null (no v0 equivalent)", cur.AdvDHCPPtValues)
	}
	if !cur.Slaacusev4iface.IsNull() {
		t.Errorf("Slaacusev4iface = %v, want null (no v0 equivalent)", cur.Slaacusev4iface)
	}
}

func TestNetworkInterfaceModelV0ToCurrentEmptyStringRetypes(t *testing.T) {
	ctx := context.Background()

	// SDKv2 stores "" for unset optional strings; the retyped integer
	// attributes must land as null, not as a parse error.
	prior := networkInterfaceModelV0{
		If:          types.StringValue("lan"),
		Description: types.StringValue("LAN"),
		Mss:         types.StringValue(""),
		SubnetV6:    types.StringValue(""),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics for empty strings: %s", diags)
	}
	if !cur.Mss.IsNull() {
		t.Errorf("Mss = %v, want null for empty string", cur.Mss)
	}
	if !cur.Subnetv6.IsNull() {
		t.Errorf("Subnetv6 = %v, want null for empty string", cur.Subnetv6)
	}
}

func TestNetworkInterfaceModelV0ToCurrentTypeMapping(t *testing.T) {
	ctx := context.Background()

	// The v0 `type` was validated with StringInSlice(["staticv4", "dhcp"]),
	// so prior state can only hold those two values or "" (the SDKv2 zero
	// value). The v1 `typev4` is Required and validated with
	// OneOf("static", "dhcp", "none"), hence the explicit mapping.
	for _, tc := range []struct {
		name       string
		typev4     string
		typev6     string
		wantTypev4 string
		wantTypev6 string
	}{
		{name: "staticv4", typev4: "staticv4", typev6: "staticv6", wantTypev4: "static", wantTypev6: "staticv6"},
		{name: "dhcp", typev4: "dhcp", typev6: "dhcp6", wantTypev4: "dhcp", wantTypev6: "dhcp6"},
		{name: "unset", typev4: "", typev6: "", wantTypev4: "none", wantTypev6: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := networkInterfaceModelV0{
				If:          types.StringValue("wan"),
				Description: types.StringValue("WAN"),
				Type:        types.StringValue(tc.typev4),
				TypeV6:      types.StringValue(tc.typev6),
			}

			cur, diags := prior.toCurrent(ctx)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}
			if diags.WarningsCount() != 0 {
				t.Errorf("unexpected warnings: %s", diags.Warnings())
			}
			if cur.Typev4.ValueString() != tc.wantTypev4 {
				t.Errorf("Typev4 = %q, want %q (from type = %q)", cur.Typev4.ValueString(), tc.wantTypev4, tc.typev4)
			}
			// typev4 is Required in v1, so it must never end up null.
			if cur.Typev4.IsNull() {
				t.Errorf("Typev4 is null; the v1 attribute is Required")
			}
			if cur.Typev6.ValueString() != tc.wantTypev6 {
				t.Errorf("Typev6 = %q, want %q (from type_v6 = %q)", cur.Typev6.ValueString(), tc.wantTypev6, tc.typev6)
			}
			if tc.typev6 == "" && !cur.Typev6.IsNull() {
				t.Errorf("Typev6 = %v, want null for unset type_v6", cur.Typev6)
			}
		})
	}
}

func TestNetworkInterfaceModelV0ToCurrentUnknownTypeWarns(t *testing.T) {
	ctx := context.Background()

	// Nothing outside the v0 domain should reach state, but if it does it is
	// reported and replaced with "none" rather than failing the v1 OneOf
	// validator on the next plan.
	prior := networkInterfaceModelV0{
		If:          types.StringValue("wan"),
		Description: types.StringValue("WAN"),
		Type:        types.StringValue("ppp"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %s", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1: %s", diags.WarningsCount(), diags)
	}
	if cur.Typev4.ValueString() != "none" {
		t.Errorf("Typev4 = %q, want none for an out-of-domain v0 type", cur.Typev4.ValueString())
	}
}

func TestNetworkInterfaceModelV0ToCurrentZeroValueNormalisation(t *testing.T) {
	ctx := context.Background()

	// SDKv2 persists unset optional bools as false and unset optional ints as
	// 0; both must become null so an unconfigured attribute does not show a
	// false→null / 0→null diff on the next plan.
	prior := networkInterfaceModelV0{
		If:                        types.StringValue("lan"),
		Description:               types.StringValue("LAN"),
		Enable:                    types.BoolValue(false),
		BlockPrivate:              types.BoolValue(false),
		BlockBogons:               types.BoolValue(false),
		IpV6UseV4Iface:            types.BoolValue(false),
		AdvDhcpConfigAdvanced:     types.BoolValue(false),
		AdvDhcpConfigFileOverride: types.BoolValue(false),
		Mtu:                       types.Int64Value(0),
		AliasSubnet:               types.Int64Value(0),
		AdvDhcpPtTimeout:          types.Int64Value(0),
		AdvDhcpPtRetry:            types.Int64Value(0),
		AdvDhcpPtSelectTimeout:    types.Int64Value(0),
		AdvDhcpPtReboot:           types.Int64Value(0),
		AdvDhcpPtBackoffCutoff:    types.Int64Value(0),
		AdvDhcpPtInitialInterval:  types.Int64Value(0),
		Prefix6RdV4Plen:           types.Int64Value(0),
		Subnet:                    types.Int64Value(0),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	for name, v := range map[string]types.Bool{
		"Enable":                    cur.Enable,
		"Blockpriv":                 cur.Blockpriv,
		"Blockbogons":               cur.Blockbogons,
		"Ipv6usev4iface":            cur.Ipv6usev4iface,
		"AdvDHCPConfigAdvanced":     cur.AdvDHCPConfigAdvanced,
		"AdvDHCPConfigFileOverride": cur.AdvDHCPConfigFileOverride,
	} {
		if !v.IsNull() {
			t.Errorf("%s = %v, want null for the SDKv2 false zero value", name, v)
		}
	}

	for name, v := range map[string]types.Int64{
		"MTU":                      cur.MTU,
		"AliasSubnet":              cur.AliasSubnet,
		"AdvDHCPPtTimeout":         cur.AdvDHCPPtTimeout,
		"AdvDHCPPtRetry":           cur.AdvDHCPPtRetry,
		"AdvDHCPPtSelectTimeout":   cur.AdvDHCPPtSelectTimeout,
		"AdvDHCPPtReboot":          cur.AdvDHCPPtReboot,
		"AdvDHCPPtBackoffCutoff":   cur.AdvDHCPPtBackoffCutoff,
		"AdvDHCPPtInitialInterval": cur.AdvDHCPPtInitialInterval,
		"Prefix6rdV4plen":          cur.Prefix6rdV4plen,
	} {
		if !v.IsNull() {
			t.Errorf("%s = %v, want null for the SDKv2 0 zero value", name, v)
		}
	}

	// `subnet` was Optional in v0 and Required in v2, so — like `ipaddr` — an
	// unset v0 value is normalised to null and fails the next plan loudly.
	if !cur.Subnet.IsNull() {
		t.Errorf("Subnet = %v, want null for the SDKv2 0 zero value", cur.Subnet)
	}
}

func TestNetworkInterfaceModelV0ToCurrentInvalidInteger(t *testing.T) {
	ctx := context.Background()

	prior := networkInterfaceModelV0{
		If:          types.StringValue("wan"),
		Description: types.StringValue("WAN"),
		Mss:         types.StringValue("not-a-number"),
	}

	cur, diags := prior.toCurrent(ctx)
	if !diags.HasError() {
		t.Fatalf("expected a diagnostic for non-numeric mss, got none")
	}
	if !cur.Mss.IsNull() {
		t.Errorf("Mss = %v, want null after parse failure", cur.Mss)
	}
}

func TestNetworkInterfaceUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	// The current schema must declare version 1 so Terraform knows the
	// 0 -> 1 upgrader applies to existing pfsense_interface state.
	var schemaResp resource.SchemaResponse
	(&networkInterfaceResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("schema Version = %d, want 1", schemaResp.Schema.Version)
	}

	prior := networkInterfaceModelV0{
		If:             types.StringValue("wan"),
		Description:    types.StringValue("WAN"),
		Type:           types.StringValue("staticv4"),
		IpAddress:      types.StringValue("10.0.0.1"),
		Subnet:         types.Int64Value(24),
		Mss:            types.StringValue("1500"),
		SubnetV6:       types.StringValue("64"),
		Enable:         types.BoolValue(true),
		DhcpRejectFrom: types.ListNull(types.StringType),
	}

	// Replicate what the framework does before invoking the upgrader:
	// decode the prior raw state against the PriorSchema into req.State.
	var priorState tfsdk.State
	priorState.Schema = networkInterfacePriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The old SDKv2 state always carries the implicit id attribute; for
	// pfsense_interface that was the interface's descriptive name, which the
	// upgrader must NOT carry over — the v1 contract is id == if.
	rawJSON, err := json.Marshal(map[string]any{"id": "WAN"})
	if err != nil {
		t.Fatalf("marshaling raw state: %s", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: rawJSON},
		State:    &priorState,
	}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	(&networkInterfaceResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got networkInterfaceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}

	// The v1 id is the interface key, never the old descriptive-name id.
	if got.ID.ValueString() != "wan" {
		t.Errorf("ID = %q, want the interface key %q", got.ID.ValueString(), "wan")
	}
	if got.If.ValueString() != "wan" {
		t.Errorf("If = %q, want wan", got.If.ValueString())
	}
	if got.Descr.ValueString() != "WAN" {
		t.Errorf("Descr = %q, want WAN", got.Descr.ValueString())
	}
	if got.Typev4.ValueString() != "static" {
		t.Errorf("Typev4 = %q, want static", got.Typev4.ValueString())
	}
	if got.Ipaddr.ValueString() != "10.0.0.1" {
		t.Errorf("Ipaddr = %q, want 10.0.0.1", got.Ipaddr.ValueString())
	}
	if got.Mss.ValueInt64() != 1500 {
		t.Errorf("Mss = %d, want 1500", got.Mss.ValueInt64())
	}
	if got.Subnetv6.ValueInt64() != 64 {
		t.Errorf("Subnetv6 = %d, want 64", got.Subnetv6.ValueInt64())
	}
	if !got.Slaacusev4iface.IsNull() {
		t.Errorf("Slaacusev4iface = %v, want null", got.Slaacusev4iface)
	}
}

func TestNetworkInterfaceUpgradeStateV0To1IDIsInterfaceKey(t *testing.T) {
	ctx := context.Background()

	prior := networkInterfaceModelV0{
		If:             types.StringValue("opt1"),
		Description:    types.StringValue("OPT1"),
		DhcpRejectFrom: types.ListNull(types.StringType),
	}

	var priorState tfsdk.State
	priorState.Schema = networkInterfacePriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The current schema must declare version 1.
	var schemaResp resource.SchemaResponse
	(&networkInterfaceResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}

	// No RawState: the natural key (`if`) is the new id either way.
	req := resource.UpgradeStateRequest{State: &priorState}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	(&networkInterfaceResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got networkInterfaceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}
	if got.ID.ValueString() != "opt1" {
		t.Errorf("ID = %q, want the natural key %q", got.ID.ValueString(), "opt1")
	}
}

func TestNetworkInterfaceUpgradeStateMap(t *testing.T) {
	r := &networkInterfaceResource{}
	upgraders := r.UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatalf("no state upgrader registered for version 0")
	}
	if upgrader.PriorSchema == nil {
		t.Fatalf("PriorSchema is nil")
	}
	if upgrader.PriorSchema.Attributes["description"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute description")
	}
	if upgrader.PriorSchema.Attributes["mss"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute mss")
	}
	// The implicit SDKv2 id must NOT be part of the prior schema.
	if upgrader.PriorSchema.Attributes["id"] != nil {
		t.Fatalf("PriorSchema must not contain the implicit id attribute")
	}
}

// TestNetworkInterfaceUpgradeStateV0To1RawJSON decodes the prior state from
// raw JSON against networkInterfacePriorSchemaV0 (not via priorState.Set), so
// that a type or name mismatch between the PriorSchema and what SDKv2 actually
// wrote would fail the unmarshal. Only a representative subset of the large v0
// model is supplied; the rest is handled by IgnoreUndefinedAttributes. It
// asserts the v1 id derives from `if` (the raw id is ignored), description→
// descr, ip_address→ipaddr, and that an unset v0 `type` becomes "none".
func TestNetworkInterfaceUpgradeStateV0To1RawJSON(t *testing.T) {
	ctx := context.Background()

	rawJSON, err := json.Marshal(map[string]any{
		"id":          "stale_id",
		"if":          "wan",
		"description": "WAN",
		"type":        "",
		"ip_address":  "10.0.0.1",
		"subnet":      24,
		"spoof_mac":   "00:11:22:33:44:55",
	})
	if err != nil {
		t.Fatalf("marshal prior raw state: %v", err)
	}
	raw := &tfprotov6.RawState{JSON: rawJSON}

	priorValue, err := raw.UnmarshalWithOpts(
		networkInterfacePriorSchemaV0.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
		},
	)
	if err != nil {
		t.Fatalf("unmarshal prior raw state: %v", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: raw,
		State:    &tfsdk.State{Raw: priorValue, Schema: networkInterfacePriorSchemaV0},
	}
	var resp resource.UpgradeStateResponse

	var r networkInterfaceResource
	var sreq resource.SchemaRequest
	var sresp resource.SchemaResponse
	r.Schema(ctx, sreq, &sresp)
	resp.State.Schema = sresp.Schema

	r.upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade returned diagnostics: %s", resp.Diagnostics)
	}

	var got networkInterfaceModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("decode upgraded state: %s", resp.Diagnostics)
	}

	// The v1 id is the natural key `if`; the raw id is ignored.
	if got, want := got.ID.ValueString(), "wan"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := got.If.ValueString(), "wan"; got != want {
		t.Errorf("If = %q, want %q", got, want)
	}
	// description -> descr
	if got, want := got.Descr.ValueString(), "WAN"; got != want {
		t.Errorf("Descr = %q, want %q", got, want)
	}
	// ip_address -> ipaddr
	if got, want := got.Ipaddr.ValueString(), "10.0.0.1"; got != want {
		t.Errorf("Ipaddr = %q, want %q", got, want)
	}
	if got, want := got.Subnet.ValueInt64(), int64(24); got != want {
		t.Errorf("Subnet = %d, want %d", got, want)
	}
	// spoof_mac -> spoofmac
	if got, want := got.Spoofmac.ValueString(), "00:11:22:33:44:55"; got != want {
		t.Errorf("Spoofmac = %q, want %q", got, want)
	}
	// An unset v0 `type` (the SDKv2 "" zero value) becomes the explicit
	// "none" because typev4 is Required in v1 and can never be null.
	if got, want := got.Typev4.ValueString(), "none"; got != want {
		t.Errorf("Typev4 = %q, want %q (from an empty v0 type)", got, want)
	}
}
