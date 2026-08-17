package jq_test

import (
	"errors"
	"testing"

	"github.com/linkdata/jq"
)

type benchmarkGetPair struct {
	First  int
	Second int
}

type benchmarkGetChild struct {
	Value int `json:"value"`
}

type benchmarkGetNested struct {
	Child benchmarkGetChild `json:"child"`
}

type benchmarkGetWide struct {
	First   int
	Second  int
	Third   int
	Fourth  int
	Fifth   int
	Sixth   int
	Seventh int
	Eighth  int
	Ninth   int
	Tenth   int
}

func BenchmarkGet(b *testing.B) {
	b.Run("FirstField", func(b *testing.B) {
		value := benchmarkGetPair{First: 1}
		b.ReportAllocs()
		for b.Loop() {
			got, err := jq.Get(&value, "First")
			if err != nil || got != 1 {
				b.Fatalf("Get = %v, %v; want 1, nil", got, err)
			}
		}
	})

	b.Run("Nested", func(b *testing.B) {
		value := benchmarkGetNested{Child: benchmarkGetChild{Value: 1}}
		b.ReportAllocs()
		for b.Loop() {
			got, err := jq.Get(&value, "child.value")
			if err != nil || got != 1 {
				b.Fatalf("Get = %v, %v; want 1, nil", got, err)
			}
		}
	})

	b.Run("LastField", func(b *testing.B) {
		value := benchmarkGetWide{Tenth: 10}
		b.ReportAllocs()
		for b.Loop() {
			got, err := jq.Get(&value, "Tenth")
			if err != nil || got != 10 {
				b.Fatalf("Get = %v, %v; want 10, nil", got, err)
			}
		}
	})

	b.Run("DeepPointerInterfaces", func(b *testing.B) {
		value := wrapPointerInterfaces(&benchmarkGetChild{Value: 1}, 2048)
		b.ReportAllocs()
		for b.Loop() {
			got, err := jq.Get(value, "value")
			if err != nil || got != 1 {
				b.Fatalf("Get = %v, %v; want 1, nil", got, err)
			}
		}
	})

	b.Run("PointerInterfaceCycle", func(b *testing.B) {
		var value any
		value = &value
		b.ReportAllocs()
		for b.Loop() {
			got, err := jq.Get(value, "missing")
			if got != nil || !errors.Is(err, jq.ErrPathNotFound) {
				b.Fatalf("Get = %v, %v; want nil, ErrPathNotFound", got, err)
			}
		}
	})
}
