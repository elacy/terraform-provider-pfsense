package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// TestConstantComputedUsesStateForUnknown verifies the plan-modifier semantics
// that distinguish constantComputed* from a plain computed attribute. The API
// assigns these values once and never changes them.
//
// UseStateForUnknown is only correct for the VPN system-assigned IDs
// (ikeid/reqid/vpnid/vpnif/uniqid on phase 1/2 and OpenVPN), whose Update
// methods do NOT repopulate them from the API response: after an in-place
// update the framework would surface them as "known after apply" every plan,
// so the prior state value must carry forward. The other system-assigned IDs
// (uid/refid/uniqid on the virtual IP, created_time/vlanif) ARE repopulated by
// Update, so they must stay plain computed — UseStateForUnknown would pin the
// stale planned value and turn any API-returned discrepancy into a hard
// "inconsistent result after apply" instead of a silently-accepted refresh.
func TestConstantComputedUsesStateForUnknown(t *testing.T) {
	// A non-null prior state marks the resource as already-created (update),
	// which is the only path where UseStateForUnknown acts.
	priorState := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "present")}

	t.Run("string", func(t *testing.T) {
		attr := constantComputedStringAttribute("desc")
		if len(attr.PlanModifiers) != 1 {
			t.Fatalf("constantComputedStringAttribute has %d plan modifiers, want 1", len(attr.PlanModifiers))
		}
		req := planmodifier.StringRequest{
			Path:        path.Root("ikeid"),
			State:       priorState,
			ConfigValue: types.StringNull(),     // computed: never set in config
			StateValue:  types.StringValue("1"), // prior value from state
			PlanValue:   types.StringUnknown(),  // unknown after an in-place Update
		}
		resp := &planmodifier.StringResponse{PlanValue: types.StringUnknown()}
		attr.PlanModifiers[0].PlanModifyString(context.Background(), req, resp)
		if got := resp.PlanValue.ValueString(); got != "1" {
			t.Fatalf("UseStateForUnknown did not carry prior value forward: got %q, want %q", got, "1")
		}
	})

	t.Run("int64", func(t *testing.T) {
		attr := constantComputedIntAttribute("desc")
		if len(attr.PlanModifiers) != 1 {
			t.Fatalf("constantComputedIntAttribute has %d plan modifiers, want 1", len(attr.PlanModifiers))
		}
		req := planmodifier.Int64Request{
			Path:        path.Root("vpnid"),
			State:       priorState,
			ConfigValue: types.Int64Null(),
			StateValue:  types.Int64Value(7),
			PlanValue:   types.Int64Unknown(),
		}
		resp := &planmodifier.Int64Response{PlanValue: types.Int64Unknown()}
		attr.PlanModifiers[0].PlanModifyInt64(context.Background(), req, resp)
		if got := resp.PlanValue.ValueInt64(); got != 7 {
			t.Fatalf("UseStateForUnknown did not carry prior value forward: got %d, want %d", got, 7)
		}
	})
}
