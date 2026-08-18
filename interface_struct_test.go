package jq_test

import (
	"errors"
	"testing"

	"github.com/linkdata/jq"
)

type interfaceStructItem struct {
	Value int
}

type interfaceStructNested struct {
	Value int
}

type interfaceStructValue struct {
	Scalar  int
	Pointer *interfaceStructItem
	Map     map[string]int
	Slice   []int
	Array   [1]int
	Nested  interfaceStructNested
}

func newInterfaceStructValue() (root any, pointer *interfaceStructItem, mapped map[string]int, backing []int) {
	pointer = &interfaceStructItem{Value: 1}
	mapped = map[string]int{"key": 1}
	backing = []int{1, 99}
	root = interfaceStructValue{
		Scalar:  1,
		Pointer: pointer,
		Map:     mapped,
		Slice:   backing[:1],
		Array:   [1]int{1},
		Nested:  interfaceStructNested{Value: 1},
	}
	return
}

func requireInterfaceStructValue(t *testing.T, root any, pointer *interfaceStructItem, mapped map[string]int, backing []int) interfaceStructValue {
	t.Helper()

	value, ok := root.(interfaceStructValue)
	if !ok {
		t.Fatalf("root type = %T, want interfaceStructValue", root)
	}
	if value.Pointer != pointer {
		t.Fatal("pointer field identity changed")
	}
	if value.Array != [1]int{1} {
		t.Fatalf("array = %v, want [1]", value.Array)
	}
	if value.Nested.Value != 1 {
		t.Fatalf("nested value = %d, want 1", value.Nested.Value)
	}
	if len(value.Slice) != 1 || cap(value.Slice) != 2 || &value.Slice[0] != &backing[0] {
		t.Fatalf("slice identity/shape = %v len/cap %d/%d, want original backing with len/cap 1/2", value.Slice, len(value.Slice), cap(value.Slice))
	}
	if len(value.Map) != 1 || len(mapped) != 1 || value.Map["key"] != mapped["key"] {
		t.Fatalf("map field/external alias = %v/%v, want one shared key", value.Map, mapped)
	}
	mapped["identity"] = 3
	if value.Map["identity"] != 3 {
		t.Fatal("map field identity changed")
	}
	delete(mapped, "identity")
	return value
}

func TestSetInterfaceStructMutableDescendants(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantPointer int
		wantMap     int
		wantSlice   int
	}{
		{"pointer", "Pointer.Value", 2, 1, 1},
		{"map", "Map.key", 1, 2, 1},
		{"slice", "Slice.0", 1, 1, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, pointer, mapped, backing := newInterfaceStructValue()

			changed, err := jq.Set(&root, tc.path, 2)
			if err != nil {
				t.Fatal(err)
			}
			if !changed {
				t.Fatal("Set reported no change")
			}
			value := requireInterfaceStructValue(t, root, pointer, mapped, backing)
			if value.Scalar != 1 || pointer.Value != tc.wantPointer || mapped["key"] != tc.wantMap || backing[0] != tc.wantSlice {
				t.Fatalf("value = %#v, pointer/map/slice = %d/%d/%d; want scalar 1 and %d/%d/%d", value, pointer.Value, mapped["key"], backing[0], tc.wantPointer, tc.wantMap, tc.wantSlice)
			}
			if backing[1] != 99 {
				t.Fatalf("untargeted slice element = %d, want 99", backing[1])
			}
		})
	}
}

func TestSetInterfaceStructPromotedPointerDescendant(t *testing.T) {
	type Embedded struct {
		Value int
	}
	type holder struct {
		*Embedded
	}

	pointer := &Embedded{Value: 1}
	var root any = holder{Embedded: pointer}
	changed, err := jq.Set(&root, "Value", 2)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := root.(holder)
	if !ok {
		t.Fatalf("root type = %T, want holder", root)
	}
	if !changed || pointer.Value != 2 || value.Embedded != pointer {
		t.Fatalf("Set = (%t, %v), value/pointer = %d/%p; want true, nil, 2/%p", changed, err, pointer.Value, value.Embedded, pointer)
	}
}

func TestSetInterfaceStructFailuresAreAtomic(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		value any
		want  error
		text  string
	}{
		{"direct scalar", "Scalar", 2, jq.ErrPathNotFound, `jq: "Scalar" not found in jq_test.interfaceStructValue`},
		{"direct pointer", "Pointer", interfaceStructItem{Value: 2}, jq.ErrPathNotFound, `jq: "Pointer" not found in jq_test.interfaceStructValue`},
		{"wrong descendant type", "Pointer.Value", "wrong", jq.ErrTypeMismatch, ""},
		{"slice append", "Slice.1", 2, jq.ErrPathNotFound, `jq: "1" not found in []int`},
		{"array element", "Array.0", 2, jq.ErrPathNotFound, `jq: "0" not found in [1]int`},
		{"nested field", "Nested.Value", 2, jq.ErrPathNotFound, `jq: "Value" not found in jq_test.interfaceStructNested`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, pointer, mapped, backing := newInterfaceStructValue()

			changed, err := jq.Set(&root, tc.path, tc.value)
			if changed || !errors.Is(err, tc.want) {
				t.Fatalf("Set = (%t, %v), want false, %v", changed, err, tc.want)
			}
			if tc.text != "" && err.Error() != tc.text {
				t.Fatalf("Set error = %q, want %q", err, tc.text)
			}
			value := requireInterfaceStructValue(t, root, pointer, mapped, backing)
			if value.Scalar != 1 || pointer.Value != 1 || mapped["key"] != 1 || backing[0] != 1 || backing[1] != 99 {
				t.Fatalf("failed Set changed value: %#v, pointer/map/backing = %d/%d/%v", value, pointer.Value, mapped["key"], backing)
			}
		})
	}
}

type interfaceStructAtomicDetails struct {
	Value int
}

type interfaceStructAtomicItem struct {
	*interfaceStructAtomicDetails
	Text string
}

type interfaceStructAtomicRoot struct {
	Items []interfaceStructAtomicItem
}

func TestSetInterfaceStructMapAssignmentFailureIsAtomic(t *testing.T) {
	tests := []struct {
		name string
		set  func(*any, *int) (bool, error)
	}{
		{"Set", func(root *any, _ *int) (bool, error) {
			return jq.Set(root, "Items.0", map[string]any{"Value": 2, "Text": 3})
		}},
		{"SetChecked", func(root *any, calls *int) (bool, error) {
			return jq.SetChecked(root, "Items.0", map[string]any{"Value": 2, "Text": 3}, func() error {
				(*calls)++
				return nil
			})
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Each call builds a fresh source map. Repeat to exercise both the
			// valid-first partial write and invalid-first iteration orders.
			for range 200 {
				details := &interfaceStructAtomicDetails{Value: 1}
				items := []interfaceStructAtomicItem{{interfaceStructAtomicDetails: details, Text: "original"}}
				var root any = interfaceStructAtomicRoot{Items: items}
				calls := 0

				changed, err := tc.set(&root, &calls)
				if changed || !errors.Is(err, jq.ErrTypeMismatch) || calls != 0 {
					t.Fatalf("set = (%t, %v, %d calls), want false, ErrTypeMismatch, 0 calls", changed, err, calls)
				}
				value, ok := root.(interfaceStructAtomicRoot)
				if !ok {
					t.Fatalf("root type = %T, want interfaceStructAtomicRoot", root)
				}
				if len(value.Items) != 1 || &value.Items[0] != &items[0] || value.Items[0].interfaceStructAtomicDetails != details {
					t.Fatal("failed map assignment replaced the slice element or its pointer")
				}
				if details.Value != 1 || items[0].Text != "original" {
					t.Fatalf("failed map assignment left item = %#v, details = %#v", items[0], details)
				}
			}
		})
	}
}

func TestSetCheckedInterfaceStructRollback(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		observe func(*interfaceStructItem, map[string]int, []int) int
	}{
		{"pointer", "Pointer.Value", func(pointer *interfaceStructItem, _ map[string]int, _ []int) int { return pointer.Value }},
		{"map", "Map.key", func(_ *interfaceStructItem, mapped map[string]int, _ []int) int { return mapped["key"] }},
		{"slice", "Slice.0", func(_ *interfaceStructItem, _ map[string]int, backing []int) int { return backing[0] }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, pointer, mapped, backing := newInterfaceStructValue()
			calls := 0
			observed := 0

			changed, err := jq.SetChecked(&root, tc.path, 2, func() error {
				calls++
				observed = tc.observe(pointer, mapped, backing)
				return errCheckRejected
			})
			if changed || err != errCheckRejected || calls != 1 || observed != 2 {
				t.Fatalf("SetChecked = (%t, %v), calls/observed = %d/%d; want false, exact rejection, 1/2", changed, err, calls, observed)
			}
			value := requireInterfaceStructValue(t, root, pointer, mapped, backing)
			if value.Scalar != 1 || pointer.Value != 1 || mapped["key"] != 1 || backing[0] != 1 || backing[1] != 99 {
				t.Fatalf("rollback did not restore value: %#v, pointer/map/backing = %d/%d/%v", value, pointer.Value, mapped["key"], backing)
			}
		})
	}
}

func TestSetCheckedInterfaceStructPanicRollback(t *testing.T) {
	root, pointer, mapped, backing := newInterfaceStructValue()
	panicValue := &struct{}{}
	calls := 0
	observed := 0
	returned := false
	changed := false
	var err error
	var recovered any

	func() {
		defer func() {
			recovered = recover()
		}()
		changed, err = jq.SetChecked(&root, "Map.key", 2, func() error {
			calls++
			observed = mapped["key"]
			panic(panicValue)
		})
		returned = true
	}()
	if returned {
		t.Fatalf("SetChecked = (%t, %v), want panic", changed, err)
	}
	if recovered != panicValue || calls != 1 || observed != 2 {
		t.Fatalf("panic/calls/observed = %v/%d/%d; want exact panic/1/2", recovered, calls, observed)
	}
	value := requireInterfaceStructValue(t, root, pointer, mapped, backing)
	if value.Scalar != 1 || pointer.Value != 1 || mapped["key"] != 1 || backing[0] != 1 || backing[1] != 99 {
		t.Fatalf("panic rollback did not restore value: %#v, pointer/map/backing = %d/%d/%v", value, pointer.Value, mapped["key"], backing)
	}
}
