package jq_test

import (
	"errors"
	"reflect"
	"strconv"
	"testing"

	"github.com/linkdata/jq"
)

var invalidArrayIndexComponents = []string{"00", "01", "+1", "-0", "4294967295"}

type arrayIndexHolder struct {
	Value any
}

type arrayIndexTagged struct {
	One          int `json:"1"`
	ZeroZero     int `json:"00"`
	ZeroOne      int `json:"01"`
	PositiveOne  int `json:"+1"`
	NegativeZero int `json:"-0"`
	ReservedMax  int `json:"4294967295"`
}

func requirePathNotFound(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, jq.ErrPathNotFound) {
		t.Fatalf("error = %v, want ErrPathNotFound", err)
	}
}

func requireInvalidArrayWrites(t *testing.T, obj any, path string, snapshot func() any) {
	t.Helper()
	before := snapshot()

	changed, err := jq.Set(obj, path, 99)
	if changed {
		t.Fatal("Set reported a change for an invalid array index")
	}
	requirePathNotFound(t, err)
	if after := snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("Set changed the value: got %#v, want %#v", after, before)
	}

	calls := 0
	changed, err = jq.SetChecked(obj, path, 99, func() error {
		calls++
		return nil
	})
	if changed {
		t.Fatal("SetChecked reported a change for an invalid array index")
	}
	requirePathNotFound(t, err)
	if calls != 0 {
		t.Fatalf("SetChecked called its checker %d times, want 0", calls)
	}
	if after := snapshot(); !reflect.DeepEqual(after, before) {
		t.Fatalf("SetChecked changed the value: got %#v, want %#v", after, before)
	}
}

func TestGetRejectsInvalidArrayIndices(t *testing.T) {
	for _, component := range invalidArrayIndexComponents {
		t.Run(component, func(t *testing.T) {
			array := [2]int{10, 20}
			slice := []int{10, 20}
			arrayHolder := arrayIndexHolder{Value: &array}
			sliceHolder := arrayIndexHolder{Value: &slice}
			tests := []struct {
				name string
				obj  any
				path string
			}{
				{name: "array", obj: array, path: component},
				{name: "slice", obj: slice, path: component},
				{name: "pointer through interface to array", obj: arrayHolder, path: "Value." + component},
				{name: "pointer through interface to slice", obj: sliceHolder, path: "Value." + component},
			}

			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					_, err := jq.Get(tc.obj, tc.path)
					requirePathNotFound(t, err)
				})
			}
		})
	}
}

func TestSetRejectsInvalidArrayIndices(t *testing.T) {
	for _, component := range invalidArrayIndexComponents {
		t.Run(component, func(t *testing.T) {
			t.Run("array", func(t *testing.T) {
				value := [2]int{10, 20}
				requireInvalidArrayWrites(t, &value, component, func() any { return value })
			})

			t.Run("slice", func(t *testing.T) {
				value := []int{10, 20}
				requireInvalidArrayWrites(t, &value, component, func() any {
					return append([]int(nil), value...)
				})
			})

			t.Run("pointer through interface to array", func(t *testing.T) {
				value := [2]int{10, 20}
				holder := arrayIndexHolder{Value: &value}
				requireInvalidArrayWrites(t, &holder, "Value."+component, func() any { return value })
			})

			t.Run("pointer through interface to slice", func(t *testing.T) {
				value := []int{10, 20}
				holder := arrayIndexHolder{Value: &value}
				requireInvalidArrayWrites(t, &holder, "Value."+component, func() any {
					return append([]int(nil), value...)
				})
			})
		})
	}
}

func TestCanonicalArrayIndexOperations(t *testing.T) {
	t.Run("array", func(t *testing.T) {
		value := [2]int{10, 20}
		got, err := jq.Get(&value, "1")
		if err != nil || got != 20 {
			t.Fatalf("Get = (%v, %v), want (20, nil)", got, err)
		}

		changed, err := jq.Set(&value, "1", 21)
		if err != nil || !changed || value != [2]int{10, 21} {
			t.Fatalf("Set = (%t, %v), value = %v", changed, err, value)
		}

		calls := 0
		var checked [2]int
		changed, err = jq.SetChecked(&value, "0", 11, func() error {
			calls++
			checked = value
			return nil
		})
		if want := [2]int{11, 21}; err != nil || !changed || calls != 1 || checked != want || value != want {
			t.Fatalf("SetChecked = (%t, %v, %d calls), checked/value = %v/%v, want %v", changed, err, calls, checked, value, want)
		}
	})

	t.Run("slice", func(t *testing.T) {
		value := []int{10, 20}
		got, err := jq.Get(&value, "1")
		if err != nil || got != 20 {
			t.Fatalf("Get = (%v, %v), want (20, nil)", got, err)
		}

		changed, err := jq.Set(&value, "1", 21)
		if err != nil || !changed || !reflect.DeepEqual(value, []int{10, 21}) {
			t.Fatalf("Set existing element = (%t, %v), value = %v", changed, err, value)
		}
		changed, err = jq.Set(&value, "2", 30)
		if err != nil || !changed || !reflect.DeepEqual(value, []int{10, 21, 30}) {
			t.Fatalf("Set append = (%t, %v), value = %v", changed, err, value)
		}

		calls := 0
		var checked []int
		changed, err = jq.SetChecked(&value, "3", 40, func() error {
			calls++
			checked = append(checked, value...)
			return nil
		})
		want := []int{10, 21, 30, 40}
		if err != nil || !changed || calls != 1 || !reflect.DeepEqual(checked, want) || !reflect.DeepEqual(value, want) {
			t.Fatalf("SetChecked append = (%t, %v, %d calls), checked/value = %v/%v, want %v", changed, err, calls, checked, value, want)
		}
	})
}

func TestArrayIndexBoundary(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("host int cannot represent the JavaScript array-index boundary")
	}
	const component = "4294967295"
	length := uint64(1) << 32

	t.Run("Get", func(t *testing.T) {
		value := make([]struct{}, length)
		if _, err := jq.Get(value, "4294967294"); err != nil {
			t.Fatalf("Get at the maximum array index: %v", err)
		}
		_, err := jq.Get(value, component)
		requirePathNotFound(t, err)
	})

	t.Run("Set", func(t *testing.T) {
		value := make([]struct{}, length-1, length)
		changed, err := jq.Set(&value, component, struct{}{})
		if changed {
			t.Fatal("Set appended at the reserved maximum array index")
		}
		requirePathNotFound(t, err)
		if uint64(len(value)) != length-1 {
			t.Fatalf("slice length = %d, want %d", len(value), length-1)
		}
	})

	t.Run("SetChecked", func(t *testing.T) {
		value := make([]struct{}, length-1, length)
		calls := 0
		changed, err := jq.SetChecked(&value, component, struct{}{}, func() error {
			calls++
			return nil
		})
		if changed {
			t.Fatal("SetChecked appended at the reserved maximum array index")
		}
		requirePathNotFound(t, err)
		if calls != 0 || uint64(len(value)) != length-1 {
			t.Fatalf("SetChecked made %d checker calls and left length %d, want 0 calls and %d", calls, len(value), length-1)
		}
	})
}

func TestCanonicalArrayIndexBounds(t *testing.T) {
	t.Run("array length", func(t *testing.T) {
		value := [2]int{10, 20}
		requireInvalidArrayWrites(t, &value, "2", func() any { return value })
	})

	t.Run("beyond slice length", func(t *testing.T) {
		value := []int{10, 20}
		requireInvalidArrayWrites(t, &value, "3", func() any {
			return append([]int(nil), value...)
		})
	})
}

func TestInvalidNestedArrayIndexDoesNotAppend(t *testing.T) {
	backing := [][2]int{{1, 2}, {7, 8}}
	value := backing[:1]
	before := &value[0]

	requireInvalidArrayWrites(t, &value, "1.01", func() any {
		return struct {
			Value   [][2]int
			Backing [][2]int
		}{
			Value:   append([][2]int(nil), value...),
			Backing: append([][2]int(nil), backing...),
		}
	})
	if len(value) != 1 || cap(value) != 2 || &value[0] != before {
		t.Fatalf("slice header changed: len/cap = %d/%d", len(value), cap(value))
	}
}

func TestArrayIndexSyntaxIsDestinationAware(t *testing.T) {
	tests := []struct {
		component string
		value     int
	}{
		{component: "00", value: 1},
		{component: "01", value: 2},
		{component: "+1", value: 3},
		{component: "-0", value: 4},
		{component: "4294967295", value: 5},
		{component: "1", value: 6},
	}

	t.Run("map keys", func(t *testing.T) {
		for _, tc := range tests {
			t.Run(tc.component, func(t *testing.T) {
				value := map[string]int{tc.component: tc.value}
				got, err := jq.Get(value, tc.component)
				if err != nil || got != tc.value {
					t.Fatalf("Get = (%v, %v), want (%d, nil)", got, err, tc.value)
				}

				changed, err := jq.Set(&value, tc.component, tc.value+10)
				if err != nil || !changed || value[tc.component] != tc.value+10 {
					t.Fatalf("Set = (%t, %v), value = %v", changed, err, value)
				}

				calls := 0
				changed, err = jq.SetChecked(&value, tc.component, tc.value+20, func() error {
					calls++
					return nil
				})
				if err != nil || !changed || calls != 1 || value[tc.component] != tc.value+20 {
					t.Fatalf("SetChecked = (%t, %v, %d calls), value = %v", changed, err, calls, value)
				}
			})
		}
	})

	t.Run("struct tags", func(t *testing.T) {
		for _, tc := range tests {
			t.Run(tc.component, func(t *testing.T) {
				value := arrayIndexTagged{
					One:          6,
					ZeroZero:     1,
					ZeroOne:      2,
					PositiveOne:  3,
					NegativeZero: 4,
					ReservedMax:  5,
				}
				got, err := jq.Get(&value, tc.component)
				if err != nil || got != tc.value {
					t.Fatalf("Get = (%v, %v), want (%d, nil)", got, err, tc.value)
				}

				changed, err := jq.Set(&value, tc.component, tc.value+10)
				if err != nil || !changed {
					t.Fatalf("Set = (%t, %v), want (true, nil)", changed, err)
				}
				got, err = jq.Get(&value, tc.component)
				if err != nil || got != tc.value+10 {
					t.Fatalf("Get after Set = (%v, %v), want (%d, nil)", got, err, tc.value+10)
				}
			})
		}
	})
}

func TestMixedMapAndArrayIndexPath(t *testing.T) {
	type itemList struct {
		Items []int `json:"items"`
	}
	type root struct {
		Map map[string]itemList `json:"map"`
	}

	value := root{Map: map[string]itemList{
		"01": {Items: []int{10, 20}},
	}}
	got, err := jq.Get(&value, "map.01.items.1")
	if err != nil || got != 20 {
		t.Fatalf("Get = (%v, %v), want (20, nil)", got, err)
	}

	changed, err := jq.Set(&value, "map.01.items.1", 21)
	if err != nil || !changed || !reflect.DeepEqual(value.Map["01"].Items, []int{10, 21}) {
		t.Fatalf("Set = (%t, %v), value = %#v", changed, err, value)
	}

	changed, err = jq.Set(&value, "map.01.items.01", 99)
	if changed {
		t.Fatal("Set reported a change for a noncanonical inner index")
	}
	requirePathNotFound(t, err)
	if !reflect.DeepEqual(value.Map["01"].Items, []int{10, 21}) {
		t.Fatalf("failed Set changed value to %#v", value)
	}
}
