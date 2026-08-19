package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_firewall_alias (v0, SDKv2 provider v1) -> pfsense_firewall_alias (v1)
// ---------------------------------------------------------------------------

// firewallAliasTargetV0 is the version-0 state shape of one element of the old
// "target" list (SDKv2 TypeList of Resource with address/description), from
// the provider v1 properties map of pfsense_firewall_alias.
type firewallAliasTargetV0 struct {
	Address     types.String `tfsdk:"address"`
	Description types.String `tfsdk:"description"`
}

// firewallAliasModelV0 is the schema-version-0 (SDKv2-era) state model for
// pfsense_firewall_alias. The tfsdk tags use the OLD attribute names so that
// req.State.Get decodes the prior state verbatim. The implicit SDKv2 `id`
// attribute is intentionally absent: the v1 id is derived from the natural key
// `name` instead (see firewallAliasPriorID).
type firewallAliasModelV0 struct {
	Name        types.String            `tfsdk:"name"`
	Type        types.String            `tfsdk:"type"`
	Description types.String            `tfsdk:"description"`
	Target      []firewallAliasTargetV0 `tfsdk:"target"`
}

var _ resource.ResourceWithUpgradeState = (*firewallAliasResource)(nil)

// firewallAliasPriorSchemaV0 is the SDKv2 pfsense_firewall_alias schema (from the
// provider v1 properties map) translated to framework attributes. It mirrors
// the v0 state exactly: same attribute names, same required/optional flags,
// SDKv2 types mapped per TYPE TRANSLATION rules (the old `target` TypeList of
// nested Resource becomes a ListAttribute whose ElementType is an object type
// with the translated nested fields). The implicit SDKv2 `id` attribute is
// deliberately excluded (the v1 id is derived from the natural key `name`).
var firewallAliasPriorSchemaV0 = schema.Schema{
	Attributes: map[string]schema.Attribute{
		"name":        schema.StringAttribute{Required: true},
		"type":        schema.StringAttribute{Required: true},
		"description": schema.StringAttribute{Optional: true},
		"target": schema.ListAttribute{
			Required: true,
			ElementType: types.ObjectType{AttrTypes: map[string]attr.Type{
				"address":     types.StringType,
				"description": types.StringType,
			}},
		},
	},
}

// UpgradeState implements resource.ResourceWithUpgradeState so that existing
// pfsense_firewall_alias state (schema version 0) is migrated in-place to
// schema version 1 with no recreation.
func (r *firewallAliasResource) UpgradeState(context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &firewallAliasPriorSchemaV0,
			StateUpgrader: r.upgradeStateV0To1,
		},
	}
}

// upgradeStateV0To1 decodes the v0 state, maps every user-configurable value
// to its new home (description→descr, target flattened into address+detail),
// restores the resource id, and writes the v1 state.
func (r *firewallAliasResource) upgradeStateV0To1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior firewallAliasModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cur, diags := prior.toCurrent(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// `name` is Required in v0 so it is always populated, but guard it anyway
	// (defense-in-depth): an empty id would make Read/Update/Delete target the
	// first unrelated alias via findByKey(..., "name", "").
	if prior.Name.ValueString() == "" {
		resp.Diagnostics.AddError(
			"failed to upgrade state for pfsense_firewall_alias",
			"unable to derive the resource id from the prior state: \"name\" is empty",
		)
		return
	}
	cur.ID = firewallAliasPriorID(prior)

	resp.Diagnostics.Append(resp.State.Set(ctx, &cur)...)
}

// firewallAliasPriorID derives the v1 resource id from the natural key
// (`name`), which v0 validated as Required. The v1 id is the alias name
// (matching the old d.SetId(name)), so deriving it keeps state in the v1
// `id == name` contract rather than carrying the raw v0 id over.
func firewallAliasPriorID(prior firewallAliasModelV0) types.String {
	return prior.Name
}

// toCurrent maps every v0 value to its v1 home:
//
//	name        -> name
//	type        -> type
//	description -> descr
//	target      -> address + detail (flattened, zipped by index:
//	              target[i].address -> address[i],
//	              target[i].description -> detail[i],
//	              absent/null description -> "")
//
// The optional `description` goes through emptyToNull so the SDKv2 "" zero
// value does not land in v1 state as an empty string where the framework means
// null; the flattened `detail` elements deliberately keep "" so they stay
// index-aligned with `address`.
//
// The computed "id" is set by the StateUpgrader, not here. Both old lists are
// built with explicit element types; a null prior list stays null.
func (m firewallAliasModelV0) toCurrent(ctx context.Context) (firewallAliasModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	cur := firewallAliasModel{
		Name:  m.Name,
		Type:  m.Type,
		Descr: emptyToNull(m.Description),
	}

	// SDKv2 persisted an unset optional list as [] (its zero value), so a nil
	// or empty target both become null (the list counterpart of emptyToNull).
	if len(m.Target) == 0 {
		cur.Address = types.ListNull(types.StringType)
		cur.Detail = types.ListNull(types.StringType)
		return cur, diags
	}

	addresses := make([]attr.Value, 0, len(m.Target))
	details := make([]attr.Value, 0, len(m.Target))
	allEmpty := true
	for _, target := range m.Target {
		addresses = append(addresses, types.StringValue(target.Address.ValueString()))
		// Missing/null description decodes to "" (ValueString of a null
		// types.String), preserving the 1:1 index alignment with address.
		details = append(details, types.StringValue(target.Description.ValueString()))
		if target.Description.ValueString() != "" {
			allEmpty = false
		}
	}

	cur.Address = types.ListValueMust(types.StringType, addresses)
	// Index alignment with address only carries meaning when at least one
	// description is non-empty; when every description is empty the list is
	// normalised to null so an omitted `detail` config does not plan a
	// spurious [""] -> null diff (the same zero-value normalisation applied
	// everywhere else in this upgrader).
	if allEmpty {
		cur.Detail = types.ListNull(types.StringType)
	} else {
		cur.Detail = types.ListValueMust(types.StringType, details)
	}

	return cur, diags
}
