package jq_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/linkdata/jq"
)

type mapElementDetails struct {
	Number int `json:"number"`
}

type mapElementItem struct {
	*mapElementDetails
	Label string `json:"label"`
}

type mapAssignmentError struct {
	message string
}

func (e *mapAssignmentError) Error() string {
	return e.message
}

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
		stored, ok := x.Value.(*int)
		if !ok || stored != &value {
			t.Fatalf("Value = %#v (%T), want original *int", x.Value, x.Value)
		}

		var nilValue *int
		changed, err = jq.Set(&x, "", map[string]any{"Value": nilValue})
		maybeError(t, err)
		mustEqual(t, changed, true)
		stored, ok = x.Value.(*int)
		if !ok || stored != nil {
			t.Fatalf("Value = %#v (%T), want nil *int", x.Value, x.Value)
		}

		changed, err = jq.Set(&x, "", map[string]any{"Value": nilValue})
		maybeError(t, err)
		mustEqual(t, changed, false)
	})

	t.Run("numeric conversion", func(t *testing.T) {
		type container struct {
			Value int64
		}
		var x container
		value := int8(42)
		changed, err := jq.Set(&x, "", map[string]any{"Value": &value})
		maybeError(t, err)
		mustEqual(t, changed, true)
		mustEqual(t, x.Value, int64(42))
	})

	t.Run("map to struct", func(t *testing.T) {
		type child struct {
			Value int
		}
		type container struct {
			Child child
		}
		var x container
		value := map[string]any{"Value": 42}
		changed, err := jq.Set(&x, "", map[string]any{"Child": &value})
		maybeError(t, err)
		mustEqual(t, changed, true)
		mustEqual(t, x.Child.Value, 42)
	})
}

func TestSetStructPreservesPointerMapValueInInterface(t *testing.T) {
	type item struct {
		Number int `json:"number"`
	}
	type holder struct {
		Value any `json:"value"`
	}

	t.Run("identity and descendants", func(t *testing.T) {
		source := &item{Number: 1}
		var value holder
		changed, err := jq.Set(&value, "", map[string]any{"value": source})
		if err != nil {
			t.Fatal(err)
		}
		stored, ok := value.Value.(*item)
		if !changed || !ok || stored != source {
			t.Fatalf("Set = (%t, %v), Value = %#v (%T); want true, nil, original pointer", changed, err, value.Value, value.Value)
		}

		checks := 0
		replacement := &item{Number: 1}
		changed, err = jq.SetChecked(&value, "", map[string]any{"value": replacement}, func() error {
			checks++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if changed || checks != 0 || value.Value != source {
			t.Fatalf("deeply equal SetChecked = (%t, %v), checks/Value = %d/%#v; want false, nil, 0/original pointer", changed, err, checks, value.Value)
		}

		changed, err = jq.Set(&value, "value.number", 2)
		if err != nil {
			t.Fatal(err)
		}
		if !changed || source.Number != 2 {
			t.Fatalf("descendant Set = (%t, %v), source.Number = %d; want true, nil, 2", changed, err, source.Number)
		}
	})

	t.Run("non-copyable pointee", func(t *testing.T) {
		var source strings.Builder
		if _, err := source.WriteString("hello"); err != nil {
			t.Fatal(err)
		}
		var value holder
		changed, err := jq.Set(&value, "", map[string]any{"value": &source})
		if err != nil {
			t.Fatal(err)
		}
		stored, ok := value.Value.(*strings.Builder)
		if !changed || !ok || stored != &source {
			t.Fatalf("Set = (%t, %v), Value = %T; want true, nil, original *strings.Builder", changed, err, value.Value)
		}
		if _, err := stored.WriteString(" world"); err != nil {
			t.Fatal(err)
		}
		if got := source.String(); got != "hello world" {
			t.Fatalf("source = %q, want %q", got, "hello world")
		}
	})
}

func TestSetStructAcceptsPointerOnlyInterfaceMapValue(t *testing.T) {
	type holder struct {
		Err error `json:"err"`
	}
	input := errors.New("boom")

	t.Run("Set", func(t *testing.T) {
		var value holder
		changed, err := jq.Set(&value, "", map[string]any{"err": input})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value.Err != input {
			t.Fatalf("Set = (%t, %v), error = %v; want true, nil, original error", changed, err, value.Err)
		}
	})

	t.Run("SetChecked accepted", func(t *testing.T) {
		var value holder
		calls := 0
		var observed error
		changed, err := jq.SetChecked(&value, "", map[string]any{"err": input}, func() error {
			calls++
			observed = value.Err
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || calls != 1 || observed != input || value.Err != input {
			t.Fatalf("SetChecked = (%t, %v), calls/error = %d/%v, stored = %v; want true, nil, 1/original, original", changed, err, calls, observed, value.Err)
		}
	})

	t.Run("SetChecked rejected", func(t *testing.T) {
		before := errors.New("before")
		rejected := errors.New("rejected")
		value := holder{Err: before}
		calls := 0
		var observed error
		changed, err := jq.SetChecked(&value, "", map[string]any{"err": input}, func() error {
			calls++
			observed = value.Err
			return rejected
		})
		if changed || err != rejected || calls != 1 || observed != input || value.Err != before {
			t.Fatalf("SetChecked = (%t, %v), calls/error = %d/%v, stored = %v; want false, exact rejection, 1/original, previous", changed, err, calls, observed, value.Err)
		}
	})

	t.Run("pointer to interface type", func(t *testing.T) {
		var value holder
		input := error(errors.New("element"))
		changed, err := jq.Set(&value, "", map[string]any{"err": &input})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) || value.Err != nil {
			t.Fatalf("Set = (%t, %v), error = %v; want false, ErrTypeMismatch, nil", changed, err, value.Err)
		}
	})

	t.Run("mismatch reports pointer type", func(t *testing.T) {
		var value holder
		number := 1
		changed, err := jq.Set(&value, "", map[string]any{"err": &number})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) {
			t.Fatalf("Set = (%t, %v), want false, ErrTypeMismatch", changed, err)
		}
		if want := "jq: expected error, not *int"; err.Error() != want {
			t.Fatalf("error = %q, want %q", err, want)
		}
	})

	t.Run("deeply equal replacement", func(t *testing.T) {
		before := &mapAssignmentError{message: "same"}
		replacement := &mapAssignmentError{message: "same"}
		value := holder{Err: before}
		changed, err := jq.Set(&value, "", map[string]any{"err": replacement})
		if err != nil {
			t.Fatal(err)
		}
		if changed || value.Err != before {
			t.Fatalf("Set = (%t, %v), error = %v; want false, nil, previous error", changed, err, value.Err)
		}
	})
}

func TestSetMapElementAcceptsMapToStruct(t *testing.T) {
	details := &mapElementDetails{Number: 1}
	items := map[string]mapElementItem{
		"key": {mapElementDetails: details, Label: "preserved"},
	}

	input := map[string]any{"number": 2, "label": "updated"}
	changed, err := jq.Set(&items, "key", input)
	if err != nil {
		t.Fatal(err)
	}
	got := items["key"]
	if !changed || got.mapElementDetails != details || got.Number != 2 || got.Label != "updated" {
		t.Fatalf("changed, item = %t, %#v; want true, number 2 and updated label with preserved pointer", changed, got)
	}

	checks := 0
	changed, err = jq.SetChecked(&items, "key", input, func() error {
		checks++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed || checks != 0 || items["key"] != got {
		t.Fatalf("no-op SetChecked = (%t, %v, %d calls), item = %#v; want false, nil, 0 calls, unchanged", changed, err, checks, items["key"])
	}
}

func TestSetMapElementMapToStructNoOps(t *testing.T) {
	var nilMap map[string]any
	tests := []struct {
		name  string
		input any
	}{
		{"empty", map[string]any{}},
		{"nil", nilMap},
		{"unknown", map[string]any{"unknown": 2}},
		{"non-string key", map[int]any{1: 2}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			details := &mapElementDetails{Number: 1}
			initial := mapElementItem{mapElementDetails: details, Label: "original"}
			items := map[string]mapElementItem{"key": initial}
			checks := 0

			changed, err := jq.SetChecked(&items, "key", tc.input, func() error {
				checks++
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if changed || checks != 0 || items["key"] != initial || details.Number != 1 {
				t.Fatalf("SetChecked = (%t, %v, %d calls), item = %#v; want false, nil, 0 calls, unchanged", changed, err, checks, items["key"])
			}
		})
	}
}

func TestSetMapElementConvertsNumber(t *testing.T) {
	items := map[string]int{"key": 1}

	changed, err := jq.Set(&items, "key", int8(2))
	if err != nil {
		t.Fatal(err)
	}
	if !changed || items["key"] != 2 {
		t.Fatalf("Set = (%t, %v), item = %d; want true, nil, 2", changed, err, items["key"])
	}

	changed, err = jq.Set(&items, "key", int8(2))
	if err != nil {
		t.Fatal(err)
	}
	if changed || items["key"] != 2 {
		t.Fatalf("no-op Set = (%t, %v), item = %d; want false, nil, 2", changed, err, items["key"])
	}
}

func TestSetMapElementMapToStructFailureIsAtomic(t *testing.T) {
	for range 200 {
		details := &mapElementDetails{Number: 1}
		initial := mapElementItem{mapElementDetails: details, Label: "original"}
		items := map[string]mapElementItem{"key": initial}
		input := map[string]any{"number": 2, "label": 3}

		changed, err := jq.Set(&items, "key", input)
		if changed || !errors.Is(err, jq.ErrTypeMismatch) {
			t.Fatalf("Set = (%t, %v); want false, ErrTypeMismatch", changed, err)
		}
		if items["key"] != initial || details.Number != 1 {
			t.Fatalf("Set left item = %#v and number = %d; want original item and 1", items["key"], details.Number)
		}

		checks := 0
		changed, err = jq.SetChecked(&items, "key", input, func() error {
			checks++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) || checks != 0 {
			t.Fatalf("SetChecked = (%t, %v, %d calls); want false, ErrTypeMismatch, 0 calls", changed, err, checks)
		}
		if items["key"] != initial || details.Number != 1 {
			t.Fatalf("SetChecked left item = %#v and number = %d; want original item and 1", items["key"], details.Number)
		}
	}
}

func TestSetCheckedMapElementMapToStructRollback(t *testing.T) {
	details := &mapElementDetails{Number: 1}
	initial := mapElementItem{mapElementDetails: details, Label: "original"}
	items := map[string]mapElementItem{"key": initial}
	rejected := errors.New("rejected map element")
	checks := 0
	var observedSameDetails bool
	var observedNumber int
	var observedLabel string

	changed, err := jq.SetChecked(&items, "key", map[string]any{"number": 2, "label": "updated"}, func() error {
		checks++
		item := items["key"]
		observedSameDetails = item.mapElementDetails == details
		observedNumber = item.Number
		observedLabel = item.Label
		return rejected
	})
	if changed || err != rejected || checks != 1 {
		t.Fatalf("SetChecked = (%t, %v, %d calls); want false, exact rejection, 1 call", changed, err, checks)
	}
	if !observedSameDetails || observedNumber != 2 || observedLabel != "updated" {
		t.Fatalf("checker saw same pointer/number/label = %t/%d/%q; want true/2/updated", observedSameDetails, observedNumber, observedLabel)
	}
	if items["key"] != initial || details.Number != 1 {
		t.Fatalf("rollback left item = %#v and number = %d; want original item and 1", items["key"], details.Number)
	}
}
