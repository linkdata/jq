package jq_test

import (
	"errors"
	"testing"

	"github.com/linkdata/jq"
)

type referenceItem struct {
	Value int
}

type referenceComposite struct {
	values []int
}

func TestSetReplacesDeeplyEqualReferences(t *testing.T) {
	t.Run("pointer", func(t *testing.T) {
		old := &referenceItem{Value: 1}
		value := old
		replacement := &referenceItem{Value: 1}

		changed, err := jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value != replacement {
			t.Fatalf("changed, value = %t, %p; want true, %p", changed, value, replacement)
		}
		replacement.Value = 2
		if value.Value != 2 || old.Value != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", value.Value, old.Value)
		}

		changed, err = jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change when assigning the same pointer")
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
		if !changed {
			t.Fatal("Set reported no change for a distinct map")
		}
		replacement["key"] = 2
		if value["key"] != 2 || old["key"] != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", value["key"], old["key"])
		}

		changed, err = jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change when assigning the same map")
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
		if !changed || cap(value) != cap(replacement) {
			t.Fatalf("changed, cap(value) = %t, %d; want true, %d", changed, cap(value), cap(replacement))
		}
		replacement[0] = 2
		if value[0] != 2 || old[0] != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", value[0], old[0])
		}

		changed, err = jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change when assigning the same slice")
		}
	})

	t.Run("slice capacity", func(t *testing.T) {
		backing := []int{1, 2}
		value := backing[:1:1]
		replacement := backing[:1:2]

		changed, err := jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if !changed || cap(value) != 2 {
			t.Fatalf("changed, cap(value) = %t, %d; want true, 2", changed, cap(value))
		}
		value = append(value, 3)
		if backing[1] != 3 {
			t.Fatalf("backing[1] = %d, want 3", backing[1])
		}
	})

	t.Run("composite", func(t *testing.T) {
		old := referenceComposite{values: []int{1}}
		value := old
		replacement := referenceComposite{values: []int{1}}

		changed, err := jq.Set(&value, "", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if !changed {
			t.Fatal("Set reported no change for a distinct reference nested in a struct")
		}
		replacement.values[0] = 2
		if value.values[0] != 2 || old.values[0] != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", value.values[0], old.values[0])
		}
	})
}

func TestSetReplacesDeeplyEqualMapEntryReferences(t *testing.T) {
	t.Run("pointer", func(t *testing.T) {
		old := &referenceItem{Value: 1}
		replacement := &referenceItem{Value: 1}
		value := map[string]any{"key": old}

		changed, err := jq.Set(&value, "key", replacement)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := value["key"].(*referenceItem)
		if !changed || !ok || got != replacement {
			t.Fatalf("changed, value = %t, %T %v; want true, replacement pointer", changed, value["key"], value["key"])
		}
		replacement.Value = 2
		if got.Value != 2 || old.Value != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", got.Value, old.Value)
		}

		changed, err = jq.Set(&value, "key", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change when assigning the same map-entry pointer")
		}
	})

	t.Run("map", func(t *testing.T) {
		old := map[string]int{"nested": 1}
		replacement := map[string]int{"nested": 1}
		value := map[string]any{"key": old}

		changed, err := jq.Set(&value, "key", replacement)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := value["key"].(map[string]int)
		if !changed || !ok {
			t.Fatalf("changed, value type = %t, %T; want true, map[string]int", changed, value["key"])
		}
		replacement["nested"] = 2
		if got["nested"] != 2 || old["nested"] != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", got["nested"], old["nested"])
		}

		changed, err = jq.Set(&value, "key", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change when assigning the same map-entry map")
		}
	})

	t.Run("slice", func(t *testing.T) {
		old := []int{1}
		replacement := make([]int, 1, 4)
		replacement[0] = 1
		value := map[string]any{"key": old}

		changed, err := jq.Set(&value, "key", replacement)
		if err != nil {
			t.Fatal(err)
		}
		got, ok := value["key"].([]int)
		if !changed || !ok || cap(got) != cap(replacement) {
			t.Fatalf("changed, value = %t, %T with cap %d; want true, []int with cap %d", changed, value["key"], cap(got), cap(replacement))
		}
		replacement[0] = 2
		if got[0] != 2 || old[0] != 1 {
			t.Fatalf("replacement/old values = %d/%d, want 2/1", got[0], old[0])
		}

		changed, err = jq.Set(&value, "key", replacement)
		if err != nil {
			t.Fatal(err)
		}
		if changed {
			t.Fatal("Set reported a change when assigning the same map-entry slice")
		}
	})
}

func TestSetCheckedRejectedReferenceReplacementRestoresIdentity(t *testing.T) {
	wantErr := errors.New("reject reference replacement")

	t.Run("root pointer", func(t *testing.T) {
		old := &referenceItem{Value: 1}
		value := old
		replacement := &referenceItem{Value: 1}
		calls := 0
		observedReplacement := false

		changed, err := jq.SetChecked(&value, "", replacement, func() error {
			calls++
			observedReplacement = value == replacement
			return wantErr
		})
		if changed || err != wantErr {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if calls != 1 || !observedReplacement || value != old {
			t.Fatalf("calls, observed replacement, restored old = %d, %t, %t; want 1, true, true", calls, observedReplacement, value == old)
		}
		replacement.Value = 2
		if value.Value != 1 {
			t.Fatalf("restored value = %d, want 1", value.Value)
		}
	})

	t.Run("map entry", func(t *testing.T) {
		old := &referenceItem{Value: 1}
		replacement := &referenceItem{Value: 1}
		value := map[string]any{"key": old}
		calls := 0
		observedReplacement := false

		changed, err := jq.SetChecked(&value, "key", replacement, func() error {
			calls++
			observedReplacement = value["key"] == replacement
			return wantErr
		})
		if changed || err != wantErr {
			t.Fatalf("changed, err = %t, %v; want false, exact rejection error", changed, err)
		}
		if calls != 1 || !observedReplacement || value["key"] != old {
			t.Fatalf("calls, observed replacement, restored old = %d, %t, %t; want 1, true, true", calls, observedReplacement, value["key"] == old)
		}
		replacement.Value = 2
		if value["key"].(*referenceItem).Value != 1 {
			t.Fatalf("restored value = %d, want 1", value["key"].(*referenceItem).Value)
		}
	})
}
