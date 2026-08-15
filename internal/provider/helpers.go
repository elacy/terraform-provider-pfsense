package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// Object lookup helpers. The pfSense v2 API uses array indices (and, for a few
// models, persistent string IDs) as object identifiers. Because indices are not
// stable across reordering/deletion, resources identify objects by a natural
// key (a unique field) and resolve it to the current index at each operation.
// ---------------------------------------------------------------------------

// findByKey lists pluralPath with keyField__exact=keyValue and returns the first
// matching raw object plus its "id" value (native JSON type: int/string).
// found is false when no object matches.
func findByKey(ctx context.Context, c *client.Client, pluralPath, keyField, keyValue string) (id any, raw map[string]any, found bool, err error) {
	items, err := c.List(ctx, pluralPath, client.Query{}.Set(keyField+"__exact", keyValue))
	if err != nil {
		return nil, nil, false, err
	}
	for _, item := range items {
		var obj map[string]any
		if err := json.Unmarshal(item, &obj); err != nil {
			return nil, nil, false, fmt.Errorf("decoding object: %w", err)
		}
		if objectKey(obj, keyField) == keyValue {
			return obj["id"], obj, true, nil
		}
	}
	return nil, nil, false, nil
}

// findByKeys lists pluralPath (unfiltered) and returns the first object whose
// fields match every key/value in filters.
func findByKeys(ctx context.Context, c *client.Client, pluralPath string, filters map[string]string) (id any, raw map[string]any, found bool, err error) {
	items, err := c.List(ctx, pluralPath, nil)
	if err != nil {
		return nil, nil, false, err
	}
	for _, item := range items {
		var obj map[string]any
		if err := json.Unmarshal(item, &obj); err != nil {
			return nil, nil, false, fmt.Errorf("decoding object: %w", err)
		}
		match := true
		for k, v := range filters {
			if objectKey(obj, k) != v {
				match = false
				break
			}
		}
		if match {
			return obj["id"], obj, true, nil
		}
	}
	return nil, nil, false, nil
}

// formatID converts a native id (int/string) into a query-parameter string.
func formatID(id any) string {
	switch v := id.(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case json.Number:
		return v.String()
	case string:
		return v
	default:
		return fmt.Sprintf("%v", id)
	}
}

// objectKey extracts a string field from a raw object, handling the JSON
// number/string/bool variants.
func objectKey(obj map[string]any, key string) string {
	switch v := obj[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case nil:
		return ""
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

// ---------------------------------------------------------------------------
// Framework <-> Go type conversion helpers.
// ---------------------------------------------------------------------------

// ptrString returns a *string from a framework String, or nil when null/unknown.
func ptrString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// ptrBool returns a *bool from a framework Bool, or nil when null/unknown.
func ptrBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// ptrInt returns a *int64 from a framework Int64, or nil when null/unknown.
func ptrInt(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// listStrings converts a framework List of String to []string. Null/unknown
// yields nil.
func listStrings(v types.List) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	elems := v.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		s, ok := e.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		out = append(out, s.ValueString())
	}
	return out
}

// setStrings converts a framework Set of String to []string.
func setStrings(v types.Set) []string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	elems := v.Elements()
	out := make([]string, 0, len(elems))
	for _, e := range elems {
		s, ok := e.(types.String)
		if !ok || s.IsNull() || s.IsUnknown() {
			continue
		}
		out = append(out, s.ValueString())
	}
	return out
}

// strValue wraps a *string into a framework String (null when nil).
func strValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// boolValue wraps a *bool into a framework Bool.
func boolValue(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// intValue wraps an *int64 into a framework Int64.
func intValue(i *int64) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*i)
}

// strListValue wraps []string into a framework List of String.
func strListValue(ctx context.Context, s []string) types.List {
	if s == nil {
		return types.ListNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(s))
	for _, v := range s {
		elems = append(elems, types.StringValue(v))
	}
	out, diags := types.ListValue(types.StringType, elems)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return out
}

// strSetValue wraps []string into a framework Set of String.
func strSetValue(ctx context.Context, s []string) types.Set {
	if s == nil {
		return types.SetNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(s))
	for _, v := range s {
		elems = append(elems, types.StringValue(v))
	}
	out, diags := types.SetValue(types.StringType, elems)
	if diags.HasError() {
		return types.SetNull(types.StringType)
	}
	return out
}

// ---------------------------------------------------------------------------
// Payload building helpers.
// ---------------------------------------------------------------------------

// setString sets m[key] to the framework string value, omitting the key when
// null/unknown (so the API applies its default).
func setString(m map[string]any, key string, v types.String) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	m[key] = v.ValueString()
}

// setStringForce sets m[key] to "" when null/unknown (used for fields that must
// be explicitly cleared).
func setStringForce(m map[string]any, key string, v types.String) {
	if v.IsNull() || v.IsUnknown() {
		m[key] = ""
		return
	}
	m[key] = v.ValueString()
}

func setBool(m map[string]any, key string, v types.Bool) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	m[key] = v.ValueBool()
}

func setInt(m map[string]any, key string, v types.Int64) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	m[key] = v.ValueInt64()
}

func setStringList(m map[string]any, key string, v types.List) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	m[key] = listStrings(v)
}

func setStringSet(m map[string]any, key string, v types.Set) {
	if v.IsNull() || v.IsUnknown() {
		return
	}
	m[key] = setStrings(v)
}

// ---------------------------------------------------------------------------
// Response decoding helpers.
// ---------------------------------------------------------------------------

// applyNow marks a request payload to apply changes immediately (the API's
// `apply` control parameter). Models that always apply ignore it; models that
// defer changes (firewall, interfaces, routing, dhcp, dns, vpn) reload the
// relevant subsystem on mutation.
func applyNow(m map[string]any) map[string]any {
	m["apply"] = true
	return m
}

func getString(obj map[string]any, key string) *string {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		return &t
	case float64:
		s := strconv.FormatFloat(t, 'f', -1, 64)
		return &s
	default:
		s := fmt.Sprintf("%v", v)
		return &s
	}
}

func getBool(obj map[string]any, key string) *bool {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
}

func getInt(obj map[string]any, key string) *int64 {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case float64:
		i := int64(t)
		return &i
	case int:
		i := int64(t)
		return &i
	case int64:
		return &t
	case string:
		i, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return nil
		}
		return &i
	default:
		return nil
	}
}

// getStringSlice extracts a []string field (JSON array of strings).
func getStringSlice(obj map[string]any, key string) []string {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		// Some APIs return a single string; coerce to a one-element slice.
		if s, ok := v.(string); ok {
			return []string{s}
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		switch t := e.(type) {
		case string:
			out = append(out, t)
		case float64:
			out = append(out, strconv.FormatFloat(t, 'f', -1, 64))
		}
	}
	return out
}

// getSliceMap extracts a []map[string]any field (JSON array of objects),
// used for nested model collections.
func getSliceMap(obj map[string]any, key string) []map[string]any {
	v, ok := obj[key]
	if !ok || v == nil {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, e := range arr {
		if m, ok := e.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// decodeObject unmarshals a raw JSON object into a map.
func decodeObject(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return map[string]any{}, nil
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("decoding object: %w", err)
	}
	return obj, nil
}
