package jq_test

import (
	"testing"

	"github.com/linkdata/jq"
)

func TestSetStructAcceptsInterfacePointerMapValue(t *testing.T) {
	t.Run("value field", func(t *testing.T) {
		x := testStructVal
		value := 42
		changed, err := jq.Set(&x, "T", map[string]any{"I": &value})
		maybeError(t, err)
		mustEqual(t, changed, true)
		mustEqual(t, x.T.I, 42)

		var nilValue *int
		changed, err = jq.Set(&x, "T", map[string]any{"I": nilValue})
		maybeError(t, err)
		mustEqual(t, changed, true)
		mustEqual(t, x.T.I, 0)
	})

	t.Run("interface field", func(t *testing.T) {
		type container struct {
			Value any
		}
		var x container
		value := 42
		changed, err := jq.Set(&x, "", map[string]any{"Value": &value})
		maybeError(t, err)
		mustEqual(t, changed, true)
		mustEqual(t, x.Value, 42)

		var nilValue *int
		changed, err = jq.Set(&x, "", map[string]any{"Value": nilValue})
		maybeError(t, err)
		mustEqual(t, changed, true)
		mustEqual(t, x.Value, nil)
	})
}
