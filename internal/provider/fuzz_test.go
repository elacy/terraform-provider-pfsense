package provider

import (
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
