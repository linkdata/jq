//go:build !goexperiment.jsonv2

package jq

import (
	"strings"
	"unicode"
)

// taggedJSONName returns the JSON object name a "json" struct tag selects,
// and reports whether the tag names the field at all.
//
// Without the jsonv2 GOEXPERIMENT, encoding/json takes the name up to the
// first comma and accepts only letters, digits and a fixed set of
// punctuation runes in it. Any other rune leaves the field using its Go name.
func taggedJSONName(tag string) (name string, tagged bool) {
	name, _, _ = strings.Cut(tag, ",")
	if name == "" {
		return
	}
	for _, r := range name {
		if strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", r) {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", false
		}
	}
	return name, true
}
