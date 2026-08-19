package provider

import (
	"context"
	"encoding/json"
	"strings"
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

	// deny_unknown=false is the SDKv2 zero value for an unset optional bool and
	// is indistinguishable from an explicit off, so it normalises to null and
	// the first Read re-populates the real value.
	cur, diags := (dhcpServerModelV0{
		Interface:    types.StringValue("lan"),
		DenyUnknown:  types.BoolValue(false),
		MaxLeaseTime: types.StringValue(""),
	}).toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !cur.DenyUnknown.IsNull() {
		t.Errorf("DenyUnknown = %v, want null (from deny_unknown=false)", cur.DenyUnknown)
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

func TestDhcpServerModelV0ToCurrentZeroValueNormalisation(t *testing.T) {
	ctx := context.Background()

	// SDKv2 persists unset optional bools as false and unset optional ints as
	// 0; both must become null so they do not show a spurious diff.
	cur, diags := (dhcpServerModelV0{
		Interface:        types.StringValue("lan"),
		Enable:           types.BoolValue(false),
		DefaultLeaseTime: types.Int64Value(0),
	}).toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !cur.Enable.IsNull() {
		t.Errorf("Enable = %v, want null for the SDKv2 false zero value", cur.Enable)
	}
	if !cur.DefaultLease.IsNull() {
		t.Errorf("DefaultLease = %v, want null for the SDKv2 0 zero value", cur.DefaultLease)
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
	priorState.Schema = dhcpServerPriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The old SDKv2 state always carries the implicit id attribute. Give it a
	// value that contradicts the natural key so the assertion proves the v1 id
	// is derived from `interface`, not read from the raw id.
	rawJSON, err := json.Marshal(map[string]any{"id": "stale_id"})
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

	(&dhcpServerResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got dhcpServerModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}

	// The v1 id is the natural key (`interface`), never the raw id (which the
	// fixture set to "stale_id" to prove the upgrader ignores it).
	if got.ID.ValueString() != "lan" {
		t.Errorf("ID = %q, want natural key %q", got.ID.ValueString(), "lan")
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
	priorState.Schema = dhcpServerPriorSchemaV0
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

	(&dhcpServerResource{}).upgradeStateV0To1(ctx, req, &resp)
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

// TestDhcpServerModelV0ToCurrentDroppedAttributeWarnings covers the v0
// attributes the v1 pfsense_services_dhcp_server resource does not model.
// They are access-control / DHCP option data, so dropping them must be
// surfaced rather than silent.
func TestDhcpServerModelV0ToCurrentDroppedAttributeWarnings(t *testing.T) {
	ctx := context.Background()

	list := func(v string) types.List {
		return types.ListValueMust(types.StringType, []attr.Value{types.StringValue(v)})
	}

	for _, tc := range []struct {
		name  string
		prior dhcpServerModelV0
	}{
		{"domain_search_list", dhcpServerModelV0{DomainSearchList: list("example.com")}},
		{"mac_allow_list", dhcpServerModelV0{MacAllowList: list("00:11:22:33:44:55")}},
		{"mac_deny_list", dhcpServerModelV0{MacDenyList: list("00:11:22:33:44:55")}},
		{"ignore_bootp", dhcpServerModelV0{IgnoreBootp: types.BoolValue(true)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prior := tc.prior
			prior.Interface = types.StringValue("lan")

			_, diags := prior.toCurrent(ctx)
			if diags.HasError() {
				t.Fatalf("unexpected error diagnostics: %s", diags)
			}
			if len(diags) != 1 {
				t.Fatalf("diags = %d entries, want 1 warning: %s", len(diags), diags)
			}
			if got := diags[0].Severity().String(); got != "Warning" {
				t.Errorf("diag severity = %s, want Warning", got)
			}
			if !strings.Contains(diags[0].Detail(), tc.name) {
				t.Errorf("warning detail %q does not name the dropped attribute %q", diags[0].Detail(), tc.name)
			}
			// The replacement resource must be named so the warning is actionable.
			if !strings.Contains(diags[0].Detail(), "pfsense_services_dhcp_address_pool") {
				t.Errorf("warning detail %q does not point at pfsense_services_dhcp_address_pool", diags[0].Detail())
			}
		})
	}

	// Empty lists and ignore_bootp=false (the SDKv2 zero values for unset
	// optional attributes) must not warn.
	_, diags := (dhcpServerModelV0{
		Interface:        types.StringValue("lan"),
		DomainSearchList: types.ListValueMust(types.StringType, []attr.Value{}),
		MacAllowList:     types.ListNull(types.StringType),
		IgnoreBootp:      types.BoolValue(false),
	}).toCurrent(ctx)
	if len(diags) != 0 {
		t.Errorf("unset v0 attributes produced %d diagnostics, want none: %s", len(diags), diags)
	}
}
