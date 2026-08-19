package provider

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
)

// Fuzz targets for the provider's untrusted-input boundaries: composite-key
// parsers, numeric coercion, and JSON decoding. Run with:
//
//	go test ./internal/provider -run '^$' -fuzz FuzzAtoi64 -fuzztime 30s
//
// Each target asserts an invariant that must hold for ALL inputs (not just the
// seed corpus); a violation is reported as a crash by the fuzzer.

// FuzzAtoi64 asserts atoi64 never returns a negative value for a non-negative
// digit prefix, and matches strconv.ParseInt when the prefix is in range.
// Integer overflow must not silently wrap to a negative number (a negative
// value could resolve to the wrong object and be updated/deleted).
func FuzzAtoi64(f *testing.F) {
	f.Add("0")
	f.Add("123")
	f.Add("abc")
	f.Add("12a34")
	f.Add("999999999999999999999999999999")
	f.Add("-5")
	f.Add("")
	f.Add(" 42")
	f.Fuzz(func(t *testing.T, s string) {
		p := leadingDigits(s)
		got := atoi64(s)
		if p == "" {
			if got != 0 {
				t.Fatalf("atoi64(%q) = %d, want 0 (no leading digits)", s, got)
			}
			return
		}
		want, err := strconv.ParseInt(p, 10, 64)
		if err == nil {
			if got != want {
				t.Fatalf("atoi64(%q) = %d, want %d", s, got, want)
			}
			return
		}
		// Overflow: atoi64 must saturate to MaxInt64, not silently wrap negative.
		if got != math.MaxInt64 {
			t.Fatalf("atoi64(%q) overflowed to %d, want math.MaxInt64", s, got)
		}
	})
}

func leadingDigits(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return s[:i]
}

// FuzzSplitKeyN asserts splitKeyN always returns exactly n parts, and that it
// never panics on any input string for part counts normalised into 1..64.
func FuzzSplitKeyN(f *testing.F) {
	f.Add("a|b|c", 3)
	f.Add("a", 5)
	f.Add("a|b|c|d|e|f", 2)
	f.Add("", 1)
	f.Add("||||", 4)
	f.Add("trailing|", 3)
	f.Fuzz(func(t *testing.T, id string, n int) {
		// Normalise the part count into the 1..64 range splitKeyN supports
		// rather than discarding generated inputs wholesale.
		n = (n%64 + 64) % 64
		if n == 0 {
			n = 64
		}
		parts := splitKeyN(id, n)
		if len(parts) != n {
			t.Fatalf("splitKeyN(%q, %d) returned %d parts, want %d", id, n, len(parts), n)
		}
		// splitKeyN must equal strings.SplitN padded with empty strings at
		// the END (missing trailing parts become "").
		base := strings.SplitN(id, "|", n)
		for i := range base {
			if parts[i] != base[i] {
				t.Fatalf("splitKeyN(%q, %d): part[%d]=%q, want %q", id, n, i, parts[i], base[i])
			}
		}
		for i := len(base); i < n; i++ {
			if parts[i] != "" {
				t.Fatalf("splitKeyN(%q, %d): trailing part[%d]=%q, want empty", id, n, i, parts[i])
			}
		}
	})
}

// FuzzSplitRouteKey asserts the network/gateway split round-trips: the two
// halves, rejoined with '|', reproduce the input when it contains a '|'.
func FuzzSplitRouteKey(f *testing.F) {
	f.Add("10.0.0.0/24|WANGW")
	f.Add("nokey")
	f.Add("a|b|c|d")
	f.Add("|")
	f.Add("")
	f.Fuzz(func(t *testing.T, id string) {
		network, gateway := splitRouteKey(id)
		if strings.Contains(id, "|") {
			if network+"|"+gateway != id {
				t.Fatalf("splitRouteKey(%q) = (%q, %q); round-trip mismatch", id, network, gateway)
			}
			// The split must be at the LAST separator, otherwise gateway could
			// still contain '|' and `network|gateway` import IDs would be
			// ambiguous.
			if strings.Contains(gateway, "|") {
				t.Fatalf("splitRouteKey(%q) split at a non-final separator: gateway %q contains '|'", id, gateway)
			}
		} else {
			if network != id || gateway != "" {
				t.Fatalf("splitRouteKey(%q) = (%q, %q); want (%q, \"\")", id, network, gateway, id)
			}
		}
	})
}

// FuzzDecodeObject asserts decodeObject is panic-free and that it never returns
// both a nil map and a nil error for any input, including empty.
func FuzzDecodeObject(f *testing.F) {
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))
	f.Add([]byte(`{"nested":{"x":[1,2,3]}}`))
	f.Add([]byte(`not json`))
	f.Add([]byte(`{"unterminated":`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		obj, err := decodeObject(raw)
		if err == nil && obj == nil {
			t.Fatalf("decodeObject(%q) returned nil map and nil error", raw)
		}
	})
}

// FuzzGetInt asserts getInt never panics regardless of the shape of the value
// stored in a map (fed as arbitrary JSON decoded to map[string]any).
func FuzzGetInt(f *testing.F) {
	f.Add([]byte(`{"k":42}`))
	f.Add([]byte(`{"k":"42"}`))
	f.Add([]byte(`{"k":null}`))
	f.Add([]byte(`{"k":1e300}`))
	f.Add([]byte(`{"k":["not","an","int"]}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Skip()
		}
		// Exercise both a present key and an absent one.
		_ = getInt(m, "k")
		_ = getInt(m, "missing")
	})
}

// FuzzFormatID asserts formatID never panics on any JSON-decoded id value and
// leaves string ids unchanged (pfSense object ids are either numeric or
// string; a mangled id could resolve to the wrong object).
func FuzzFormatID(f *testing.F) {
	f.Add([]byte(`42`))
	f.Add([]byte(`"42"`))
	f.Add([]byte(`"wan"`))
	f.Add([]byte(`null`))
	f.Add([]byte(`3.14`))
	f.Add([]byte(`[1,2,3]`))
	f.Add([]byte(`{"a":1}`))
	f.Add([]byte(`true`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var id any
		if err := json.Unmarshal(raw, &id); err != nil {
			t.Skip()
		}
		got := formatID(id)
		// String ids must round-trip unchanged.
		if s, ok := id.(string); ok && got != s {
			t.Fatalf("formatID(%q) = %q, want the string unchanged", s, got)
		}
	})
}

// FuzzObjectKey asserts objectKey never panics on any JSON-decoded map value,
// for both a present and an absent key.
func FuzzObjectKey(f *testing.F) {
	f.Add([]byte(`{"k":"v"}`))
	f.Add([]byte(`{"k":42}`))
	f.Add([]byte(`{"k":true}`))
	f.Add([]byte(`{"k":null}`))
	f.Add([]byte(`{"k":[1,2]}`))
	f.Add([]byte(`{"k":{"nested":"obj"}}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Skip()
		}
		_ = objectKey(m, "k")
		_ = objectKey(m, "missing")
	})
}

// FuzzGetString asserts getString never panics on any JSON-decoded map value,
// for both a present and an absent key.
func FuzzGetString(f *testing.F) {
	f.Add([]byte(`{"k":"v"}`))
	f.Add([]byte(`{"k":42}`))
	f.Add([]byte(`{"k":1.5}`))
	f.Add([]byte(`{"k":true}`))
	f.Add([]byte(`{"k":null}`))
	f.Add([]byte(`{"k":[1,"x"]}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Skip()
		}
		_ = getString(m, "k")
		_ = getString(m, "missing")
	})
}

// FuzzGetBool asserts getBool never panics on any JSON-decoded map value, for
// both a present and an absent key.
func FuzzGetBool(f *testing.F) {
	f.Add([]byte(`{"k":true}`))
	f.Add([]byte(`{"k":false}`))
	f.Add([]byte(`{"k":"true"}`))
	f.Add([]byte(`{"k":1}`))
	f.Add([]byte(`{"k":null}`))
	f.Add([]byte(`{"k":[true]}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Skip()
		}
		_ = getBool(m, "k")
		_ = getBool(m, "missing")
	})
}

// FuzzGetStringSlice asserts getStringSlice never panics on any JSON-decoded
// map value, coerces a bare string to a one-element slice, and never invents
// elements: everything it returns must originate from a string or float64
// element of the input list.
func FuzzGetStringSlice(f *testing.F) {
	f.Add([]byte(`{"k":["a","b"]}`))
	f.Add([]byte(`{"k":"single"}`))
	f.Add([]byte(`{"k":[1,2.5,"x",true,null,{"o":1},["n"]]}`))
	f.Add([]byte(`{"k":null}`))
	f.Add([]byte(`{"k":[]}`))
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Skip()
		}
		got := getStringSlice(m, "k")
		_ = getStringSlice(m, "missing")
		v, ok := m["k"]
		if !ok || v == nil {
			if got != nil {
				t.Fatalf("getStringSlice(missing/nil key) = %v, want nil", got)
			}
			return
		}
		switch val := v.(type) {
		case string:
			if len(got) != 1 || got[0] != val {
				t.Fatalf("getStringSlice(single string %q) = %v, want [%q]", val, got, val)
			}
		case []any:
			if len(got) > len(val) {
				t.Fatalf("getStringSlice returned %d elements for %d-element input: %v", len(got), len(val), got)
			}
			for _, g := range got {
				if !sliceElemOriginatesFrom(val, g) {
					t.Fatalf("getStringSlice returned %q, which does not originate from any input element %v", g, val)
				}
			}
		}
	})
}

// sliceElemOriginatesFrom reports whether g is either a string element of arr
// or the decimal rendering of a float64 element of arr (the only two shapes
// getStringSlice is allowed to emit).
func sliceElemOriginatesFrom(arr []any, g string) bool {
	for _, e := range arr {
		switch e := e.(type) {
		case string:
			if e == g {
				return true
			}
		case float64:
			if strconv.FormatFloat(e, 'f', -1, 64) == g {
				return true
			}
		}
	}
	return false
}

// FuzzAliasDecode exercises the firewall_alias Read decode path end-to-end:
// arbitrary JSON decoded to map[string]any is fed through the same
// getString/getStringSlice calls the model Read makes, then through the
// framework value wrappers. The invariant is panic-freedom: a malformed API
// response must never crash the Read.
func FuzzAliasDecode(f *testing.F) {
	f.Add([]byte(`{"type":"host","descr":"d","address":["10.0.0.1/32"],"detail":[]}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"type":42,"descr":true,"address":"10.0.0.1","detail":[1,2]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var obj map[string]any
		if err := json.Unmarshal(raw, &obj); err != nil {
			t.Skip()
		}
		_ = strValue(getString(obj, "type"))
		_ = strValue(getString(obj, "descr"))
		_ = strListValue(context.Background(), getStringSlice(obj, "address"))
		_ = strListValue(context.Background(), getStringSlice(obj, "detail"))
	})
}
