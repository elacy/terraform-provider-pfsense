package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

func requiredStringAttribute(desc string) schema.StringAttribute {
	return schema.StringAttribute{Required: true, Description: desc}
}

func optionalStringAttribute(desc string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Description: desc}
}

func optionalBoolAttribute(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Description: desc}
}

func optionalIntAttribute(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{Optional: true, Description: desc}
}

func enumAttribute(desc string, choices ...string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: desc,
		Validators:  []validator.String{stringvalidator.OneOf(choices...)},
	}
}
