package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRuleModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	prior := firewallRuleModelV0{
		AckQueue:        types.StringValue("q_ack"),
		DefaultQueue:    types.StringValue("q_default"),
		Description:     types.StringValue("Allow web"),
		Direction:       types.StringValue("in"),
		Disabled:        types.BoolValue(false),
		DnPipe:          types.StringValue("dnpipe1"),
		Destination:     types.StringValue("10.0.0.0/24"),
		DestinationPort: types.StringValue("443"),
		Floating:        types.BoolValue(false),
		Gateway:         types.StringValue("WANGW"),
		ICMPType:        types.ListValueMust(types.StringType, []attr.Value{types.StringValue("echoreq")}),
		Interface:       types.ListValueMust(types.StringType, []attr.Value{types.StringValue("wan"), types.StringValue("lan")}),
		IPProtocol:      types.StringValue("inet"),
		Log:             types.BoolValue(true),
		PdPipe:          types.StringValue("pdnpipe1"),
		Protocol:        types.StringValue("tcp"),
		Quick:           types.BoolValue(true),
		Schedule:        types.StringValue("workhours"),
		Source:          types.StringValue("any"),
		SourcePort:      types.StringValue("any"),
		StateType:       types.StringValue("keep state"),
		TCPFlag: []firewallRuleTCPFlagV0{
			{Flag: types.StringValue("syn"), Present: types.BoolValue(true)},
			{Flag: types.StringValue("ack"), Present: types.BoolValue(false)},
		},
		Type: types.StringValue("pass"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	// Identity pass-through.
	if cur.Type.ValueString() != "pass" {
		t.Errorf("Type = %q, want pass", cur.Type.ValueString())
	}
	if cur.Direction.ValueString() != "in" {
		t.Errorf("Direction = %q, want in", cur.Direction.ValueString())
	}
	// disabled = false is the SDKv2 zero value for an unset optional bool, so
	// it is normalised to null rather than carried over as false.
	if !cur.Disabled.IsNull() {
		t.Errorf("Disabled = %v, want null (from disabled = false)", cur.Disabled)
	}
	if cur.Destination.ValueString() != "10.0.0.0/24" {
		t.Errorf("Destination = %q, want 10.0.0.0/24", cur.Destination.ValueString())
	}
	if cur.DestinationPort.ValueString() != "443" {
		t.Errorf("DestinationPort = %q, want 443", cur.DestinationPort.ValueString())
	}
	if cur.Gateway.ValueString() != "WANGW" {
		t.Errorf("Gateway = %q, want WANGW", cur.Gateway.ValueString())
	}
	if cur.IPProtocol.ValueString() != "inet" {
		t.Errorf("IPProtocol = %q, want inet", cur.IPProtocol.ValueString())
	}
	if cur.Log.ValueBool() != true {
		t.Errorf("Log = %v, want true", cur.Log.ValueBool())
	}
	if cur.Protocol.ValueString() != "tcp" {
		t.Errorf("Protocol = %q, want tcp", cur.Protocol.ValueString())
	}
	if cur.Quick.ValueBool() != true {
		t.Errorf("Quick = %v, want true", cur.Quick.ValueBool())
	}
	if cur.Source.ValueString() != "any" {
		t.Errorf("Source = %q, want any", cur.Source.ValueString())
	}
	if cur.SourcePort.ValueString() != "any" {
		t.Errorf("SourcePort = %q, want any", cur.SourcePort.ValueString())
	}
	if cur.StateType.ValueString() != "keep state" {
		t.Errorf("StateType = %q, want keep state", cur.StateType.ValueString())
	}

	// Renamed attributes.
	if cur.AckQueue.ValueString() != "q_ack" {
		t.Errorf("AckQueue = %q, want q_ack (from ack_queue)", cur.AckQueue.ValueString())
	}
	if cur.DefaultQueue.ValueString() != "q_default" {
		t.Errorf("DefaultQueue = %q, want q_default (from default_queue)", cur.DefaultQueue.ValueString())
	}
	if cur.Descr.ValueString() != "Allow web" {
		t.Errorf("Descr = %q, want Allow web (from description)", cur.Descr.ValueString())
	}
	if cur.DnPipe.ValueString() != "dnpipe1" {
		t.Errorf("DnPipe = %q, want dnpipe1 (from dn_pipe)", cur.DnPipe.ValueString())
	}
	if cur.PdPipe.ValueString() != "pdnpipe1" {
		t.Errorf("PdPipe = %q, want pdnpipe1 (from pdn_pipe)", cur.PdPipe.ValueString())
	}
	if cur.Sched.ValueString() != "workhours" {
		t.Errorf("Sched = %q, want workhours (from schedule)", cur.Sched.ValueString())
	}

	// ID rule: id == descr == old description (old tracker NOT carried).
	if cur.ID.ValueString() != "Allow web" {
		t.Errorf("ID = %q, want Allow web (new id is the description)", cur.ID.ValueString())
	}

	// Renamed lists.
	var icmpTypes []string
	if diags := cur.ICMPType.ElementsAs(ctx, &icmpTypes, false); diags.HasError() {
		t.Fatalf("decoding icmptype list: %s", diags)
	}
	if len(icmpTypes) != 1 || icmpTypes[0] != "echoreq" {
		t.Errorf("ICMPType = %v, want [echoreq] (from icmp_type)", icmpTypes)
	}

	var interfaces []string
	if diags := cur.Interface.ElementsAs(ctx, &interfaces, false); diags.HasError() {
		t.Fatalf("decoding interface list: %s", diags)
	}
	wantInterfaces := []string{"wan", "lan"}
	if len(interfaces) != len(wantInterfaces) {
		t.Fatalf("Interface = %v, want %v", interfaces, wantInterfaces)
	}
	for i := range wantInterfaces {
		if interfaces[i] != wantInterfaces[i] {
			t.Errorf("Interface[%d] = %q, want %q", i, interfaces[i], wantInterfaces[i])
		}
	}

	// tcp_flag split: set = present==true, out_of = all flags in order,
	// any = false (non-empty covered list).
	var setFlags []string
	if diags := cur.TCPFlagsSet.ElementsAs(ctx, &setFlags, false); diags.HasError() {
		t.Fatalf("decoding tcp_flags_set list: %s", diags)
	}
	wantSet := []string{"syn"}
	if len(setFlags) != len(wantSet) {
		t.Fatalf("TCPFlagsSet = %v, want %v", setFlags, wantSet)
	}
	for i := range wantSet {
		if setFlags[i] != wantSet[i] {
			t.Errorf("TCPFlagsSet[%d] = %q, want %q", i, setFlags[i], wantSet[i])
		}
	}

	var outOfFlags []string
	if diags := cur.TCPFlagsOutOf.ElementsAs(ctx, &outOfFlags, false); diags.HasError() {
		t.Fatalf("decoding tcp_flags_out_of list: %s", diags)
	}
	wantOutOf := []string{"syn", "ack"}
	if len(outOfFlags) != len(wantOutOf) {
		t.Fatalf("TCPFlagsOutOf = %v, want %v", outOfFlags, wantOutOf)
	}
	for i := range wantOutOf {
		if outOfFlags[i] != wantOutOf[i] {
			t.Errorf("TCPFlagsOutOf[%d] = %q, want %q", i, outOfFlags[i], wantOutOf[i])
		}
	}
	if cur.TCPFlagsAny.ValueBool() != false {
		t.Errorf("TCPFlagsAny = %v, want false", cur.TCPFlagsAny.ValueBool())
	}

	// v1-only attributes stay null so Read repopulates them.
	if !cur.Dscp.IsNull() {
		t.Errorf("Dscp = %v, want null", cur.Dscp)
	}
	if !cur.Tag.IsNull() {
		t.Errorf("Tag = %v, want null", cur.Tag)
	}
}

func TestRuleModelV0ToCurrentNoTCPFlags(t *testing.T) {
	ctx := context.Background()

	// A null (unset) prior tcp_flag list maps to any=true (the old provider
	// set TCPFlagsAny when the covered-flags list was empty), with null
	// out_of/set lists rather than bare zero-value types.List.
	prior := firewallRuleModelV0{
		Type:        types.StringValue("pass"),
		Interface:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("wan")}),
		Description: types.StringValue("no flags"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if cur.TCPFlagsAny.ValueBool() != true {
		t.Errorf("TCPFlagsAny = %v, want true for empty covered flags", cur.TCPFlagsAny.ValueBool())
	}
	if !cur.TCPFlagsOutOf.IsNull() {
		t.Errorf("TCPFlagsOutOf = %v, want null", cur.TCPFlagsOutOf)
	}
	if !cur.TCPFlagsSet.IsNull() {
		t.Errorf("TCPFlagsSet = %v, want null", cur.TCPFlagsSet)
	}

	// Empty-but-present list: any=true too, with explicit empty lists.
	prior.TCPFlag = []firewallRuleTCPFlagV0{}
	cur, diags = prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if cur.TCPFlagsAny.ValueBool() != true {
		t.Errorf("TCPFlagsAny = %v, want true for empty covered flags", cur.TCPFlagsAny.ValueBool())
	}
	if cur.TCPFlagsOutOf.IsNull() || len(cur.TCPFlagsOutOf.Elements()) != 0 {
		t.Errorf("TCPFlagsOutOf = %v, want empty list", cur.TCPFlagsOutOf)
	}
	if cur.TCPFlagsSet.IsNull() || len(cur.TCPFlagsSet.Elements()) != 0 {
		t.Errorf("TCPFlagsSet = %v, want empty list", cur.TCPFlagsSet)
	}
}

func TestRuleModelV0ToCurrentZeroValueNormalisation(t *testing.T) {
	ctx := context.Background()

	// SDKv2 persists unset optional bools as false; carrying that over would
	// give every rule that never set disabled/log/quick a false→null diff.
	cur, diags := (firewallRuleModelV0{
		Description: types.StringValue("rule"),
		Type:        types.StringValue("pass"),
		IPProtocol:  types.StringValue("inet"),
		Disabled:    types.BoolValue(false),
		Log:         types.BoolValue(false),
		Quick:       types.BoolValue(false),
	}).toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	for name, v := range map[string]types.Bool{
		"Disabled": cur.Disabled,
		"Log":      cur.Log,
		"Quick":    cur.Quick,
	} {
		if !v.IsNull() {
			t.Errorf("%s = %v, want null for the SDKv2 false zero value", name, v)
		}
	}
}

func TestRuleModelV0ToCurrentFloatingWarning(t *testing.T) {
	ctx := context.Background()

	// floating=true produces a warning; the attribute itself is dropped.
	prior := firewallRuleModelV0{
		Type:        types.StringValue("pass"),
		Interface:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("wan")}),
		Description: types.StringValue("allow web"),
		Floating:    types.BoolValue(true),
	}

	_, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %s", diags)
	}
	if len(diags) != 1 {
		t.Fatalf("diags = %d entries, want 1 warning (floating)", len(diags))
	}
	if got := diags[0].Severity().String(); got != "Warning" {
		t.Errorf("diag severity = %s, want Warning", got)
	}

	// floating=false/absent must be dropped silently.
	prior.Floating = types.BoolValue(false)
	_, diags = prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %s", diags)
	}
	if len(diags) != 0 {
		t.Fatalf("diags = %d entries, want none for a non-floating rule: %s", len(diags), diags)
	}
}

// TestRuleModelV0ToCurrentIPProtocolWarning covers the v0-only "inet46" value:
// the v1 ipprotocol validator only accepts inet/inet6, so the upgrade must
// warn instead of letting the next plan fail with an opaque validation error.
func TestRuleModelV0ToCurrentIPProtocolWarning(t *testing.T) {
	ctx := context.Background()

	prior := firewallRuleModelV0{
		Type:        types.StringValue("pass"),
		Interface:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("wan")}),
		Description: types.StringValue("allow web"),
		IPProtocol:  types.StringValue("inet46"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected error diagnostics: %s", diags)
	}
	if len(diags) != 1 {
		t.Fatalf("diags = %d entries, want 1 warning (ip_protocol): %s", len(diags), diags)
	}
	if got := diags[0].Severity().String(); got != "Warning" {
		t.Errorf("diag severity = %s, want Warning", got)
	}
	// The value is still carried over so the practitioner can see what to fix.
	if got := cur.IPProtocol.ValueString(); got != "inet46" {
		t.Errorf("IPProtocol = %q, want inet46 carried over", got)
	}

	for _, valid := range []string{"inet", "inet6"} {
		prior.IPProtocol = types.StringValue(valid)
		if _, diags := prior.toCurrent(ctx); len(diags) != 0 {
			t.Errorf("ip_protocol = %q produced %d diagnostics, want none: %s", valid, len(diags), diags)
		}
	}

	// An unset (SDKv2 "") ip_protocol becomes null and must not warn.
	prior.IPProtocol = types.StringValue("")
	cur, diags = prior.toCurrent(ctx)
	if len(diags) != 0 {
		t.Errorf("empty ip_protocol produced %d diagnostics, want none: %s", len(diags), diags)
	}
	if !cur.IPProtocol.IsNull() {
		t.Errorf("IPProtocol = %v, want null for the SDKv2 empty-string zero value", cur.IPProtocol)
	}
}

// TestRuleUpgradeStateV0To1EmptyDescrAborts covers the blocking case: the v1
// provider looks rules up by `descr`, so an empty description would match the
// first unrelated descr-less rule and PATCH/DELETE it. The upgrade must fail
// rather than warn, since a warning does not stop the apply.
func TestRuleUpgradeStateV0To1EmptyDescrAborts(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	(&firewallRuleResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}

	prior := firewallRuleModelV0{
		Type:        types.StringValue("pass"),
		Interface:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("wan")}),
		Description: types.StringValue(""),
		ICMPType:    types.ListNull(types.StringType),
	}

	var priorState tfsdk.State
	priorState.Schema = firewallRulePriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	req := resource.UpgradeStateRequest{State: &priorState}
	resp := resource.UpgradeStateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	(&firewallRuleResource{}).upgradeStateV0To1(ctx, req, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatalf("expected an error diagnostic for an empty description, got: %s", resp.Diagnostics)
	}
	// The upgrader must not write any state when it aborts.
	if !resp.State.Raw.IsNull() {
		t.Errorf("state was written despite the error: %v", resp.State.Raw)
	}
}

func TestRuleUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	// The current schema must declare version 1 so Terraform knows the
	// 0 -> 1 upgrader applies to existing pfsense_firewall_rule state.
	var schemaResp resource.SchemaResponse
	(&firewallRuleResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("schema Version = %d, want 1", schemaResp.Schema.Version)
	}

	prior := firewallRuleModelV0{
		Type:        types.StringValue("pass"),
		Interface:   types.ListValueMust(types.StringType, []attr.Value{types.StringValue("wan")}),
		ICMPType:    types.ListNull(types.StringType),
		Description: types.StringValue("Allow ssh"),
		Protocol:    types.StringValue("tcp"),
		Destination: types.StringValue("any"),
		TCPFlag: []firewallRuleTCPFlagV0{
			{Flag: types.StringValue("syn"), Present: types.BoolValue(true)},
			{Flag: types.StringValue("ack"), Present: types.BoolValue(false)},
		},
	}

	// Replicate what the framework does before invoking the upgrader:
	// decode the prior raw state against the PriorSchema into req.State.
	var priorState tfsdk.State
	priorState.Schema = firewallRulePriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	req := resource.UpgradeStateRequest{State: &priorState}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	(&firewallRuleResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got firewallRuleModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}

	// ID rule: the new id is the description (natural key), NOT the old
	// numeric tracker (which is not present in this state at all).
	if got.ID.ValueString() != "Allow ssh" {
		t.Errorf("ID = %q, want Allow ssh", got.ID.ValueString())
	}
	if got.Descr.ValueString() != "Allow ssh" {
		t.Errorf("Descr = %q, want Allow ssh", got.Descr.ValueString())
	}
	if got.Type.ValueString() != "pass" {
		t.Errorf("Type = %q, want pass", got.Type.ValueString())
	}
	if got.Protocol.ValueString() != "tcp" {
		t.Errorf("Protocol = %q, want tcp", got.Protocol.ValueString())
	}

	var setFlags []string
	if diags := got.TCPFlagsSet.ElementsAs(ctx, &setFlags, false); diags.HasError() {
		t.Fatalf("decoding upgraded tcp_flags_set: %s", diags)
	}
	if len(setFlags) != 1 || setFlags[0] != "syn" {
		t.Errorf("TCPFlagsSet = %v, want [syn]", setFlags)
	}
	var outOfFlags []string
	if diags := got.TCPFlagsOutOf.ElementsAs(ctx, &outOfFlags, false); diags.HasError() {
		t.Fatalf("decoding upgraded tcp_flags_out_of: %s", diags)
	}
	if len(outOfFlags) != 2 || outOfFlags[0] != "syn" || outOfFlags[1] != "ack" {
		t.Errorf("TCPFlagsOutOf = %v, want [syn ack]", outOfFlags)
	}
	if got.TCPFlagsAny.ValueBool() != false {
		t.Errorf("TCPFlagsAny = %v, want false", got.TCPFlagsAny.ValueBool())
	}
}

func TestRuleUpgradeStateMap(t *testing.T) {
	r := &firewallRuleResource{}
	upgraders := r.UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatalf("no state upgrader registered for version 0")
	}
	if upgrader.PriorSchema == nil {
		t.Fatalf("PriorSchema is nil")
	}
	// Every v0 attribute must be present in the prior schema.
	for _, name := range []string{
		"ack_queue", "default_queue", "description", "direction", "disabled",
		"dn_pipe", "destination", "destination_port", "floating", "gateway",
		"icmp_type", "interface", "ip_protocol", "log", "pdn_pipe", "protocol",
		"quick", "schedule", "source", "source_port", "state_type", "tcp_flag",
		"type",
	} {
		if upgrader.PriorSchema.Attributes[name] == nil {
			t.Errorf("PriorSchema missing v0 attribute %s", name)
		}
	}
	// The implicit SDKv2 id (numeric tracker) must NOT be part of the
	// prior schema: the new id is the rule description.
	if upgrader.PriorSchema.Attributes["id"] != nil {
		t.Fatalf("PriorSchema must not contain the implicit id attribute")
	}
}
