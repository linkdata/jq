package jq

import "fmt"

// ErrTypeMismatch is returned when a value does not have the expected type.
var ErrTypeMismatch errTypeMismatch

type errTypeMismatch struct {
	expect any
	actual any
}

func (e errTypeMismatch) Error() string {
	return fmt.Sprintf("jq: expected %T, not %T", e.expect, e.actual)
}

func (errTypeMismatch) Is(other error) bool {
	return other == ErrTypeMismatch
}
