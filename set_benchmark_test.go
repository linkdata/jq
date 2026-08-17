package jq_test

import (
	"errors"
	"testing"

	"github.com/linkdata/jq"
)

type benchmarkSetState struct {
	Value int
	Child *benchmarkSetChild
	Items []int
	Pair  benchmarkSetPair
	Pairs []benchmarkSetPair
}

type benchmarkSetChild struct {
	Value int
}

type benchmarkSetPair struct {
	Left  int
	Right int
}

type benchmarkPromotedValue struct {
	Value int `json:"value"`
}

type benchmarkPromotedState struct {
	benchmarkPromotedValue
}

type benchmarkPromotedPointerState struct {
	*benchmarkPromotedValue
}

func BenchmarkSet(b *testing.B) {
	b.Run("Scalar", func(b *testing.B) {
		var value int
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.Set(&value, "", i); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NestedPointer", func(b *testing.B) {
		value := benchmarkSetState{Child: &benchmarkSetChild{}}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.Set(&value, "Child.Value", i); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PromotedField", func(b *testing.B) {
		value := benchmarkPromotedState{}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.Set(&value, "value", i); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ExistingSliceElement", func(b *testing.B) {
		value := benchmarkSetState{Items: []int{0}}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.Set(&value, "Items.0", i); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SliceAppend", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			value.Items = value.Items[:0]
			if _, err := jq.Set(&value, "Items.0", i); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SliceAppendGrow", func(b *testing.B) {
		var value benchmarkSetState
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			value.Items = nil
			if _, err := jq.Set(&value, "Items.0", i); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SliceAppendConvert", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			value.Items = value.Items[:0]
			if _, err := jq.Set(&value, "Items.0", int8(i)); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SliceAppendMapToStruct", func(b *testing.B) {
		value := benchmarkSetState{Pairs: make([]benchmarkSetPair, 0, 1)}
		inputs := [2]map[string]any{
			{"Left": 1, "Right": 2},
			{"Left": 3, "Right": 4},
		}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			value.Pairs = value.Pairs[:0]
			if _, err := jq.Set(&value, "Pairs.0", inputs[i&1]); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("MapToStruct", func(b *testing.B) {
		value := benchmarkSetState{}
		inputs := [2]map[string]any{
			{"Left": 1, "Right": 2},
			{"Left": 3, "Right": 4},
		}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.Set(&value, "Pair", inputs[i&1]); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("PromotedMapToStruct", func(b *testing.B) {
		value := benchmarkPromotedPointerState{benchmarkPromotedValue: &benchmarkPromotedValue{}}
		inputs := [2]map[string]any{{"value": 1}, {"value": 2}}
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.Set(&value, "", inputs[i&1]); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSetChecked(b *testing.B) {
	b.Run("NilCheck", func(b *testing.B) {
		var value int
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.SetChecked(&value, "", i, nil); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AcceptScalar", func(b *testing.B) {
		var value int
		check := func() error { return nil }
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.SetChecked(&value, "", i, check); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RejectScalar", func(b *testing.B) {
		var value int
		errRejected := errors.New("rejected")
		check := func() error { return errRejected }
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.SetChecked(&value, "", i, check); err != errRejected {
				b.Fatalf("SetChecked error = %v, want exact rejection error", err)
			}
		}
	})

	b.Run("AcceptSliceAppend", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		check := func() error { return nil }
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			value.Items = value.Items[:0]
			if _, err := jq.SetChecked(&value, "Items.0", i, check); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RejectSliceAppend", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		errRejected := errors.New("rejected")
		check := func() error { return errRejected }
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.SetChecked(&value, "Items.0", i, check); err != errRejected {
				b.Fatalf("SetChecked error = %v, want exact rejection error", err)
			}
		}
	})

	b.Run("AcceptMapToStruct", func(b *testing.B) {
		value := benchmarkSetState{}
		inputs := [2]map[string]any{
			{"Left": 1, "Right": 2},
			{"Left": 3, "Right": 4},
		}
		check := func() error { return nil }
		i := 0
		b.ReportAllocs()
		for b.Loop() {
			i++
			if _, err := jq.SetChecked(&value, "Pair", inputs[i&1], check); err != nil {
				b.Fatal(err)
			}
		}
	})
}
