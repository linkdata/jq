package jq

import (
	"reflect"
	"testing"
)

func TestAssignmentEqual(t *testing.T) {
	type composite struct {
		Number  int
		Pointer *int
	}

	number := 1
	equalNumber := 1
	otherNumber := 2
	sharedMap := map[string]int{"key": 1}
	distinctMap := map[string]int{"key": 1}
	sharedSlice := []int{1, 2}
	distinctSlice := []int{1, 2}
	fn := func() {}
	var nilInterface any
	var numberInterface any = number
	var nilFunc func()

	tests := []struct {
		name        string
		current     reflect.Value
		replacement reflect.Value
		want        bool
	}{
		{"invalid", reflect.Value{}, reflect.Value{}, true},
		{"one invalid", reflect.ValueOf(&nilInterface).Elem(), reflect.ValueOf(number), false},
		{"different types", reflect.ValueOf(number), reflect.ValueOf("1"), false},
		{"interface", reflect.ValueOf(&numberInterface).Elem(), reflect.ValueOf(number), true},
		{"equal scalar", reflect.ValueOf(number), reflect.ValueOf(equalNumber), true},
		{"different scalar", reflect.ValueOf(number), reflect.ValueOf(otherNumber), false},
		{"same pointer", reflect.ValueOf(&number), reflect.ValueOf(&number), true},
		{"distinct pointer", reflect.ValueOf(&number), reflect.ValueOf(&equalNumber), false},
		{"equal array", reflect.ValueOf([2]*int{&number, &number}), reflect.ValueOf([2]*int{&number, &number}), true},
		{"different array", reflect.ValueOf([2]*int{&number, &number}), reflect.ValueOf([2]*int{&number, &equalNumber}), false},
		{"equal struct", reflect.ValueOf(composite{Number: 1, Pointer: &number}), reflect.ValueOf(composite{Number: 1, Pointer: &number}), true},
		{"different struct", reflect.ValueOf(composite{Number: 1, Pointer: &number}), reflect.ValueOf(composite{Number: 1, Pointer: &equalNumber}), false},
		{"same map", reflect.ValueOf(sharedMap), reflect.ValueOf(sharedMap), true},
		{"distinct map", reflect.ValueOf(sharedMap), reflect.ValueOf(distinctMap), false},
		{"same slice", reflect.ValueOf(sharedSlice), reflect.ValueOf(sharedSlice), true},
		{"distinct slice", reflect.ValueOf(sharedSlice), reflect.ValueOf(distinctSlice), false},
		{"different slice length", reflect.ValueOf(sharedSlice[:1]), reflect.ValueOf(sharedSlice[:2]), false},
		{"different slice capacity", reflect.ValueOf(sharedSlice[:1:1]), reflect.ValueOf(sharedSlice[:1:2]), false},
		{"nil slice and empty slice", reflect.ValueOf([]int(nil)), reflect.ValueOf([]int{}), false},
		{"nil functions", reflect.ValueOf(nilFunc), reflect.ValueOf(nilFunc), true},
		{"non-nil function", reflect.ValueOf(fn), reflect.ValueOf(fn), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := assignmentEqual(tc.current, tc.replacement); got != tc.want {
				t.Fatalf("assignmentEqual() = %t, want %t", got, tc.want)
			}
			if got := assignmentEqual(tc.replacement, tc.current); got != tc.want {
				t.Fatalf("assignmentEqual() in reverse = %t, want %t", got, tc.want)
			}
		})
	}
}
