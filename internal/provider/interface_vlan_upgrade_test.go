package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestInterfaceVLANModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	prior := interfaceVLANModelV0{
		If:          types.StringValue("vmx0"),
		Tag:         types.Int64Value(10),
		PCP:         types.Int64Value(3),
		Description: types.StringValue("guest VLAN"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	if got, want := cur.If.ValueString(), "vmx0"; got != want {
		t.Errorf("If = %q, want %q", got, want)
	}
	if got, want := cur.Tag.ValueInt64(), int64(10); got != want {
		t.Errorf("Tag = %d, want %d", got, want)
	}
	if got, want := cur.PCP.ValueInt64(), int64(3); got != want {
		t.Errorf("PCP = %d, want %d", got, want)
	}
	// description -> descr
	if got, want := cur.Descr.ValueString(), "guest VLAN"; got != want {
		t.Errorf("Descr = %q, want %q", got, want)
	}
	// id and vlanif are set by the StateUpgrader, not by toCurrent.
	if !cur.ID.IsNull() || !cur.VLANIf.IsNull() {
		t.Errorf("ID/VLANIf must be left to the upgrader, got %v/%v", cur.ID, cur.VLANIf)
	}

	// The SDKv2 empty-string zero value for the unset optional description
	// must become null, not "".
	prior.Description = types.StringValue("")
	cur, diags = prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !cur.Descr.IsNull() {
		t.Errorf("Descr = %v, want null for the SDKv2 empty-string zero value", cur.Descr)
	}
}

func TestInterfaceVLANUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	// The current schema must declare version 1 so Terraform knows the
	// 0 -> 1 upgrader applies to existing pfsense_interface_vlan state.
	var schemaResp resource.SchemaResponse
	(&interfaceVLANResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("schema Version = %d, want 1", schemaResp.Schema.Version)
	}

	prior := interfaceVLANModelV0{
		If:          types.StringValue("vmx0"),
		Tag:         types.Int64Value(10),
		PCP:         types.Int64Value(3),
		Description: types.StringValue("guest VLAN"),
	}

	// Replicate what the framework does before invoking the upgrader:
	// decode the prior raw state against the PriorSchema into req.State.
	var priorState tfsdk.State
	priorState.Schema = interfaceVLANPriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The old SDKv2 state always carries the implicit id attribute; for
	// pfsense_interface_vlan that was the generated VLAN interface name.
	rawJSON, err := json.Marshal(map[string]any{"id": "vmx0.10"})
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

	(&interfaceVLANResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got interfaceVLANModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}

	// The v1 id is the natural key, not the old vlanif-based id.
	if want := "vmx0|10"; got.ID.ValueString() != want {
		t.Errorf("ID = %q, want %q", got.ID.ValueString(), want)
	}
	// The old id becomes the new computed vlanif attribute.
	if want := "vmx0.10"; got.VLANIf.ValueString() != want {
		t.Errorf("VLANIf = %q, want the prior id %q", got.VLANIf.ValueString(), want)
	}
	if got.If.ValueString() != "vmx0" {
		t.Errorf("If = %q, want vmx0", got.If.ValueString())
	}
	if got.Tag.ValueInt64() != 10 {
		t.Errorf("Tag = %d, want 10", got.Tag.ValueInt64())
	}
	if got.PCP.ValueInt64() != 3 {
		t.Errorf("PCP = %d, want 3", got.PCP.ValueInt64())
	}
	if got.Descr.ValueString() != "guest VLAN" {
		t.Errorf("Descr = %q, want %q", got.Descr.ValueString(), "guest VLAN")
	}
}

// TestInterfaceVLANUpgradeStateV0To1NoRawID covers the case where the prior
// raw state carries no implicit id: vlanif stays null and the first Read
// repopulates it from the API.
func TestInterfaceVLANUpgradeStateV0To1NoRawID(t *testing.T) {
	ctx := context.Background()

	var schemaResp resource.SchemaResponse
	(&interfaceVLANResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}

	prior := interfaceVLANModelV0{
		If:  types.StringValue("vmx1"),
		Tag: types.Int64Value(4094),
	}

	var priorState tfsdk.State
	priorState.Schema = interfaceVLANPriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	req := resource.UpgradeStateRequest{State: &priorState}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	(&interfaceVLANResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got interfaceVLANModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}
	if want := "vmx1|4094"; got.ID.ValueString() != want {
		t.Errorf("ID = %q, want %q", got.ID.ValueString(), want)
	}
	if !got.VLANIf.IsNull() {
		t.Errorf("VLANIf = %v, want null when the prior state has no id", got.VLANIf)
	}
}

func TestInterfaceVLANUpgradeStateMap(t *testing.T) {
	r := &interfaceVLANResource{}
	upgraders := r.UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatalf("no state upgrader registered for version 0")
	}
	if upgrader.PriorSchema == nil {
		t.Fatalf("PriorSchema is nil")
	}
	for _, name := range []string{"if", "tag", "pcp", "description"} {
		if upgrader.PriorSchema.Attributes[name] == nil {
			t.Errorf("PriorSchema missing v0 attribute %s", name)
		}
	}
	// The v1-only attributes must NOT be part of the prior schema.
	for _, name := range []string{"id", "descr", "vlanif"} {
		if upgrader.PriorSchema.Attributes[name] != nil {
			t.Errorf("PriorSchema must not contain the v1-only attribute %s", name)
		}
	}
}
