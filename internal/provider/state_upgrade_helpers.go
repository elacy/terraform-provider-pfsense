package provider

import (
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
		diags.AddAttributeError(
			path.Root(attrName),
			"Invalid Value During State Upgrade",
			"An error was encountered when upgrading "+resourceType+" state "+
				"from schema version 0 to version 1: the value of attribute "+
				"\""+attrName+"\" (\""+v.ValueString()+"\") could not be parsed as an integer. "+
				"Set a valid integer value or remove the attribute from the resource configuration, then re-apply.",
		)
		return types.Int64Null()
	}
	return types.Int64Value(n)
}
