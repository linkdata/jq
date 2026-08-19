package jq_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/linkdata/jq"
)

type nilAssignmentChild struct {
	Value int `json:"value"`
}

type nilAssignmentTags map[string]string

type nilAssignmentIntPtr *int

type nilAssignmentError struct {
	message string
}

func (e *nilAssignmentError) Error() string {
	return e.message
}

func TestSetStructTreatsNilInterfaceMapValuesAsNull(t *testing.T) {
	number := 1
	value := struct {
		Scalar  int                `json:"scalar"`
		Pointer *int               `json:"pointer"`
		Any     any                `json:"any"`
		Map     map[string]int     `json:"map"`
		Slice   []int              `json:"slice"`
		Child   nilAssignmentChild `json:"child"`
	}{
		Scalar:  1,
		Pointer: &number,
		Any:     "value",
		Map:     map[string]int{"key": 1},
		Slice:   []int{1},
		Child:   nilAssignmentChild{Value: 1},
	}

	changed, err := jq.Set(&value, "", map[string]any{
		"scalar":  nil,
		"pointer": nil,
		"any":     nil,
		"map":     nil,
		"slice":   nil,
		"child":   nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		t.Fatalf("Set = (%t, %v), value = %#v; want true, nil, zero value", changed, err, value)
	}
}

func TestSetStructNormalizesCompatibleTypedNilMapValues(t *testing.T) {
	t.Run("generic map", func(t *testing.T) {
		type holder struct {
			Scalar  int                `json:"scalar"`
			Wide    int64              `json:"wide"`
			Map     map[string]int     `json:"map"`
			Slice   []int              `json:"slice"`
			Child   nilAssignmentChild `json:"child"`
			Overlay nilAssignmentChild `json:"overlay"`
		}
		value := holder{
			Scalar:  1,
			Wide:    2,
			Map:     map[string]int{"key": 3},
			Slice:   []int{4},
			Child:   nilAssignmentChild{Value: 5},
			Overlay: nilAssignmentChild{Value: 6},
		}
		var scalar *int
		var wide *int8
		var mapping *map[string]int
		var slice *[]int
		var child *nilAssignmentChild
		var overlay *map[string]any

		changed, err := jq.Set(&value, "", map[string]any{
			"scalar":  scalar,
			"wide":    wide,
			"map":     mapping,
			"slice":   slice,
			"child":   child,
			"overlay": overlay,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || !reflect.DeepEqual(value, holder{}) {
			t.Fatalf("Set = (%t, %v), value = %#v; want true, nil, zero value", changed, err, value)
		}
	})

	t.Run("typed map", func(t *testing.T) {
		value := struct {
			Scalar int `json:"scalar"`
		}{Scalar: 1}
		var scalar *int

		changed, err := jq.Set(&value, "", map[string]*int{"scalar": scalar})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value.Scalar != 0 {
			t.Fatalf("Set = (%t, %v), scalar = %d; want true, nil, 0", changed, err, value.Scalar)
		}
	})
}

func TestSetStructCompatibleTypedNilDefinedTypesAreIdempotent(t *testing.T) {
	t.Run("map", func(t *testing.T) {
		value := struct {
			Tags nilAssignmentTags `json:"tags"`
		}{Tags: nilAssignmentTags{"key": "value"}}
		var input *map[string]string

		changed, err := jq.Set(&value, "", map[string]any{"tags": input})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value.Tags != nil {
			t.Fatalf("Set = (%t, %v), tags = %#v; want true, nil, nil", changed, err, value.Tags)
		}

		checks := 0
		changed, err = jq.SetChecked(&value, "", map[string]any{"tags": input}, func() error {
			checks++
			return nil
		})
		if changed || err != nil || checks != 0 || value.Tags != nil {
			t.Fatalf("SetChecked = (%t, %v), checks/tags = %d/%#v; want false, nil, 0/nil", changed, err, checks, value.Tags)
		}
	})

	t.Run("pointer", func(t *testing.T) {
		number := 1
		value := struct {
			Pointer nilAssignmentIntPtr `json:"pointer"`
		}{Pointer: nilAssignmentIntPtr(&number)}
		var input *int

		changed, err := jq.Set(&value, "", map[string]any{"pointer": input})
		if err != nil {
			t.Fatal(err)
		}
		if !changed || value.Pointer != nil {
			t.Fatalf("Set = (%t, %v), pointer = %#v; want true, nil, nil", changed, err, value.Pointer)
		}

		checks := 0
		changed, err = jq.SetChecked(&value, "", map[string]any{"pointer": input}, func() error {
			checks++
			return nil
		})
		if changed || err != nil || checks != 0 || value.Pointer != nil {
			t.Fatalf("SetChecked = (%t, %v), checks/pointer = %d/%#v; want false, nil, 0/nil", changed, err, checks, value.Pointer)
		}
	})
}

func TestSetStructPreservesAssignableTypedNilMapValues(t *testing.T) {
	type holder struct {
		Pointer *int  `json:"pointer"`
		Any     any   `json:"any"`
		Err     error `json:"err"`
	}
	number := 1
	var nilInt *int
	var nilErr *nilAssignmentError
	value := holder{
		Pointer: &number,
		Any:     1,
		Err:     &nilAssignmentError{message: "before"},
	}
	input := map[string]any{
		"pointer": nilInt,
		"any":     nilInt,
		"err":     nilErr,
	}

	changed, err := jq.Set(&value, "", input)
	if err != nil {
		t.Fatal(err)
	}
	storedInt, intOK := value.Any.(*int)
	storedErr, errOK := value.Err.(*nilAssignmentError)
	if !changed || value.Pointer != nil || !intOK || storedInt != nil || !errOK || storedErr != nil {
		t.Fatalf("Set = (%t, %v), value = %#v; want true, nil, typed nil pointers", changed, err, value)
	}

	changed, err = jq.Set(&value, "", input)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("repeated Set reported a write for equal typed nil pointers")
	}
}

func TestSetStructRejectsIncompatibleTypedNilMapValues(t *testing.T) {
	type holder struct {
		Scalar  int                `json:"scalar"`
		Pointer *int               `json:"pointer"`
		Err     error              `json:"err"`
		Map     map[string]int     `json:"map"`
		Slice   []int              `json:"slice"`
		Child   nilAssignmentChild `json:"child"`
	}
	newValue := func() holder {
		number := 2
		return holder{
			Scalar:  1,
			Pointer: &number,
			Err:     &nilAssignmentError{message: "before"},
			Map:     map[string]int{"key": 3},
			Slice:   []int{4},
			Child:   nilAssignmentChild{Value: 5},
		}
	}
	var nilString *string
	var nilInt *int
	var nilSlice *[]int
	var nilMap *map[string]int
	tests := []struct {
		name  string
		field string
		value any
	}{
		{name: "scalar", field: "scalar", value: nilString},
		{name: "pointer", field: "pointer", value: nilString},
		{name: "interface", field: "err", value: nilInt},
		{name: "map", field: "map", value: nilSlice},
		{name: "slice", field: "slice", value: nilMap},
		{name: "struct", field: "child", value: nilString},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := newValue()
			before := value
			changed, err := jq.Set(&value, "", map[string]any{tt.field: tt.value})
			if changed || !errors.Is(err, jq.ErrTypeMismatch) || !reflect.DeepEqual(value, before) {
				t.Fatalf("Set = (%t, %v), value = %#v; want false, ErrTypeMismatch, %#v", changed, err, value, before)
			}
		})
	}

	t.Run("typed map", func(t *testing.T) {
		value := struct {
			Scalar int `json:"scalar"`
		}{Scalar: 1}
		before := value
		changed, err := jq.Set(&value, "", map[string]*string{"scalar": nilString})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) || value != before {
			t.Fatalf("Set = (%t, %v), value = %#v; want false, ErrTypeMismatch, %#v", changed, err, value, before)
		}
	})
}

func TestSetStructTypedNilFailureIsAtomic(t *testing.T) {
	type holder struct {
		Scalar  int  `json:"scalar"`
		Pointer *int `json:"pointer"`
	}
	var clear *int
	var wrong *string
	// Repeat because map iteration order is intentionally unspecified.
	for range 200 {
		number := 2
		value := holder{Scalar: 1, Pointer: &number}
		input := map[string]any{"scalar": clear, "pointer": wrong}
		changed, err := jq.Set(&value, "", input)
		if changed || !errors.Is(err, jq.ErrTypeMismatch) || value.Scalar != 1 || value.Pointer != &number {
			t.Fatalf("Set = (%t, %v), value = %#v; want false, ErrTypeMismatch, unchanged", changed, err, value)
		}

		checks := 0
		changed, err = jq.SetChecked(&value, "", input, func() error {
			checks++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) || checks != 0 || value.Scalar != 1 || value.Pointer != &number {
			t.Fatalf("SetChecked = (%t, %v), checks/value = %d/%#v; want false, ErrTypeMismatch, 0/unchanged", changed, err, checks, value)
		}
	}
}

func TestSetCheckedStructTypedNilMapValue(t *testing.T) {
	type holder struct {
		Err error `json:"err"`
	}
	var nilErr *nilAssignmentError

	t.Run("no-op", func(t *testing.T) {
		value := holder{Err: nilErr}
		checks := 0
		changed, err := jq.SetChecked(&value, "", map[string]any{"err": nilErr}, func() error {
			checks++
			return nil
		})
		stored, ok := value.Err.(*nilAssignmentError)
		if changed || err != nil || checks != 0 || !ok || stored != nil {
			t.Fatalf("SetChecked = (%t, %v), checks/value = %d/%#v; want false, nil, 0/typed nil", changed, err, checks, value.Err)
		}
	})

	t.Run("rejected", func(t *testing.T) {
		before := &nilAssignmentError{message: "before"}
		rejected := errors.New("rejected")
		value := holder{Err: before}
		checks := 0
		observed := false
		changed, err := jq.SetChecked(&value, "", map[string]any{"err": nilErr}, func() error {
			checks++
			stored, ok := value.Err.(*nilAssignmentError)
			observed = ok && stored == nil
			return rejected
		})
		if changed || err != rejected || checks != 1 || !observed || value.Err != before {
			t.Fatalf("SetChecked = (%t, %v), checks/observed/value = %d/%t/%#v; want false, exact rejection, 1/true/previous", changed, err, checks, observed, value.Err)
		}
	})

	t.Run("mismatch", func(t *testing.T) {
		before := &nilAssignmentError{message: "before"}
		value := holder{Err: before}
		var wrong *int
		checks := 0
		changed, err := jq.SetChecked(&value, "", map[string]any{"err": wrong}, func() error {
			checks++
			return nil
		})
		if changed || !errors.Is(err, jq.ErrTypeMismatch) || checks != 0 || value.Err != before {
			t.Fatalf("SetChecked = (%t, %v), checks/value = %d/%#v; want false, ErrTypeMismatch, 0/previous", changed, err, checks, value.Err)
		}
	})
}
