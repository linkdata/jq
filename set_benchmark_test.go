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
}

type benchmarkSetChild struct {
	Value int
}

type benchmarkSetPair struct {
	Left  int
	Right int
}

func BenchmarkSet(b *testing.B) {
	b.Run("Scalar", func(b *testing.B) {
		var value int
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.Set(&value, "", i+1); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NestedPointer", func(b *testing.B) {
		value := benchmarkSetState{Child: &benchmarkSetChild{}}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.Set(&value, "Child.Value", i+1); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ExistingSliceElement", func(b *testing.B) {
		value := benchmarkSetState{Items: []int{0}}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.Set(&value, "Items.0", i+1); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SliceAppend", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			value.Items = value.Items[:0]
			if _, err := jq.Set(&value, "Items.0", i+1); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SliceAppendGrow", func(b *testing.B) {
		var value benchmarkSetState
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			value.Items = nil
			if _, err := jq.Set(&value, "Items.0", i+1); err != nil {
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
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.Set(&value, "Pair", inputs[i&1]); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkSetChecked(b *testing.B) {
	b.Run("NilCheck", func(b *testing.B) {
		var value int
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.SetChecked(&value, "", i+1, nil); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("AcceptScalar", func(b *testing.B) {
		var value int
		check := func() error { return nil }
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.SetChecked(&value, "", i+1, check); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RejectScalar", func(b *testing.B) {
		var value int
		errRejected := errors.New("rejected")
		check := func() error { return errRejected }
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.SetChecked(&value, "", i+1, check); err != errRejected {
				b.Fatalf("SetChecked error = %v, want exact rejection error", err)
			}
		}
	})

	b.Run("AcceptSliceAppend", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		check := func() error { return nil }
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			value.Items = value.Items[:0]
			if _, err := jq.SetChecked(&value, "Items.0", i+1, check); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("RejectSliceAppend", func(b *testing.B) {
		value := benchmarkSetState{Items: make([]int, 0, 1)}
		errRejected := errors.New("rejected")
		check := func() error { return errRejected }
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.SetChecked(&value, "Items.0", i+1, check); err != errRejected {
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
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := jq.SetChecked(&value, "Pair", inputs[i&1], check); err != nil {
				b.Fatal(err)
			}
		}
	})
}
