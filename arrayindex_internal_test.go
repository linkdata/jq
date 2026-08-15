package jq

import "testing"

func TestParseArrayIndex(t *testing.T) {
	maxInt := uint64(^uint(0) >> 1)
	tests := []struct {
		name  string
		input string
		value uint64
		valid bool
	}{
		{name: "zero", input: "0", value: 0, valid: true},
		{name: "one", input: "1", value: 1, valid: true},
		{name: "multiple digits", input: "10", value: 10, valid: true},
		{name: "32-bit signed maximum", input: "2147483647", value: 2147483647, valid: true},
		{name: "above 32-bit signed maximum", input: "2147483648", value: 2147483648, valid: true},
		{name: "maximum array index", input: "4294967294", value: 4294967294, valid: true},
		{name: "empty", input: ""},
		{name: "leading zero", input: "01"},
		{name: "zeroes", input: "00"},
		{name: "positive sign", input: "+1"},
		{name: "negative zero", input: "-0"},
		{name: "negative", input: "-1"},
		{name: "decimal point", input: "1.0"},
		{name: "exponent", input: "1e0"},
		{name: "leading space", input: " 1"},
		{name: "trailing space", input: "1 "},
		{name: "non-ASCII digit", input: "１"},
		{name: "reserved maximum", input: "4294967295"},
		{name: "above uint32", input: "4294967296"},
		{name: "above uint64", input: "18446744073709551616"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			index, ok := parseArrayIndex(tc.input)
			wantOK := tc.valid && tc.value <= maxInt
			if ok != wantOK {
				t.Fatalf("parseArrayIndex(%q) ok = %t, want %t", tc.input, ok, wantOK)
			}
			if ok && uint64(index) != tc.value {
				t.Fatalf("parseArrayIndex(%q) = %d, want %d", tc.input, index, tc.value)
			}
		})
	}
}
