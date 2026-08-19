package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_interface_vlan (v0, SDKv2 provider v1) -> pfsense_interface_vlan (v1)
// ---------------------------------------------------------------------------

// interfaceVLANModelV0 is the schema-version-0 (SDKv2-era) state model for
// pfsense_interface_vlan. The tfsdk tags use the OLD attribute names so that
// req.State.Get decodes the prior state verbatim. The implicit SDKv2 `id`
// attribute is intentionally absent: in v0 it was the generated VLAN
// interface name (`vlanif`), which the v1 resource models as its own computed
// attribute, so it is read from req.RawState instead (see upgradeStateV0To1).
type interfaceVLANModelV0 struct {
	If          types.String `tfsdk:"if"`
	Tag         types.Int64  `tfsdk:"tag"`
	PCP         types.Int64  `tfsdk:"pcp"`
	Description types.String `tfsdk:"description"`
}

var _ resource.ResourceWithUpgradeState = (*interfaceVLANResource)(nil)

// interfaceVLANPriorSchemaV0 is the SDKv2 pfsense_interface_vlan schema (from
// the provider v1 properties map) translated to framework attributes. It
// mirrors the v0 state exactly: same attribute names, same required/optional
// flags, SDKv2 types mapped per TYPE TRANSLATION rules. The implicit SDKv2
// `id` attribute is deliberately excluded (it is read from RawState).
var interfaceVLANPriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"if":          schema.StringAttribute{Required: true},
		"tag":         schema.Int64Attribute{Required: true},
		"pcp":         schema.Int64Attribute{Optional: true},
		"description": schema.StringAttribute{Optional: true},
	},
}

// UpgradeState implements resource.ResourceWithUpgradeState so that existing
// pfsense_interface_vlan state (schema version 0) is migrated in-place to
// schema version 1 with no recreation.
func (r *interfaceVLANResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &interfaceVLANPriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 decodes the v0 state, maps every user-configurable value
// to its new home, derives the resource id, and writes the v1 state.
//
// The v0 id was the generated VLAN interface name returned by the API
// (`vlanif`, e.g. "vlan0.10"); the v1 id is the natural key "<if>|<tag>" and
// `vlanif` became a computed attribute, so the old id is moved there.
func (r *interfaceVLANResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior interfaceVLANModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cur, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Both natural-key attributes are Required in v0, so they are always
	// populated; the v1 Read looks the VLAN up by `if` + `tag`.
	iface := prior.If.ValueString()
	if iface == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_interface_vlan",
			"unable to derive the resource id from the prior state: \"if\" is empty",
		)
		return
	}
	cur.ID = types.StringValue(r.key(iface, prior.Tag.ValueInt64()))

	// Carry the old id over as `vlanif`. When the prior state has no implicit
	// id the attribute stays null and the first Read repopulates it.
	cur.VLANIf = emptyToNull(types.StringValue(priorResourceID(req.RawState)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
}

// toCurrent maps every v0 value to its v1 home:
//
//	if          -> if
//	tag         -> tag
//	pcp         -> pcp
//	description -> descr
//
// `pcp` is carried over verbatim. The SDKv2 persisted an unset optional int as
// 0, which is also the pfSense default priority code point, so the value the
// v1 Read gets back from the API matches either way. The optional
// `description` goes through emptyToNull so the SDKv2 "" zero value does not
// land in v1 state as an empty string where the framework means null. The
// computed "id" and "vlanif" are set by the StateUpgrader, not here.
func (m interfaceVLANModelV0) toCurrent(ctx context.Context) (interfaceVLANModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	return interfaceVLANModel{
		If:    m.If,
		Tag:   m.Tag,
		PCP:   m.PCP,
		Descr: emptyToNull(m.Description),
	}, diags
}
