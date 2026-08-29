//go:build goexperiment.jsonv2

package jq

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// taggedJSONName returns the JSON object name a "json" struct tag selects,
// and reports whether the tag names the field at all.
//
// With the jsonv2 GOEXPERIMENT, enabled by default as of Go 1.27,
// encoding/json accepts almost any unescaped name and reserves only the
// comma, the backslash and the three quote runes. A name that stops on a
// reserved rune is reparsed as a tag option, which keeps a leading Go
// identifier and otherwise leaves the field using its Go name.
func taggedJSONName(tag string) (name string, tagged bool) {
	if tag == "" || strings.HasPrefix(tag, ",") {
		return
	}
	n := len(tag) - len(strings.TrimLeftFunc(tag, unreservedTagRune))
	name = tag[:n]
	if n < len(tag) && !strings.HasPrefix(tag[n:], ",") {
		if r, _ := utf8.DecodeRuneInString(tag); r != '_' && !unicode.IsLetter(r) {
			return "", false
		}
		name = tag[:len(tag)-len(strings.TrimLeftFunc(tag, identifierTagRune))]
	}
	if !utf8.ValidString(name) {
		name = string([]rune(name))
	}
	return name, true
}

// unreservedTagRune reports whether r may appear in an unescaped tag name.
func unreservedTagRune(r rune) bool {
	return !strings.ContainsRune(",\\'\"`", r)
}

// identifierTagRune reports whether r may continue a Go identifier used as a
// tag option.
func identifierTagRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}
