package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Schema attribute constructors shared across resources to keep resource
// definitions concise.

func computedIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Computed:    true,
		Description: "The resource identifier.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func requiredNameAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: "The object name. Unique; immutable after creation.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func requiredStringAttribute(desc string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{Required: true, Description: desc, Validators: validators}
}

// optionalStringAttribute returns an Optional string attribute. Shape
// validators are wrapped so they also accept the empty string: this provider's
// convention is that the pfSense API echoes "" for an Optional string it
// considers unset, and practitioners pin `attr = ""` so the plan settles, so a
// validator that rejects "" would make legitimate "unset" configs unwritable.
func optionalStringAttribute(desc string, validators ...validator.String) schema.StringAttribute {
	wrapped := make([]validator.String, 0, len(validators))
	for _, v := range validators {
		wrapped = append(wrapped, allowEmpty(v))
	}
	return schema.StringAttribute{Optional: true, Description: desc, Validators: wrapped}
}

// allowEmpty wraps a string validator so it accepts "" in addition to whatever
// it already accepts, treating the empty string as the "unset" sentinel.
func allowEmpty(v validator.String) validator.String {
	return stringvalidator.Any(stringvalidator.OneOf(""), v)
}

func optionalBoolAttribute(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Description: desc}
}

func optionalIntAttribute(desc string, validators ...validator.Int64) schema.Int64Attribute {
	return schema.Int64Attribute{Optional: true, Description: desc, Validators: validators}
}

func enumAttribute(desc string, choices ...string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: desc,
		Validators:  []validator.String{stringvalidator.OneOf(choices...)},
	}
}

// The constructors below cover API-assigned (computed) fields, credentials,
// natural-key attributes and list-of-string fields, which the VPN and
// system-extra models use heavily.

// keyAttribute is a required, immutable natural-key attribute: its value
// identifies the object in the API, so changing it replaces the resource.
func keyAttribute(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: desc,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

// parentIDAttribute is the required natural key of the parent object a
// parent-child resource belongs to. It behaves identically to keyAttribute but
// is named separately so parent references read distinctly at the call site.
func parentIDAttribute(desc string) schema.StringAttribute {
	return keyAttribute(desc)
}

// computedStringAttribute is a computed string with no plan modifier: on an
// unknown it stays unknown and is populated by Read/Update.
func computedStringAttribute(desc string) schema.StringAttribute {
	return schema.StringAttribute{Computed: true, Description: desc}
}

func computedIntAttribute(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{Computed: true, Description: desc}
}

func computedBoolAttribute(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{Computed: true, Description: desc}
}

func requiredIntAttribute(desc string, validators ...validator.Int64) schema.Int64Attribute {
	return schema.Int64Attribute{Required: true, Description: desc, Validators: validators}
}

// sensitiveStringAttribute is an optional attribute holding a credential; its
// value is redacted from plan output and logs.
func sensitiveStringAttribute(desc string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Sensitive: true, Description: desc}
}

// requiredSensitiveStringAttribute is a required attribute holding a
// credential; its value is redacted from plan output and logs.
func requiredSensitiveStringAttribute(desc string, validators ...validator.String) schema.StringAttribute {
	return schema.StringAttribute{Required: true, Sensitive: true, Description: desc, Validators: validators}
}

func requiredEnumAttribute(desc string, choices ...string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: desc,
		Validators:  []validator.String{stringvalidator.OneOf(choices...)},
	}
}

func optionalStringListAttribute(desc string, validators ...validator.List) schema.ListAttribute {
	return schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: desc, Validators: validators}
}

func optionalIntListAttribute(desc string) schema.ListAttribute {
	return schema.ListAttribute{ElementType: types.Int64Type, Optional: true, Description: desc}
}

func requiredStringListAttribute(desc string, validators ...validator.List) schema.ListAttribute {
	return schema.ListAttribute{
		ElementType: types.StringType,
		Required:    true,
		Description: desc,
		Validators:  validators,
	}
}
