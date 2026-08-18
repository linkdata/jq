package jq_test

import (
	"errors"
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
