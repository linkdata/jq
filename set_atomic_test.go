package jq_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/linkdata/jq"
)

func TestSetAppendFailureIsAtomic(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		value any
		want  error
	}{
		{
			name:  "type mismatch",
			path:  "1",
			value: "not an integer",
			want:  jq.ErrTypeMismatch,
		},
		{
			name:  "missing descendant",
			path:  "1.missing",
			value: 2,
			want:  jq.ErrPathNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			backing := []int{1, 99}
			value := backing[:1]
			before := &value[0]

			changed, err := jq.Set(&value, tc.path, tc.value)
			if changed {
				t.Fatal("Set reported a change on error")
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("Set error = %v, want %v", err, tc.want)
			}
			if len(value) != 1 || cap(value) != 2 {
				t.Fatalf("slice len/cap = %d/%d, want 1/2", len(value), cap(value))
			}
			if &value[0] != before {
				t.Fatal("Set replaced the slice backing array on error")
			}
			if backing[0] != 1 || backing[1] != 99 {
				t.Fatalf("backing slice = %v, want [1 99]", backing)
			}
		})
	}
}

func TestSetAppendZeroReportsStructuralChange(t *testing.T) {
	value := []int{1}

	changed, err := jq.Set(&value, "1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("Set reported no change after growing the slice")
	}
	if len(value) != 2 || value[0] != 1 || value[1] != 0 {
		t.Fatalf("value = %v, want [1 0]", value)
	}
}

func TestSetAppendPreparesConvertedValues(t *testing.T) {
	t.Run("number", func(t *testing.T) {
		value := []int{1}

		changed, err := jq.Set(&value, "1", int8(2))
		if err != nil {
			t.Fatal(err)
		}
		if !changed || len(value) != 2 || value[1] != 2 {
			t.Fatalf("changed, value = %t, %v; want true, [1 2]", changed, value)
		}
	})

	t.Run("map to struct", func(t *testing.T) {
		type item struct {
			Value int
		}
		value := []item{{Value: 1}}

		changed, err := jq.Set(&value, "1", map[string]any{"Value": 2})
		if err != nil {
			t.Fatal(err)
		}
		if want := []item{{Value: 1}, {Value: 2}}; !changed || !reflect.DeepEqual(value, want) {
			t.Fatalf("changed, value = %t, %v; want true, %v", changed, value, want)
		}
	})

	t.Run("nested field starts from zero", func(t *testing.T) {
		type item struct {
			Value int
			Other int
		}
		backing := []item{{Value: 1}, {Value: 99, Other: 99}}
		value := backing[:1]

		changed, err := jq.Set(&value, "1.Value", 2)
		if err != nil {
			t.Fatal(err)
		}
		if want := []item{{Value: 1}, {Value: 2}}; !changed || !reflect.DeepEqual(value, want) {
			t.Fatalf("changed, value = %t, %v; want true, %v", changed, value, want)
		}
	})
}

func TestSetUnexportedReferenceFieldReturnsNotFound(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		type state struct {
			values map[string]int
		}
		value := state{values: map[string]int{"key": 1}}

		changed, err := jq.Set(&value, "values.key", 2)
		if changed || !errors.Is(err, jq.ErrPathNotFound) {
			t.Fatalf("Set = (%t, %v), want false, ErrPathNotFound", changed, err)
		}
		if value.values["key"] != 1 {
			t.Fatalf("map value = %d, want 1", value.values["key"])
		}

		calls := 0
		changed, err = jq.SetChecked(&value, "values.key", 2, func() error {
			calls++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrPathNotFound) || calls != 0 {
			t.Fatalf("SetChecked = (%t, %v, %d calls), want false, ErrPathNotFound, 0 calls", changed, err, calls)
		}
		if value.values["key"] != 1 {
			t.Fatalf("map value = %d, want 1", value.values["key"])
		}
	})

	t.Run("pointer", func(t *testing.T) {
		type state struct {
			value *int
		}
		initial := 1
		value := state{value: &initial}

		changed, err := jq.Set(&value, "value", 2)
		if changed || !errors.Is(err, jq.ErrPathNotFound) {
			t.Fatalf("Set = (%t, %v), want false, ErrPathNotFound", changed, err)
		}
		if initial != 1 || value.value != &initial {
			t.Fatal("Set changed the unexported pointer field")
		}

		calls := 0
		changed, err = jq.SetChecked(&value, "value", 2, func() error {
			calls++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrPathNotFound) || calls != 0 {
			t.Fatalf("SetChecked = (%t, %v, %d calls), want false, ErrPathNotFound, 0 calls", changed, err, calls)
		}
		if initial != 1 || value.value != &initial {
			t.Fatal("SetChecked changed the unexported pointer field")
		}
	})
}

func TestSetInterfaceArrayElementReturnsNotFound(t *testing.T) {
	value := struct {
		Any any
	}{Any: [1]int{1}}

	changed, err := jq.Set(&value, "Any.0", 2)
	if changed || !errors.Is(err, jq.ErrPathNotFound) {
		t.Fatalf("Set = (%t, %v), want false, ErrPathNotFound", changed, err)
	}
	if value.Any != [1]int{1} {
		t.Fatalf("value = %v, want [1]", value.Any)
	}
}

func TestSetMapToStructFailureIsAtomic(t *testing.T) {
	typeOf := reflect.StructOf([]reflect.StructField{
		{Name: "Number", Type: reflect.TypeFor[int](), Tag: `json:"number"`},
		{Name: "Text", Type: reflect.TypeFor[string](), Tag: `json:"text"`},
	})
	for range 200 {
		value := reflect.New(typeOf)
		value.Elem().FieldByName("Number").SetInt(1)
		value.Elem().FieldByName("Text").SetString("original")

		changed, err := jq.Set(value.Interface(), "", map[string]any{"number": 2, "text": 2})
		if changed {
			t.Fatal("Set reported a change on error")
		}
		if !errors.Is(err, jq.ErrTypeMismatch) {
			t.Fatalf("Set error = %v, want ErrTypeMismatch", err)
		}
		if number := value.Elem().FieldByName("Number").Int(); number != 1 {
			t.Fatalf("Number = %d, want 1", number)
		}
		if text := value.Elem().FieldByName("Text").String(); text != "original" {
			t.Fatalf("Text = %q, want original", text)
		}
	}
}
