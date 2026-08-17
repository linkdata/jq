package jq

import (
	"reflect"
	"slices"
	"strings"
	"sync"
	"unicode"
)

var structFieldsCache sync.Map

// cachedStructFields returns a shared map of JSON names to field paths.
// Callers must not modify the map or paths.
func cachedStructFields(tp reflect.Type) (fields map[string][]int) {
	if cached, ok := structFieldsCache.Load(tp); ok {
		return cached.(map[string][]int)
	}
	fields = resolveStructFields(tp)
	actual, _ := structFieldsCache.LoadOrStore(tp, fields)
	return actual.(map[string][]int)
}

// resolveStructFields applies encoding/json v1's embedding and dominance rules.
func resolveStructFields(tp reflect.Type) (resolved map[string][]int) {
	type queueEntry struct {
		typ           reflect.Type
		index         []int
		visitChildren bool
	}
	type selection struct {
		index     []int
		tagged    bool
		ambiguous bool
	}

	queue := []queueEntry{{typ: tp, visitChildren: true}}
	seen := map[reflect.Type]bool{tp: true}
	selected := make(map[string]selection)

	// Queue entries stay in breadth-first order, so the first field of a name is
	// shallowest. Scan repeated types for conflicts, but descend only once.
	for i := 0; i < len(queue); i++ {
		parent := queue[i]
		for j := 0; j < parent.typ.NumField(); j++ {
			field := parent.typ.Field(j)
			fieldType := field.Type
			if field.Anonymous {
				underlying := fieldType
				if underlying.Kind() == reflect.Pointer {
					underlying = underlying.Elem()
				}
				if !field.IsExported() && underlying.Kind() != reflect.Struct {
					continue
				}
			} else if !field.IsExported() {
				continue
			}
			if fieldType.Name() == "" && fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}

			name, tagged, ignored := jsonFieldName(field)
			if ignored {
				continue
			}
			index := append(slices.Clone(parent.index), j)
			if tagged || !field.Anonymous || fieldType.Kind() != reflect.Struct {
				current, ok := selected[name]
				switch {
				case !ok:
					selected[name] = selection{index: index, tagged: tagged}
				case len(index) > len(current.index):
					// The shallower field wins even when it is ambiguous.
				case tagged && !current.tagged:
					selected[name] = selection{index: index, tagged: true}
				case tagged == current.tagged:
					current.ambiguous = true
					selected[name] = current
				default:
					// The existing tagged field wins at this depth.
				}
				continue
			}
			if parent.visitChildren {
				queue = append(queue, queueEntry{typ: fieldType, index: index, visitChildren: !seen[fieldType]})
				seen[fieldType] = true
			}
		}
	}

	resolved = make(map[string][]int, len(selected))
	for name, field := range selected {
		if !field.ambiguous {
			resolved[name] = field.index
		}
	}
	return
}

func jsonFieldName(field reflect.StructField) (name string, tagged, ignored bool) {
	tag := field.Tag.Get("json")
	if ignored = tag == "-"; ignored {
		return
	}
	name, _, _ = strings.Cut(tag, ",")
	if !validJSONFieldName(name) {
		name = ""
	}
	tagged = name != ""
	if !tagged {
		name = field.Name
	}
	return
}

func validJSONFieldName(name string) (valid bool) {
	if name == "" {
		return
	}
	for _, r := range name {
		if strings.ContainsRune("!#$%&()*+-./:;<=>?@[]^_{|}~ ", r) {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return
		}
	}
	return true
}

// structFieldValue follows index without allocating pointers and reports whether
// it dereferenced one. A nil pointer returns an invalid field.
func structFieldValue(value reflect.Value, index []int) (field reflect.Value, throughPointer bool) {
	field = value
	for _, i := range index {
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				return reflect.Value{}, false
			}
			field = field.Elem()
			throughPointer = true
		}
		field = field.Field(i)
	}
	return
}
