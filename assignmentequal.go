package jq

import "reflect"

// assignmentEqual defines Set's no-write comparison for assignable values.
// Reference identity is part of the comparison; referenced contents are not
// compared.
func assignmentEqual(current, replacement reflect.Value) (equal bool) {
	if current.Kind() == reflect.Interface {
		current = current.Elem()
	}
	if replacement.Kind() == reflect.Interface {
		replacement = replacement.Elem()
	}
	if !current.IsValid() || !replacement.IsValid() {
		equal = current.IsValid() == replacement.IsValid()
		return
	}
	if current.Type() != replacement.Type() {
		return
	}

	switch current.Kind() {
	case reflect.Array:
		equal = true
		for i := range current.Len() {
			if !assignmentEqual(current.Index(i), replacement.Index(i)) {
				equal = false
				break
			}
		}
	case reflect.Struct:
		equal = true
		for i := range current.NumField() {
			if !assignmentEqual(current.Field(i), replacement.Field(i)) {
				equal = false
				break
			}
		}
	case reflect.Map:
		equal = current.UnsafePointer() == replacement.UnsafePointer()
	case reflect.Slice:
		equal = current.UnsafePointer() == replacement.UnsafePointer() &&
			current.Len() == replacement.Len() &&
			current.Cap() == replacement.Cap()
	case reflect.Func:
		equal = current.IsNil() && replacement.IsNil()
	default:
		equal = current.Equal(replacement)
	}
	return
}
