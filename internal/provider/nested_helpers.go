package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// attrToGo converts a framework attr.Value to its Go representation for use in
// a JSON payload.
func attrToGo(v attr.Value) any {
	switch t := v.(type) {
	case types.String:
		if t.IsNull() || t.IsUnknown() {
			return nil
		}
		return t.ValueString()
	case types.Bool:
		if t.IsNull() || t.IsUnknown() {
			return nil
		}
		return t.ValueBool()
	case types.Int64:
		if t.IsNull() || t.IsUnknown() {
			return nil
		}
		return t.ValueInt64()
	case types.Float64:
		if t.IsNull() || t.IsUnknown() {
			return nil
		}
		return t.ValueFloat64()
	case types.List:
		if t.IsNull() || t.IsUnknown() {
			return nil
		}
		out := make([]any, 0, len(t.Elements()))
		for _, e := range t.Elements() {
			out = append(out, attrToGo(e))
		}
		return out
	default:
		// Fall back to the string representation for other primitive types.
		return nil
	}
}

// listObjects converts a types.List of objects to []map[string]any, using the
// object's attribute names as keys. Null/unknown lists yield nil.
func listObjects(v types.List) []map[string]any {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	out := make([]map[string]any, 0, len(v.Elements()))
	for _, e := range v.Elements() {
		obj, ok := e.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}
		m := make(map[string]any, len(obj.Attributes()))
		for name, val := range obj.Attributes() {
			m[name] = attrToGo(val)
		}
		out = append(out, m)
	}
	return out
}

// objectsToListValue builds a framework List of objects from []map[string]any
// using the supplied attr.Type for the object shape. Used to populate nested
// attributes on read.
func objectsToListValue(ctx context.Context, objs []map[string]any, objType attr.Type) types.List {
	if objs == nil {
		return types.ListNull(objType)
	}
	elems := make([]attr.Value, 0, len(objs))
	objAttrs := objType.(types.ObjectType).AttrTypes
	for _, m := range objs {
		attrs := make(map[string]attr.Value, len(objAttrs))
		for name, typ := range objAttrs {
			attrs[name] = goToAttrValue(typ, m[name])
		}
		obj, diags := types.ObjectValue(objAttrs, attrs)
		if diags.HasError() {
			continue
		}
		elems = append(elems, obj)
	}
	out, diags := types.ListValue(objType, elems)
	if diags.HasError() {
		return types.ListNull(objType)
	}
	return out
}

// goToAttrValue converts a decoded JSON value into the matching framework value.
func goToAttrValue(typ attr.Type, v any) attr.Value {
	if lt, ok := typ.(types.ListType); ok {
		arr, ok := v.([]any)
		if !ok {
			return types.ListNull(lt.ElemType)
		}
		elems := make([]attr.Value, 0, len(arr))
		for _, e := range arr {
			elems = append(elems, goToAttrValue(lt.ElemType, e))
		}
		out, diags := types.ListValue(lt.ElemType, elems)
		if diags.HasError() {
			return types.ListNull(lt.ElemType)
		}
		return out
	}
	if v == nil {
		return nullValue(typ)
	}
	switch typ {
	case types.StringType:
		if s, ok := v.(string); ok {
			return types.StringValue(s)
		}
		if s, ok := v.(float64); ok {
			return types.StringValue(formatID(s))
		}
		return types.StringNull()
	case types.BoolType:
		if b, ok := v.(bool); ok {
			return types.BoolValue(b)
		}
		return types.BoolNull()
	case types.Int64Type:
		switch n := v.(type) {
		case float64:
			return types.Int64Value(int64(n))
		case int64:
			return types.Int64Value(n)
		default:
			return types.Int64Null()
		}
	default:
		return nullValue(typ)
	}
}

// nullValue returns the null value for a given framework type.
func nullValue(typ attr.Type) attr.Value {
	if lt, ok := typ.(types.ListType); ok {
		return types.ListNull(lt.ElemType)
	}
	switch typ {
	case types.StringType:
		return types.StringNull()
	case types.BoolType:
		return types.BoolNull()
	case types.Int64Type:
		return types.Int64Null()
	case types.Float64Type:
		return types.Float64Null()
	case types.NumberType:
		return types.NumberNull()
	default:
		return types.StringNull()
	}
}
