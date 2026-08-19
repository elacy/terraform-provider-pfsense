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
	if cur.AdvDHCPPtSelectTimeout.ValueInt64() != 0 {
		t.Errorf("AdvDHCPPtSelectTimeout = %d, want 0", cur.AdvDHCPPtSelectTimeout.ValueInt64())
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
	if cur.Blockpriv.ValueBool() != false {
		t.Errorf("Blockpriv = %v, want false (from block_private)", cur.Blockpriv.ValueBool())
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
	if cur.Typev4.ValueString() != "staticv4" {
		t.Errorf("Typev4 = %q, want staticv4 (from type)", cur.Typev4.ValueString())
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
	if got.Typev4.ValueString() != "staticv4" {
		t.Errorf("Typev4 = %q, want staticv4", got.Typev4.ValueString())
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
