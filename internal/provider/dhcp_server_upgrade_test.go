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

func TestDhcpServerModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	dnsServers := types.ListValueMust(types.StringType, []attr.Value{
		types.StringValue("1.1.1.1"),
		types.StringValue("8.8.8.8"),
	})

	prior := dhcpServerModelV0{
		DefaultLeaseTime: types.Int64Value(7200),
		DenyUnknown:      types.BoolValue(true),
		DNSServer:        dnsServers,
		Domain:           types.StringValue("example.com"),
		DomainSearchList: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("corp.example.com")}),
		Enable:           types.BoolValue(true),
		Gateway:          types.StringValue("10.0.0.1"),
		IgnoreBootp:      types.BoolValue(false),
		Interface:        types.StringValue("lan"),
		MacAllowList:     types.ListValueMust(types.StringType, []attr.Value{types.StringValue("00:11:22:33:44:55")}),
		MacDenyList:      types.ListValueMust(types.StringType, []attr.Value{types.StringValue("aa:bb:cc:dd:ee:ff")}),
		MaxLeaseTime:     types.StringValue("86400"),
		RangeFrom:        types.StringValue("10.0.0.100"),
		RangeTo:          types.StringValue("10.0.0.200"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	// Identity pass-through.
	if cur.Interface.ValueString() != "lan" {
		t.Errorf("Interface = %q, want lan", cur.Interface.ValueString())
	}
	if cur.Enable.ValueBool() != true {
		t.Errorf("Enable = %v, want true", cur.Enable.ValueBool())
	}
	if cur.RangeFrom.ValueString() != "10.0.0.100" {
		t.Errorf("RangeFrom = %q, want 10.0.0.100", cur.RangeFrom.ValueString())
	}
	if cur.RangeTo.ValueString() != "10.0.0.200" {
		t.Errorf("RangeTo = %q, want 10.0.0.200", cur.RangeTo.ValueString())
	}
	if cur.Domain.ValueString() != "example.com" {
		t.Errorf("Domain = %q, want example.com", cur.Domain.ValueString())
	}
	if cur.Gateway.ValueString() != "10.0.0.1" {
		t.Errorf("Gateway = %q, want 10.0.0.1", cur.Gateway.ValueString())
	}
	if !cur.DNSServer.Equal(dnsServers) {
		t.Errorf("DNSServer = %v, want %v", cur.DNSServer, dnsServers)
	}

	// Renamed attributes.
	if cur.DefaultLease.ValueInt64() != 7200 {
		t.Errorf("DefaultLease = %d, want 7200 (from default_lease_time)", cur.DefaultLease.ValueInt64())
	}

	// Retyped attributes (v0 string -> v1 integer, v0 bool -> v1 string).
	if cur.MaxLease.ValueInt64() != 86400 {
		t.Errorf("MaxLease = %d, want 86400 (parsed from string %q)", cur.MaxLease.ValueInt64(), prior.MaxLeaseTime.ValueString())
	}
	if cur.DenyUnknown.ValueString() != "enabled" {
		t.Errorf("DenyUnknown = %q, want enabled (from deny_unknown=true)", cur.DenyUnknown.ValueString())
	}

	// v1-only attributes must stay null.
	if !cur.WINSServer.IsNull() {
		t.Errorf("WINSServer = %v, want null (no v0 equivalent)", cur.WINSServer)
	}
	if !cur.NTPServer.IsNull() {
		t.Errorf("NTPServer = %v, want null (no v0 equivalent)", cur.NTPServer)
	}
	if !cur.StaticMap.IsNull() {
		t.Errorf("StaticMap = %v, want null (no v0 equivalent)", cur.StaticMap)
	}
}

func TestDhcpServerModelV0ToCurrentDenyUnknownFalseAndNull(t *testing.T) {
	ctx := context.Background()

	// deny_unknown=false -> "disabled"
	cur, diags := (dhcpServerModelV0{
		Interface:    types.StringValue("lan"),
		DenyUnknown:  types.BoolValue(false),
		MaxLeaseTime: types.StringValue(""),
	}).toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if cur.DenyUnknown.ValueString() != "disabled" {
		t.Errorf("DenyUnknown = %q, want disabled (from deny_unknown=false)", cur.DenyUnknown.ValueString())
	}
	// max_lease_time="" -> null, not a parse error.
	if !cur.MaxLease.IsNull() {
		t.Errorf("MaxLease = %v, want null for empty string", cur.MaxLease)
	}

	// deny_unknown unset (null) -> null.
	cur, diags = (dhcpServerModelV0{
		Interface: types.StringValue("lan"),
	}).toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !cur.DenyUnknown.IsNull() {
		t.Errorf("DenyUnknown = %v, want null (from null deny_unknown)", cur.DenyUnknown)
	}
}

func TestDhcpServerModelV0ToCurrentInvalidInteger(t *testing.T) {
	ctx := context.Background()

	cur, diags := (dhcpServerModelV0{
		Interface:    types.StringValue("lan"),
		MaxLeaseTime: types.StringValue("abc"),
	}).toCurrent(ctx)
	if !diags.HasError() {
		t.Fatalf("expected a diagnostic for non-numeric max_lease_time, got none")
	}
	if !cur.MaxLease.IsNull() {
		t.Errorf("MaxLease = %v, want null after parse failure", cur.MaxLease)
	}
}

func TestDhcpServerUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	// The current schema must declare version 1.
	var schemaResp resource.SchemaResponse
	(&dhcpServerResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("schema Version = %d, want 1", schemaResp.Schema.Version)
	}

	prior := dhcpServerModelV0{
		Interface:        types.StringValue("lan"),
		Enable:           types.BoolValue(true),
		RangeFrom:        types.StringValue("10.0.0.100"),
		RangeTo:          types.StringValue("10.0.0.200"),
		Domain:           types.StringValue("example.com"),
		DefaultLeaseTime: types.Int64Value(7200),
		MaxLeaseTime:     types.StringValue("86400"),
		Gateway:          types.StringValue("10.0.0.1"),
		DenyUnknown:      types.BoolValue(true),
		DNSServer:        types.ListNull(types.StringType),
		DomainSearchList: types.ListNull(types.StringType),
		MacAllowList:     types.ListNull(types.StringType),
		MacDenyList:      types.ListNull(types.StringType),
	}

	var priorState tfsdk.State
	priorState.Schema = dhcpServerPriorSchema()
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The old SDKv2 state always carries the implicit id attribute; for
	// pfsense_dhcp_server the id equals the interface value.
	rawJSON, err := json.Marshal(map[string]any{"id": "lan"})
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

	dhcpServerUpgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got dhcpServerModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}

	if got.ID.ValueString() != "lan" {
		t.Errorf("ID = %q, want prior id %q", got.ID.ValueString(), "lan")
	}
	if got.Interface.ValueString() != "lan" {
		t.Errorf("Interface = %q, want lan", got.Interface.ValueString())
	}
	if got.DefaultLease.ValueInt64() != 7200 {
		t.Errorf("DefaultLease = %d, want 7200", got.DefaultLease.ValueInt64())
	}
	if got.MaxLease.ValueInt64() != 86400 {
		t.Errorf("MaxLease = %d, want 86400", got.MaxLease.ValueInt64())
	}
	if got.DenyUnknown.ValueString() != "enabled" {
		t.Errorf("DenyUnknown = %q, want enabled", got.DenyUnknown.ValueString())
	}
	if !got.WINSServer.IsNull() {
		t.Errorf("WINSServer = %v, want null", got.WINSServer)
	}
}

func TestDhcpServerUpgradeStateV0To1FallbackID(t *testing.T) {
	ctx := context.Background()

	prior := dhcpServerModelV0{
		Interface:        types.StringValue("opt1"),
		DNSServer:        types.ListNull(types.StringType),
		DomainSearchList: types.ListNull(types.StringType),
		MacAllowList:     types.ListNull(types.StringType),
		MacDenyList:      types.ListNull(types.StringType),
	}

	var priorState tfsdk.State
	priorState.Schema = dhcpServerPriorSchema()
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The current schema must declare version 1.
	var schemaResp resource.SchemaResponse
	(&dhcpServerResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}

	// No RawState: the natural key (`interface`) must become the new id.
	req := resource.UpgradeStateRequest{State: &priorState}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	dhcpServerUpgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got dhcpServerModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}
	if got.ID.ValueString() != "opt1" {
		t.Errorf("ID = %q, want fallback to natural key %q", got.ID.ValueString(), "opt1")
	}
}

func TestDhcpServerUpgradeStateMap(t *testing.T) {
	r := &dhcpServerResource{}
	upgraders := r.UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatalf("no state upgrader registered for version 0")
	}
	if upgrader.PriorSchema == nil {
		t.Fatalf("PriorSchema is nil")
	}
	if upgrader.PriorSchema.Attributes["interface"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute interface")
	}
	// max_lease_time was a TypeString in v0 and must decode as a string.
	if upgrader.PriorSchema.Attributes["max_lease_time"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute max_lease_time")
	}
	// The implicit SDKv2 id must NOT be part of the prior schema.
	if upgrader.PriorSchema.Attributes["id"] != nil {
		t.Fatalf("PriorSchema must not contain the implicit id attribute")
	}
}
