package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The SDKv2 could not distinguish an explicitly configured zero value from an
// unset optional attribute — it persisted "" / false / 0 for both. The
// normalisers map every zero value to null so the framework sees an unset
// optional attribute; the first Read after the upgrade re-reads the real value
// from the pfSense API.
func TestEmptyToNull(t *testing.T) {
	if got := emptyToNull(types.StringValue("")); !got.IsNull() {
		t.Errorf("emptyToNull(\"\") = %v, want null", got)
	}
	if got := emptyToNull(types.StringValue("wan")); got.ValueString() != "wan" {
		t.Errorf("emptyToNull(\"wan\") = %v, want wan", got)
	}
	if got := emptyToNull(types.StringNull()); !got.IsNull() {
		t.Errorf("emptyToNull(null) = %v, want null", got)
	}
	if got := emptyToNull(types.StringUnknown()); !got.IsUnknown() {
		t.Errorf("emptyToNull(unknown) = %v, want unknown", got)
	}
}

func TestFalseToNull(t *testing.T) {
	if got := falseToNull(types.BoolValue(false)); !got.IsNull() {
		t.Errorf("falseToNull(false) = %v, want null", got)
	}
	if got := falseToNull(types.BoolValue(true)); !got.ValueBool() {
		t.Errorf("falseToNull(true) = %v, want true", got)
	}
	if got := falseToNull(types.BoolNull()); !got.IsNull() {
		t.Errorf("falseToNull(null) = %v, want null", got)
	}
	if got := falseToNull(types.BoolUnknown()); !got.IsUnknown() {
		t.Errorf("falseToNull(unknown) = %v, want unknown", got)
	}
}

func TestZeroToNull(t *testing.T) {
	if got := zeroToNull(types.Int64Value(0)); !got.IsNull() {
		t.Errorf("zeroToNull(0) = %v, want null", got)
	}
	if got := zeroToNull(types.Int64Value(1500)); got.ValueInt64() != 1500 {
		t.Errorf("zeroToNull(1500) = %v, want 1500", got)
	}
	if got := zeroToNull(types.Int64Null()); !got.IsNull() {
		t.Errorf("zeroToNull(null) = %v, want null", got)
	}
	if got := zeroToNull(types.Int64Unknown()); !got.IsUnknown() {
		t.Errorf("zeroToNull(unknown) = %v, want unknown", got)
	}
}
