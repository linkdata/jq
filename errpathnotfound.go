package jq

import "fmt"

// ErrPathNotFound is returned when a JSON path can't be resolved
var ErrPathNotFound errPathNotFound

type errPathNotFound struct {
	index   string
	objtype string
}

func (e errPathNotFound) Error() string {
	return fmt.Sprintf("%q not found in %s", e.index, e.objtype)
}

func (errPathNotFound) Is(other error) bool {
	return other == ErrPathNotFound
}
