package jq

import "errors"

// ErrInvalidReceiver is returned when [Set] is called for an invalid pointer.
// (The first argument to [Set] must be a non-nil pointer.)
var ErrInvalidReceiver = errors.New("jq: invalid receiver")
