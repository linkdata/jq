package jq

import (
	"fmt"
	"reflect"
)

// ErrTypeMismatch is returned when a value does not have the expected type.
var ErrTypeMismatch errTypeMismatch

type errTypeMismatch struct {
	expect reflect.Type
	actual reflect.Type
}

func (e errTypeMismatch) Error() string {
	return fmt.Sprintf("jq: expected %s, not %s", e.expect, e.actual)
}

func (errTypeMismatch) Is(other error) bool {
	return other == ErrTypeMismatch
}
