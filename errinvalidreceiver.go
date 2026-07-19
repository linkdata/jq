package jq

import "errors"

// ErrInvalidReceiver indicates a nil or non-pointer receiver.
//
// [Set] and [SetChecked] require their first argument to be a non-nil pointer.
var ErrInvalidReceiver = errors.New("jq: invalid receiver")
