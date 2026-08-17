package jq

import (
	"reflect"
	"testing"
)

func TestResolveStructFields(t *testing.T) {
	type leaf struct {
		Value   int `json:"value"`
		Ignored int `json:"-"`
	}
	type middle struct {
		leaf
	}
	type nested struct {
		middle
		Direct string `json:"direct"`
	}
	type left struct {
		Value int
	}
	type right struct {
		Value int
	}
	type ambiguous struct {
		left
		right
	}

	tests := []struct {
		name string
		typ  reflect.Type
		want map[string][]int
	}{
		{
			name: "nested",
			typ:  reflect.TypeFor[nested](),
			want: map[string][]int{
				"value":  {0, 0, 0},
				"direct": {1},
			},
		},
		{
			name: "ambiguous",
			typ:  reflect.TypeFor[ambiguous](),
			want: map[string][]int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveStructFields(tc.typ); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("resolveStructFields() = %v, want %v", got, tc.want)
			}
		})
	}
}
