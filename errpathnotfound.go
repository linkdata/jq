package jq

import "fmt"

// ErrPathNotFound is returned when a JSON path can't be resolved
var ErrPathNotFound errPathNotFound

type errPathNotFound struct {
	index string
	obj   any
}

func (e errPathNotFound) Error() string {
	return fmt.Sprintf("%q not found in %T", e.index, e.obj)
}

func (errPathNotFound) Is(other error) bool {
	return other == ErrPathNotFound
}
