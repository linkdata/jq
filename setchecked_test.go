package jq_test

import (
	"errors"
	"testing"

	"github.com/linkdata/jq"
)

var errCheckRejected = errors.New("check rejected tentative value")

type checkedItem struct {
	Value int
}

type checkedPair struct {
	Left  int
	Right int
}

type checkedState struct {
	Value      int
	Pointer    *checkedItem
	Items      []int
	Structs    []checkedItem
	Map        map[string]int
	SliceMap   map[string][]int
	StructMap  map[string]checkedItem
	PointerMap map[string]*checkedItem
	Any        any
	Pair       checkedPair
}

func TestSetCheckedCallbackContract(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		value := checkedState{Value: 1}
		calls := 0
		changed, err := jq.SetChecked(&value, "Value", 2, func() error {
			calls++
			if value.Value != 2 {
				t.Fatalf("check saw Value = %d, want 2", value.Value)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value.Value != 2 || calls != 1 {
			t.Fatalf("changed, value, calls = %t, %d, %d; want true, 2, 1", changed, value.Value, calls)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		value := checkedState{Value: 1}
		calls := 0
		changed, err := jq.SetChecked(&value, "Value", 2, func() error {
			calls++
			if value.Value != 2 {
				t.Fatalf("check saw Value = %d, want 2", value.Value)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if value.Value != 1 || calls != 1 {
			t.Fatalf("value, calls = %d, %d; want 1, 1", value.Value, calls)
		}
	})

	t.Run("unchanged", func(t *testing.T) {
		value := checkedState{Value: 1}
		calls := 0
		changed, err := jq.SetChecked(&value, "Value", 1, func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if changed || calls != 0 || value.Value != 1 {
			t.Fatalf("changed, calls, value = %t, %d, %d; want false, 0, 1", changed, calls, value.Value)
		}
	})

	t.Run("setter error", func(t *testing.T) {
		value := checkedState{Value: 1}
		calls := 0
		changed, err := jq.SetChecked(&value, "Value", "wrong", func() error {
			calls++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) {
			t.Fatalf("changed, err = %t, %v; want false, ErrTypeMismatch", changed, err)
		}
		if calls != 0 || value.Value != 1 {
			t.Fatalf("calls, value = %d, %d; want 0, 1", calls, value.Value)
		}
	})

	t.Run("invalid receiver", func(t *testing.T) {
		calls := 0
		changed, err := jq.SetChecked(1, "", 2, func() error {
			calls++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrInvalidReceiver) {
			t.Fatalf("changed, err = %t, %v; want false, ErrInvalidReceiver", changed, err)
		}
		if calls != 0 {
			t.Fatalf("check calls = %d, want 0", calls)
		}
	})

	t.Run("nil check", func(t *testing.T) {
		value := checkedState{Value: 1}
		changed, err := jq.SetChecked(&value, "Value", 2, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value.Value != 2 {
			t.Fatalf("changed, value = %t, %d; want true, 2", changed, value.Value)
		}
	})

	t.Run("rejected numeric conversion", func(t *testing.T) {
		value := checkedState{Value: 1}
		changed, err := jq.SetChecked(&value, "Value", int8(2), func() error {
			if value.Value != 2 {
				t.Fatalf("check saw Value = %d, want 2", value.Value)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if value.Value != 1 {
			t.Fatalf("value = %d, want 1", value.Value)
		}
	})
}

func TestSetCheckedRejectRestoresAliases(t *testing.T) {
	t.Run("pointer", func(t *testing.T) {
		item := &checkedItem{Value: 1}
		value := checkedState{Pointer: item}

		changed, err := jq.SetChecked(&value, "Pointer.Value", 2, func() error {
			if value.Pointer != item || item.Value != 2 {
				t.Fatalf("check saw pointer/value = %p/%d, want %p/2", value.Pointer, item.Value, item)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if value.Pointer != item || item.Value != 1 {
			t.Fatalf("pointer/value = %p/%d, want %p/1", value.Pointer, item.Value, item)
		}
	})

	t.Run("map", func(t *testing.T) {
		alias := map[string]int{"key": 1}
		value := checkedState{Map: alias}

		changed, err := jq.SetChecked(&value, "Map.key", 2, func() error {
			if alias["key"] != 2 {
				t.Fatalf("check saw map value = %d, want 2", alias["key"])
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if alias["key"] != 1 || value.Map["key"] != 1 {
			t.Fatalf("map values = %d/%d, want 1/1", alias["key"], value.Map["key"])
		}
		alias["identity"] = 3
		if value.Map["identity"] != 3 {
			t.Fatal("rollback replaced the map instead of restoring its entry")
		}
	})

	t.Run("existing slice element", func(t *testing.T) {
		backing := []int{1, 99}
		value := checkedState{Items: backing[:1]}
		before := &value.Items[0]

		changed, err := jq.SetChecked(&value, "Items.0", 2, func() error {
			if backing[0] != 2 {
				t.Fatalf("check saw slice value = %d, want 2", backing[0])
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if &value.Items[0] != before || backing[0] != 1 || backing[1] != 99 {
			t.Fatalf("slice was not restored: value=%v backing=%v", value.Items, backing)
		}
	})

	t.Run("append with spare capacity", func(t *testing.T) {
		backing := []int{1, 99}
		value := checkedState{Items: backing[:1]}
		before := &value.Items[0]

		changed, err := jq.SetChecked(&value, "Items.1", 2, func() error {
			if len(value.Items) != 2 || value.Items[1] != 2 {
				t.Fatalf("check saw slice = %v, want [1 2]", value.Items)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if len(value.Items) != 1 || cap(value.Items) != 2 || &value.Items[0] != before {
			t.Fatalf("slice header was not restored: len/cap=%d/%d", len(value.Items), cap(value.Items))
		}
		if backing[0] != 1 || backing[1] != 99 {
			t.Fatalf("backing slice = %v, want [1 99]", backing)
		}
	})

	t.Run("append with allocation", func(t *testing.T) {
		items := []int{1}
		value := checkedState{Items: items}
		before := &value.Items[0]

		changed, err := jq.SetChecked(&value, "Items.1", 2, func() error {
			if len(value.Items) != 2 || value.Items[1] != 2 {
				t.Fatalf("check saw slice = %v, want [1 2]", value.Items)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if len(value.Items) != 1 || cap(value.Items) != 1 || &value.Items[0] != before {
			t.Fatalf("slice header was not restored: len/cap=%d/%d", len(value.Items), cap(value.Items))
		}
	})

	t.Run("append with tail", func(t *testing.T) {
		backing := []checkedItem{{Value: 1}, {Value: 99}}
		value := checkedState{Structs: backing[:1]}
		before := &value.Structs[0]

		changed, err := jq.SetChecked(&value, "Structs.1.Value", 2, func() error {
			if len(value.Structs) != 2 || value.Structs[1].Value != 2 {
				t.Fatalf("check saw slice = %v, want [{1} {2}]", value.Structs)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if len(value.Structs) != 1 || cap(value.Structs) != 2 || &value.Structs[0] != before {
			t.Fatalf("slice header was not restored: len/cap=%d/%d", len(value.Structs), cap(value.Structs))
		}
		if backing[1].Value != 99 {
			t.Fatalf("backing slot = %#v, want {99}", backing[1])
		}
	})

	t.Run("append inside map", func(t *testing.T) {
		backing := []int{1, 99}
		alias := map[string][]int{"key": backing[:1]}
		value := checkedState{SliceMap: alias}

		changed, err := jq.SetChecked(&value, "SliceMap.key.1", 2, func() error {
			if got := alias["key"]; len(got) != 2 || got[1] != 2 {
				t.Fatalf("check saw slice = %v, want [1 2]", got)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if got := alias["key"]; len(got) != 1 || got[0] != 1 || backing[1] != 99 {
			t.Fatalf("map slice was not restored: alias=%v backing=%v", got, backing)
		}
	})

	t.Run("map struct", func(t *testing.T) {
		alias := map[string]checkedItem{"key": {Value: 1}}
		value := checkedState{StructMap: alias}

		changed, err := jq.SetChecked(&value, "StructMap.key.Value", 2, func() error {
			if alias["key"].Value != 2 {
				t.Fatalf("check saw map struct value = %d, want 2", alias["key"].Value)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if alias["key"].Value != 1 || value.StructMap["key"].Value != 1 {
			t.Fatalf("map struct values = %d/%d, want 1/1", alias["key"].Value, value.StructMap["key"].Value)
		}
	})

	t.Run("map pointer", func(t *testing.T) {
		item := &checkedItem{Value: 1}
		alias := map[string]*checkedItem{"key": item}
		value := checkedState{PointerMap: alias}

		changed, err := jq.SetChecked(&value, "PointerMap.key.Value", 2, func() error {
			if item.Value != 2 {
				t.Fatalf("check saw pointed value = %d, want 2", item.Value)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if value.PointerMap["key"] != item || alias["key"] != item || item.Value != 1 {
			t.Fatalf("map pointer was not restored: value=%p alias=%p item=%p/%d", value.PointerMap["key"], alias["key"], item, item.Value)
		}
	})

	t.Run("interface pointer", func(t *testing.T) {
		item := &checkedItem{Value: 1}
		value := checkedState{Any: item}

		changed, err := jq.SetChecked(&value, "Any.Value", 2, func() error {
			if item.Value != 2 {
				t.Fatalf("check saw pointed value = %d, want 2", item.Value)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if value.Any != item || item.Value != 1 {
			t.Fatalf("interface pointer was not restored: value=%p item=%p/%d", value.Any, item, item.Value)
		}
	})

	t.Run("map to struct", func(t *testing.T) {
		value := checkedState{Pair: checkedPair{Left: 1, Right: 2}}

		changed, err := jq.SetChecked(&value, "Pair", map[string]any{"Left": 3, "Right": 4}, func() error {
			if want := (checkedPair{Left: 3, Right: 4}); value.Pair != want {
				t.Fatalf("check saw Pair = %#v, want %#v", value.Pair, want)
			}
			return errCheckRejected
		})
		if changed || err != errCheckRejected {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if want := (checkedPair{Left: 1, Right: 2}); value.Pair != want {
			t.Fatalf("Pair = %#v, want %#v", value.Pair, want)
		}
	})
}

func TestSetCheckedPanicRestoresValue(t *testing.T) {
	panicValue := &struct{ message string }{message: "check panic"}
	backing := []int{1, 99}
	value := checkedState{Items: backing[:1]}

	func() {
		defer func() {
			if recovered := recover(); recovered != panicValue {
				t.Fatalf("recovered = %#v, want exact panic value %#v", recovered, panicValue)
			}
		}()
		changed, err := jq.SetChecked(&value, "Items.1", 2, func() error {
			panic(panicValue)
		})
		t.Fatalf("SetChecked returned (%t, %v), want panic", changed, err)
	}()

	if len(value.Items) != 1 || cap(value.Items) != 2 || backing[0] != 1 || backing[1] != 99 {
		t.Fatalf("value was not restored after panic: value=%v backing=%v", value.Items, backing)
	}
}
