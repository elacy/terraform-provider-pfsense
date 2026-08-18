package provider

import (
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
