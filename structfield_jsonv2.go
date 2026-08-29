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
	name, _, _ = strings.Cut(tag, ",")
	if strings.ContainsAny(name, "\\'\"`") {
		for i, r := range name {
			if r == '_' || unicode.IsLetter(r) || i > 0 && unicode.IsNumber(r) {
				continue
			}
			name = name[:i]
			break
		}
	}
	if !utf8.ValidString(name) {
		name = string([]rune(name))
	}
	tagged = name != ""
	return
}
