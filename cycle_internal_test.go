package jq

import (
	"reflect"
	"testing"
)

func TestPointerCycleDetector(t *testing.T) {
	values := make([]int, 8)
	tests := []struct {
		name       string
		sequence   []int
		detectedAt int
	}{
		{name: "acyclic", sequence: []int{0, 1, 2, 3, 4, 5, 6, 7}, detectedAt: -1},
		{name: "self-cycle", sequence: []int{0, 0}, detectedAt: 1},
		{name: "two-pointer cycle", sequence: []int{0, 1, 0, 1}, detectedAt: 3},
		{name: "tail before cycle", sequence: []int{0, 1, 2, 3, 4, 2, 3}, detectedAt: 6},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var detector pointerCycleDetector
			detectedAt := -1
			for i, index := range tc.sequence {
				if detector.visit(reflect.ValueOf(&values[index])) {
					detectedAt = i
					break
				}
			}
			if detectedAt != tc.detectedAt {
				t.Fatalf("cycle detected at %d, want %d", detectedAt, tc.detectedAt)
			}
		})
	}
}
