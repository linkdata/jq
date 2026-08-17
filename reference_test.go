package jq_test

import (
	"testing"

	"github.com/linkdata/jq"
)

type deepEqualItem struct {
	Value int
}

func TestSetDeeplyEqualReferenceReplacementIsNoOp(t *testing.T) {
	t.Run("pointer", func(t *testing.T) {
		old := &deepEqualItem{Value: 1}
		value := old
		replacement := &deepEqualItem{Value: 1}

		changed, err := jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed || value != old {
			t.Fatalf("changed, retained old = %t, %t; want false, true", changed, value == old)
		}
		replacement.Value = 2
		if value.Value != 1 {
			t.Fatalf("value = %d after mutating replacement, want 1", value.Value)
		}
	})

	t.Run("map", func(t *testing.T) {
		old := map[string]int{"key": 1}
		value := old
		replacement := map[string]int{"key": 1}

		changed, err := jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change for a deeply equal map")
		}
		replacement["key"] = 2
		old["retained"] = 3
		if value["key"] != 1 || value["retained"] != 3 {
			t.Fatalf("value = %v, want retained old map", value)
		}
	})

	t.Run("slice", func(t *testing.T) {
		old := []int{1}
		value := old
		replacement := make([]int, 1, 4)
		replacement[0] = 1

		changed, err := jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed || len(value) != len(old) || cap(value) != cap(old) {
			t.Fatalf("changed, len, cap = %t, %d, %d; want false, %d, %d", changed, len(value), cap(value), len(old), cap(old))
		}
		if &value[0] != &old[0] {
			t.Fatal("Set replaced the deeply equal slice backing storage")
		}
		replacement[0] = 2
		if value[0] != 1 {
			t.Fatalf("value[0] = %d after mutating replacement, want 1", value[0])
		}
	})
}

func TestSetCheckedDeeplyEqualMapEntrySkipsCheck(t *testing.T) {
	old := &deepEqualItem{Value: 1}
	replacement := &deepEqualItem{Value: 1}
	value := map[string]any{"key": old}
	checks := 0

	changed, err := jq.SetChecked(&value, "key", replacement, func() error {
		checks++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ok := value["key"].(*deepEqualItem)
	if changed || checks != 0 || !ok || got != old {
		t.Fatalf("changed, checks, retained old = %t, %d, %t; want false, 0, true", changed, checks, ok && got == old)
	}
	replacement.Value = 2
	if got.Value != 1 {
		t.Fatalf("map entry = %d after mutating replacement, want 1", got.Value)
	}
}
