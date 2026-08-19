package provider

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// priorResourceID extracts the implicit "id" attribute from the raw prior
// state during a schema version upgrade.
//
// The "id" attribute is not part of the PriorSchema — it is implicit in both
// the SDKv2 (v1) provider and the framework (v2) provider — so it cannot be
// read via req.State.Get. Instead the raw state is unmarshalled against a
// minimal object type that only declares "id", with undefined attributes
// ignored (the same option the framework server itself uses when decoding the
// prior state). Any decoding problem yields an empty string so callers can
// fall back to the natural-key attributes.
func priorResourceID(raw *tfprotov6.RawState) string {
	if raw == nil {
		return ""
	}

	value, err := raw.UnmarshalWithOpts(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{"id": tftypes.String}},
		tfprotov6.UnmarshalOpts{
			ValueFromJSONOpts: tftypes.ValueFromJSONOpts{
				IgnoreUndefinedAttributes: true,
			},
		},
	)
	if err != nil {
		return ""
	}

	var attrs map[string]tftypes.Value
	if err := value.As(&attrs); err != nil {
		return ""
	}

	idValue, ok := attrs["id"]
	if !ok || idValue.IsNull() || !idValue.IsKnown() {
		return ""
	}

	var id string
	if err := idValue.As(&id); err != nil {
		return ""
	}
	return id
}

// emptyToNull normalises an optional string carried over from SDKv2 state.
//
// The SDKv2 always persisted unset optional strings as "" (its zero value),
// while the framework and the pfSense API both treat "" and "unset" as
// different things. Copying "" verbatim into version-1 state therefore makes
// `terraform plan -refresh=false` report a spurious "" → null diff on every
// attribute the practitioner never configured. Mapping "" back to null
// restores the framework's notion of an unset optional attribute.
func emptyToNull(v types.String) types.String {
	if v.IsNull() || v.IsUnknown() {
		return v
	}
	if v.ValueString() == "" {
		return types.StringNull()
	}
	return v
}

// falseToNull normalises an optional bool carried over from SDKv2 state.
//
// This is the bool counterpart of emptyToNull: the SDKv2 persisted an unset
// optional bool as its zero value `false`, while the framework distinguishes
// `false` from "unset". Copying `false` verbatim into version-1 state makes
// every practitioner who never configured the attribute see a spurious
// `false` → null diff on the next plan.
//
// The ambiguity is unavoidable and one-way: SDKv2 state cannot tell an
// explicit `false` from an unset attribute, so both become null. That is the
// safe direction — the attribute is Optional, so a null carries no meaning of
// its own, and the first Read after the upgrade repopulates the state value
// from the pfSense API. That repopulates *state* only: because the attribute
// is Optional and not Computed, a config that omits it may still plan a
// persistent diff against the API's value on later refreshes — a
// pre-existing resource-level behaviour the upgrader cannot change. Never use
// this on a Required attribute.
func falseToNull(v types.Bool) types.Bool {
	if v.IsUnknown() {
		return v
	}
	if v.IsNull() || !v.ValueBool() {
		return types.BoolNull()
	}
	return v
}

// zeroToNull normalises an optional integer carried over from SDKv2 state.
//
// This is the integer counterpart of emptyToNull and falseToNull: the SDKv2
// persisted an unset optional int as its zero value `0`, so copying it
// verbatim produces a spurious `0` → null diff on every attribute the
// practitioner never configured.
//
// As with falseToNull the mapping is ambiguous and one-way — an explicit `0`
// and an unset attribute are indistinguishable in SDKv2 state and both become
// null — and the state value self-heals on the first Read (which repopulates
// state, not the plan: Optional-not-Computed attributes may still show a
// persistent diff on later refreshes). Like emptyToNull it is also applied
// to attributes that are Required in v2 but Optional in v0 (e.g. the network
// interface `subnet`): a null there fails the next plan loudly instead of
// silently carrying a semantically wrong zero value, and the break is
// documented in docs/UPGRADING.md. Do not use it where `0` is a meaningful
// configured value that the API does not report back. (`pcp` on
// pfsense_interface_vlan is an accepted exception: `0` is a valid 802.1p
// priority, but the attribute is Optional-not-Computed so an explicit `0`
// still plans against the API's `0`; the trade-off is documented at the call
// site rather than here.)
func zeroToNull(v types.Int64) types.Int64 {
	if v.IsUnknown() {
		return v
	}
	if v.IsNull() || v.ValueInt64() == 0 {
		return types.Int64Null()
	}
	return v
}

// emptyListToNull normalises an optional list carried over from SDKv2 state.
//
// This is the list counterpart of emptyToNull: the SDKv2 cannot persist a null
// list — an unset optional TypeList is written to state as [] (its zero value)
// — so copying it verbatim produces a spurious [] → null diff on every list
// the practitioner never configured. The ambiguity is the same one-way
// mapping as falseToNull/zeroToNull: an explicit empty list and an unset list
// are indistinguishable in SDKv2 state, and both become null. The attribute is
// Optional, so null carries no meaning of its own and the first Read after the
// upgrade repopulates the state value from the pfSense API. (State, not plan:
// an Optional-not-Computed attribute a config omits may still show a
// persistent diff on later refreshes.) Do not use it where an empty list is a
// meaningful configured value — see the TCP-flag note in
// firewall_rule_upgrade.go for a case where that trade-off is taken
// deliberately (an empty set is meaningful there, and is normalised to null
// anyway).
func emptyListToNull(ctx context.Context, v types.List) types.List {
	if v.IsNull() || v.IsUnknown() {
		return v
	}
	if len(v.Elements()) == 0 {
		return types.ListNull(v.ElementType(ctx))
	}
	return v
}

// upgradeStringToInt64 converts a version-0 string attribute that held a
// number into the version-1 integer type. An empty string (the SDKv2 zero
// value for an unset optional string) maps to null; a non-numeric value
// yields a diagnostic rather than silently corrupting state.
//
// resourceType is the version-0 Terraform type name (e.g. "pfsense_interface")
// and is only used to make the diagnostic actionable.
func upgradeStringToInt64(v types.String, attrName, resourceType string, diags *diag.Diagnostics) types.Int64 {
	if v.IsNull() || v.ValueString() == "" {
		return types.Int64Null()
	}
	n, err := strconv.ParseInt(v.ValueString(), 10, 64)
	if err != nil {
		// The v0 attribute name is referenced in the message text rather than
		// anchored via path.Root: the two call sites pass v0 names whose v1
		// counterparts differ ("mss" is shared with v1, but "subnet_v6" ->
		// "subnetv6"), and anchoring would surface an error pointing at a
		// non-existent attribute.
		diags.AddError(
			"Invalid Value During State Upgrade",
			"An error was encountered when upgrading "+resourceType+" state "+
				"from schema version 0 to version 1: the value of attribute "+
				"\""+attrName+"\" (\""+v.ValueString()+"\") could not be parsed as an integer. "+
				"Fix the value in pfSense and refresh the state under v1, or hand-edit the backed-up state file, so the attribute holds a numeric value, then re-run the upgrade.",
		)
		return types.Int64Null()
	}
	return types.Int64Value(n)
}
