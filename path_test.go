package jq_test

import (
	"strings"
	"testing"

	"github.com/linkdata/jq"
)

func TestLargeEmptyComponentPath(t *testing.T) {
	runIsolatedTest(t, testLargeEmptyComponentPath)
}

func testLargeEmptyComponentPath(t *testing.T) {
	t.Helper()
	path := strings.Repeat(".", 700_000)
	type leaf struct {
		Value int
	}
	type root struct {
		Leaf  leaf
		Value int
	}

	t.Run("Get", func(t *testing.T) {
		got, err := jq.Get(1, path)
		if err != nil || got != 1 {
			t.Fatalf("Get = (%v, %v), want (1, nil)", got, err)
		}
	})

	t.Run("GetAs before component", func(t *testing.T) {
		value := root{Value: 1}
		got, err := jq.GetAs[int](value, path+"Value")
		if err != nil || got != 1 {
			t.Fatalf("GetAs = (%v, %v), want (1, nil)", got, err)
		}
	})

	t.Run("Set", func(t *testing.T) {
		value := 1
		changed, err := jq.Set(&value, path, 2)
		if err != nil || !changed || value != 2 {
			t.Fatalf("Set = (%t, %v), value = %d; want true, nil, 2", changed, err, value)
		}
	})

	t.Run("Set between components", func(t *testing.T) {
		value := root{Leaf: leaf{Value: 1}}
		changed, err := jq.Set(&value, "Leaf"+path+"Value", 2)
		if err != nil || !changed || value.Leaf.Value != 2 {
			t.Fatalf("Set = (%t, %v), value = %d; want true, nil, 2", changed, err, value.Leaf.Value)
		}
	})

	t.Run("SetChecked after component", func(t *testing.T) {
		value := root{Value: 1}
		calls := 0
		observed := 0
		changed, err := jq.SetChecked(&value, "Value"+path, 2, func() error {
			calls++
			observed = value.Value
			return nil
		})
		if err != nil || !changed || value.Value != 2 || calls != 1 || observed != 2 {
			t.Fatalf("SetChecked = (%t, %v), value/calls/observed = %d/%d/%d; want true, nil, 2/1/2", changed, err, value.Value, calls, observed)
		}
	})
}
