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

func TestAliasModelV0ToCurrent(t *testing.T) {
	ctx := context.Background()

	prior := firewallAliasModelV0{
		Name:        types.StringValue("test_alias"),
		Type:        types.StringValue("host"),
		Description: types.StringValue("web servers"),
		Target: []firewallAliasTargetV0{
			{Address: types.StringValue("10.0.0.1"), Description: types.StringValue("web1")},
			// Missing/empty description must survive as an empty string at
			// the same index (old provider skips setting empty descriptions).
			{Address: types.StringValue("10.0.0.2"), Description: types.StringNull()},
		},
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}

	// Identity pass-through.
	if cur.Name.ValueString() != "test_alias" {
		t.Errorf("Name = %q, want test_alias", cur.Name.ValueString())
	}
	if cur.Type.ValueString() != "host" {
		t.Errorf("Type = %q, want host", cur.Type.ValueString())
	}
	// Renamed attribute.
	if cur.Descr.ValueString() != "web servers" {
		t.Errorf("Descr = %q, want web servers (from description)", cur.Descr.ValueString())
	}
	// The computed id is set by the StateUpgrader, not toCurrent.
	if !cur.ID.IsNull() {
		t.Errorf("ID = %v, want null", cur.ID)
	}

	// Flattened lists, preserved 1:1 by index.
	var addresses []string
	if diags := cur.Address.ElementsAs(ctx, &addresses, false); diags.HasError() {
		t.Fatalf("decoding address list: %s", diags)
	}
	wantAddresses := []string{"10.0.0.1", "10.0.0.2"}
	if len(addresses) != len(wantAddresses) {
		t.Fatalf("Address = %v, want %v", addresses, wantAddresses)
	}
	for i := range wantAddresses {
		if addresses[i] != wantAddresses[i] {
			t.Errorf("Address[%d] = %q, want %q", i, addresses[i], wantAddresses[i])
		}
	}

	var details []string
	if diags := cur.Detail.ElementsAs(ctx, &details, false); diags.HasError() {
		t.Fatalf("decoding detail list: %s", diags)
	}
	wantDetails := []string{"web1", ""}
	if len(details) != len(wantDetails) {
		t.Fatalf("Detail = %v, want %v", details, wantDetails)
	}
	for i := range wantDetails {
		if details[i] != wantDetails[i] {
			t.Errorf("Detail[%d] = %q, want %q", i, details[i], wantDetails[i])
		}
	}
}

func TestAliasModelV0ToCurrentNullTarget(t *testing.T) {
	ctx := context.Background()

	// A null (unset) prior target list must stay null, not become a
	// bare zero-value types.List (which is not a valid state value).
	prior := firewallAliasModelV0{
		Name: types.StringValue("empty_alias"),
		Type: types.StringValue("port"),
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !cur.Address.IsNull() {
		t.Errorf("Address = %v, want null", cur.Address)
	}
	if !cur.Detail.IsNull() {
		t.Errorf("Detail = %v, want null", cur.Detail)
	}
}

func TestAliasModelV0ToCurrentAllEmptyDetails(t *testing.T) {
	ctx := context.Background()

	// Targets whose descriptions are all empty/null migrate to a null detail
	// list (not [""]), so an omitted `detail` config does not plan a spurious
	// [""] -> null diff. Index alignment with address only carries meaning
	// once at least one description is non-empty.
	prior := firewallAliasModelV0{
		Name: types.StringValue("no_desc"),
		Type: types.StringValue("host"),
		Target: []firewallAliasTargetV0{
			{Address: types.StringValue("10.0.0.1"), Description: types.StringNull()},
			{Address: types.StringValue("10.0.0.2"), Description: types.StringValue("")},
		},
	}

	cur, diags := prior.toCurrent(ctx)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %s", diags)
	}
	if !cur.Detail.IsNull() {
		t.Errorf("Detail = %v, want null (all descriptions empty)", cur.Detail)
	}
}

func TestAliasUpgradeStateV0To1(t *testing.T) {
	ctx := context.Background()

	// The current schema must declare version 1 so Terraform knows the
	// 0 -> 1 upgrader applies to existing pfsense_firewall_alias state.
	var schemaResp resource.SchemaResponse
	(&firewallAliasResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema diagnostics: %s", schemaResp.Diagnostics)
	}
	if schemaResp.Schema.Version != 1 {
		t.Fatalf("schema Version = %d, want 1", schemaResp.Schema.Version)
	}

	prior := firewallAliasModelV0{
		Name:        types.StringValue("test_alias"),
		Type:        types.StringValue("host"),
		Description: types.StringValue("web servers"),
		Target: []firewallAliasTargetV0{
			{Address: types.StringValue("10.0.0.1"), Description: types.StringValue("web1")},
			{Address: types.StringValue("10.0.0.2"), Description: types.StringValue("web2")},
		},
	}

	// Replicate what the framework does before invoking the upgrader:
	// decode the prior raw state against the PriorSchema into req.State.
	var priorState tfsdk.State
	priorState.Schema = firewallAliasPriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// The old SDKv2 state always carries the implicit id attribute. Give it a
	// value that contradicts the natural key so the assertion proves the v1 id
	// is derived from `name`, not read from the raw id.
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

	(&firewallAliasResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got firewallAliasModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}

	// The v1 id is the natural key (`name`), never the raw id (which the
	// fixture set to "stale_id" to prove the upgrader ignores it).
	if got.ID.ValueString() != "test_alias" {
		t.Errorf("ID = %q, want natural key %q", got.ID.ValueString(), "test_alias")
	}
	if got.Name.ValueString() != "test_alias" {
		t.Errorf("Name = %q, want test_alias", got.Name.ValueString())
	}
	if got.Type.ValueString() != "host" {
		t.Errorf("Type = %q, want host", got.Type.ValueString())
	}
	if got.Descr.ValueString() != "web servers" {
		t.Errorf("Descr = %q, want web servers", got.Descr.ValueString())
	}

	var addresses []string
	if diags := got.Address.ElementsAs(ctx, &addresses, false); diags.HasError() {
		t.Fatalf("decoding upgraded address list: %s", diags)
	}
	wantAddresses := []string{"10.0.0.1", "10.0.0.2"}
	if len(addresses) != len(wantAddresses) {
		t.Fatalf("Address = %v, want %v", addresses, wantAddresses)
	}
	for i := range wantAddresses {
		if addresses[i] != wantAddresses[i] {
			t.Errorf("Address[%d] = %q, want %q", i, addresses[i], wantAddresses[i])
		}
	}

	var details []string
	if diags := got.Detail.ElementsAs(ctx, &details, false); diags.HasError() {
		t.Fatalf("decoding upgraded detail list: %s", diags)
	}
	wantDetails := []string{"web1", "web2"}
	if len(details) != len(wantDetails) {
		t.Fatalf("Detail = %v, want %v", details, wantDetails)
	}
	for i := range wantDetails {
		if details[i] != wantDetails[i] {
			t.Errorf("Detail[%d] = %q, want %q", i, details[i], wantDetails[i])
		}
	}
}

func TestAliasUpgradeStateV0To1FallbackID(t *testing.T) {
	ctx := context.Background()

	prior := firewallAliasModelV0{
		Name:        types.StringValue("fallback_alias"),
		Type:        types.StringValue("port"),
		Description: types.StringValue("ports"),
		Target: []firewallAliasTargetV0{
			{Address: types.StringValue("443")},
		},
	}

	var priorState tfsdk.State
	priorState.Schema = firewallAliasPriorSchemaV0
	if diags := priorState.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("setting prior state: %s", diags)
	}

	// No RawState: the natural key (`name`) must become the new id.
	req := resource.UpgradeStateRequest{State: &priorState}

	var schemaResp resource.SchemaResponse
	(&firewallAliasResource{}).Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	(&firewallAliasResource{}).upgradeStateV0To1(ctx, req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("upgrade diagnostics: %s", resp.Diagnostics)
	}

	var got firewallAliasModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading upgraded state: %s", diags)
	}
	if got.ID.ValueString() != "fallback_alias" {
		t.Errorf("ID = %q, want fallback to natural key %q", got.ID.ValueString(), "fallback_alias")
	}
}

func TestAliasUpgradeStateMap(t *testing.T) {
	r := &firewallAliasResource{}
	upgraders := r.UpgradeState(context.Background())

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatalf("no state upgrader registered for version 0")
	}
	if upgrader.PriorSchema == nil {
		t.Fatalf("PriorSchema is nil")
	}
	if upgrader.PriorSchema.Attributes["name"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute name")
	}
	if upgrader.PriorSchema.Attributes["type"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute type")
	}
	if upgrader.PriorSchema.Attributes["description"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute description")
	}
	if upgrader.PriorSchema.Attributes["target"] == nil {
		t.Fatalf("PriorSchema missing v0 attribute target")
	}
	// The implicit SDKv2 id must NOT be part of the prior schema.
	if upgrader.PriorSchema.Attributes["id"] != nil {
		t.Fatalf("PriorSchema must not contain the implicit id attribute")
	}
}
