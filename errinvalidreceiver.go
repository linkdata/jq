package jq

import "errors"

// ErrInvalidReceiver indicates that [Set] or [SetChecked] received a nil or
// non-pointer receiver.
var ErrInvalidReceiver = errors.New("jq: invalid receiver")
