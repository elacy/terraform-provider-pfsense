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

// TestDHCPStaticMappingUpgradeStateV0To1 exercises the full state-upgrade path for
// pfsense_dhcp_static_mapping → pfsense_services_dhcp_static_mapping: prior
// raw state (including the implicit "id") is decoded via the PriorSchema,
// mapped to the version-1 model, and encoded against the CURRENT schema.
func TestDHCPStaticMappingUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	rawJSON, err := json.Marshal(map[string]any{
		"id":                     "wan.00:11:22:33:44:55",
		"interface":              "wan",
		"mac":                    "00:11:22:33:44:55",
		"client_identifier":      "01:aa:bb",
		"ip_address":             "192.168.1.100",
		"host_name":              "printer",
		"description":            "office printer",
		"gateway":                "192.168.1.1",
		"domain":                 "example.com",
		"domain_search_list":     []string{"example.com", "example.net"},
		"dns_servers":            []string{"1.1.1.1"},
		"arp_table_static_entry": true,
	})
	if err != nil {
		t.Fatalf("marshal prior raw state: %v", err)
	}
	raw := &tfprotov6.RawState{JSON: rawJSON}

	// priorResourceID must surface the implicit id not present in PriorSchema.
	if got, want := priorResourceID(raw), "wan.00:11:22:33:44:55"; got != want {
		t.Errorf("priorResourceID() = %q, want %q", got, want)
	}

	priorValue, err := raw.UnmarshalWithOpts(
		dhcpStaticMappingPriorSchemaV0.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
		},
	)
	if err != nil {
		t.Fatalf("unmarshal prior raw state: %v", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: raw,
		State:    &tfsdk.State{Raw: priorValue, Schema: dhcpStaticMappingPriorSchemaV0},
	}
	var resp resource.UpgradeStateResponse

	var r dhcpStaticMappingResource
	var sreq resource.SchemaRequest
	var sresp resource.SchemaResponse
	r.Schema(ctx, sreq, &sresp)
	resp.State.Schema = sresp.Schema

	r.upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade returned diagnostics: %s", resp.Diagnostics)
	}

	var got dhcpStaticMappingModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("decode upgraded state: %s", resp.Diagnostics)
	}

	if got, want := got.ID.ValueString(), "wan|00:11:22:33:44:55"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := got.ParentID.ValueString(), "wan"; got != want {
		t.Errorf("parent_id = %q, want %q", got, want)
	}
	if got, want := got.MAC.ValueString(), "00:11:22:33:44:55"; got != want {
		t.Errorf("mac = %q, want %q", got, want)
	}
	if got, want := got.Ipaddr.ValueString(), "192.168.1.100"; got != want {
		t.Errorf("ipaddr = %q, want %q", got, want)
	}
	if got, want := got.CID.ValueString(), "01:aa:bb"; got != want {
		t.Errorf("cid = %q, want %q", got, want)
	}
	if got, want := got.Hostname.ValueString(), "printer"; got != want {
		t.Errorf("hostname = %q, want %q", got, want)
	}
	if got, want := got.Descr.ValueString(), "office printer"; got != want {
		t.Errorf("descr = %q, want %q", got, want)
	}
	if got, want := got.Gateway.ValueString(), "192.168.1.1"; got != want {
		t.Errorf("gateway = %q, want %q", got, want)
	}
	if got, want := got.Domain.ValueString(), "example.com"; got != want {
		t.Errorf("domain = %q, want %q", got, want)
	}

	if got, want := listStringValues(t, got.Domainsearchlist), []string{"example.com", "example.net"}; !equalStrings(got, want) {
		t.Errorf("domainsearchlist = %v, want %v", got, want)
	}
	if got, want := listStringValues(t, got.DNSServer), []string{"1.1.1.1"}; !equalStrings(got, want) {
		t.Errorf("dnsserver = %v, want %v", got, want)
	}
	if got := got.ARPTableStaticEntry.ValueBool(); !got {
		t.Errorf("arp_table_static_entry = %v, want true", got)
	}

	// Attributes new in version 1 must be left null.
	if !got.DefaultLeaseTime.IsNull() || !got.MaxLeaseTime.IsNull() || !got.WINSServer.IsNull() || !got.NTPServer.IsNull() {
		t.Errorf("version-1-only attributes must be null, got defaultleasetime=%v maxleasetime=%v winsserver=%v ntpserver=%v",
			got.DefaultLeaseTime, got.MaxLeaseTime, got.WINSServer, got.NTPServer)
	}
}

// TestDHCPStaticMappingUpgradeStateV0To1MissingID covers the case where the
// prior raw state carries no implicit "id": the upgrader derives the id from
// the natural-key attributes (interface + mac) unconditionally — it never
// consults req.RawState.
func TestDHCPStaticMappingUpgradeStateV0To1MissingID(t *testing.T) {
	ctx := context.Background()

	rawJSON, err := json.Marshal(map[string]any{
		"interface": "lan",
		"mac":       "aa:bb:cc:dd:ee:ff",
	})
	if err != nil {
		t.Fatalf("marshal prior raw state: %v", err)
	}
	raw := &tfprotov6.RawState{JSON: rawJSON}
	if got := priorResourceID(raw); got != "" {
		t.Fatalf("priorResourceID() = %q, want empty", got)
	}

	priorValue, err := raw.UnmarshalWithOpts(
		dhcpStaticMappingPriorSchemaV0.Type().TerraformType(ctx),
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{IgnoreUndefinedAttributes: true},
		},
	)
	if err != nil {
		t.Fatalf("unmarshal prior raw state: %v", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: raw,
		State:    &tfsdk.State{Raw: priorValue, Schema: dhcpStaticMappingPriorSchemaV0},
	}
	var resp resource.UpgradeStateResponse

	var r dhcpStaticMappingResource
	var sreq resource.SchemaRequest
	var sresp resource.SchemaResponse
	r.Schema(ctx, sreq, &sresp)
	resp.State.Schema = sresp.Schema

	r.upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade returned diagnostics: %s", resp.Diagnostics)
	}

	var got dhcpStaticMappingModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if resp.Diagnostics.HasError() {
		t.Fatalf("decode upgraded state: %s", resp.Diagnostics)
	}
	if got, want := got.ID.ValueString(), "lan|aa:bb:cc:dd:ee:ff"; got != want {
		t.Errorf("ID = %q, want %q", got, want)
	}
	if got, want := got.ParentID.ValueString(), "lan"; got != want {
		t.Errorf("parent_id = %q, want %q", got, want)
	}
}

// TestDHCPStaticMappingModelV0ToCurrentZeroValueNormalisation unit-tests the
// SDKv2 zero-value (false) → null mapping on a representative version-0 model.
func TestDHCPStaticMappingModelV0ToCurrentZeroValueNormalisation(t *testing.T) {
	ctx := context.Background()

	// arp_table_static_entry = false is the SDKv2 zero value for an unset
	// optional bool, so it is normalised to null.
	current, diags := (dhcpStaticMappingModelV0{
		Interface:           types.StringValue("lan"),
		MAC:                 types.StringValue("00:11:22:33:44:55"),
		ARPTableStaticEntry: types.BoolValue(false),
	}).toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !current.ARPTableStaticEntry.IsNull() {
		t.Errorf("ARPTableStaticEntry = %v, want null for the SDKv2 false zero value", current.ARPTableStaticEntry)
	}
}

// TestDHCPStaticMappingModelV0ToCurrent unit-tests the pure mapping on a
// representative version-0 model.
func TestDHCPStaticMappingModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	prior := dhcpStaticMappingModelV0{
		Interface:        types.StringValue("wan"),
		MAC:              types.StringValue("00:11:22:33:44:55"),
		ClientIdentifier: types.StringValue("01:aa:bb"),
		IPAddress:        types.StringValue("192.168.1.100"),
		HostName:         types.StringValue("printer"),
		Description:      types.StringValue("office printer"),
		Gateway:          types.StringValue("192.168.1.1"),
		Domain:           types.StringValue("example.com"),
		DomainSearchList: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("example.com"),
		}),
		DNSServers: types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("1.1.1.1"),
		}),
		ARPTableStaticEntry: types.BoolValue(true),
	}

	current, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("toCurrent returned diagnostics: %s", diags)
	}

	if got, want := current.ParentID.ValueString(), "wan"; got != want {
		t.Errorf("parent_id = %q, want %q", got, want)
	}
	if got, want := current.MAC.ValueString(), "00:11:22:33:44:55"; got != want {
		t.Errorf("mac = %q, want %q", got, want)
	}
	if got, want := current.CID.ValueString(), "01:aa:bb"; got != want {
		t.Errorf("cid = %q, want %q", got, want)
	}
	if got, want := current.Ipaddr.ValueString(), "192.168.1.100"; got != want {
		t.Errorf("ipaddr = %q, want %q", got, want)
	}
	if got, want := current.Hostname.ValueString(), "printer"; got != want {
		t.Errorf("hostname = %q, want %q", got, want)
	}
	if got, want := current.Descr.ValueString(), "office printer"; got != want {
		t.Errorf("descr = %q, want %q", got, want)
	}
	if got, want := current.Gateway.ValueString(), "192.168.1.1"; got != want {
		t.Errorf("gateway = %q, want %q", got, want)
	}
	if got, want := current.Domain.ValueString(), "example.com"; got != want {
		t.Errorf("domain = %q, want %q", got, want)
	}
	if !current.Domainsearchlist.Equal(prior.DomainSearchList) {
		t.Errorf("domainsearchlist = %v, want %v", current.Domainsearchlist, prior.DomainSearchList)
	}
	if !current.DNSServer.Equal(prior.DNSServers) {
		t.Errorf("dnsserver = %v, want %v", current.DNSServer, prior.DNSServers)
	}
	if got := current.ARPTableStaticEntry.ValueBool(); !got {
		t.Errorf("arp_table_static_entry = %v, want true", got)
	}
	if !current.DefaultLeaseTime.IsNull() || !current.MaxLeaseTime.IsNull() || !current.WINSServer.IsNull() || !current.NTPServer.IsNull() {
		t.Errorf("version-1-only attributes must be null, got defaultleasetime=%v maxleasetime=%v winsserver=%v ntpserver=%v",
			current.DefaultLeaseTime, current.MaxLeaseTime, current.WINSServer, current.NTPServer)
	}
}

// listStringValues returns the string contents of a list of strings.
func listStringValues(t *testing.T, l types.List) []string {
	t.Helper()
	elems := l.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		s, ok := e.(types.String)
		if !ok {
			t.Fatalf("list element = %T, want types.String", e)
		}
		out = append(out, s.ValueString())
	}
	return out
}

// equalStrings compares two string slices.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
