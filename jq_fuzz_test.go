package jq_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/linkdata/jq"
)

type fuzzSubType struct {
	S string
	N int
}

type fuzzType struct {
	S   string
	I   int
	L   []int
	M   map[string]any
	P   *fuzzSubType
	Any any
	In  fuzzSubType
}

func newFuzzType() fuzzType {
	return fuzzType{
		S: "root",
		I: 7,
		L: []int{1, 2},
		M: map[string]any{
			"s":   "map",
			"n":   3,
			"sub": map[string]any{"x": 5},
		},
		P:   &fuzzSubType{S: "ptr", N: 9},
		Any: []int{1},
		In:  fuzzSubType{S: "inner", N: 4},
	}
}

func clampString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func bytesToInt(raw []byte) int {
	var n int
	for i, b := range raw {
		if i == 8 {
			break
		}
		n = (n << 4) + int(b&0x0f)
	}
	return n
}

func buildFuzzValue(raw []byte, mode uint8) any {
	if len(raw) > 64 {
		raw = raw[:64]
	}
	n := bytesToInt(raw)
	if len(raw) == 0 {
		raw = []byte("x")
	}
	switch mode % 7 {
	case 0:
		return nil
	case 1:
		return string(raw)
	case 2:
		return n
	case 3:
		return float64(n) / 10.0
	case 4:
		return []int{len(raw), int(raw[0])}
	case 5:
		return map[string]any{"S": string(raw), "N": n}
	default:
		return &fuzzSubType{S: string(raw), N: n}
	}
}

func FuzzGet_NoPanicAndErrorContract(f *testing.F) {
	for _, seed := range []string{
		"",
		".",
		"...",
		"S",
		".S",
		"In.S",
		"L.0",
		"L.2",
		"L.foo",
		"M.s",
		"M.sub.x",
		"P.S",
		"P.Missing",
		"Any.0",
		"Any.1",
		"Nope",
		"M..sub",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, path string) {
		path = clampString(path, 256)
		root := newFuzzType()

		for _, obj := range []any{root, &root} {
			v, err := jq.Get(obj, path)
			if err != nil {
				if !errors.Is(err, jq.ErrPathNotFound) {
					t.Fatalf("unexpected Get error for %T path %q: %v", obj, path, err)
				}
				continue
			}
			gotAny, err := jq.GetAs[any](obj, path)
			if err != nil {
				t.Fatalf("GetAs[any] failed after successful Get for %T path %q: %v", obj, path, err)
			}
			if !reflect.DeepEqual(v, gotAny) {
				t.Fatalf("Get/GetAs mismatch for %T path %q: %#v vs %#v", obj, path, v, gotAny)
			}
		}
	})
}

func FuzzSet_NoPanicIdempotent(f *testing.F) {
	for _, seed := range []struct {
		path string
		raw  []byte
		mode uint8
	}{
		{"", []byte("root"), 1},
		{"S", []byte("x"), 1},
		{".S", []byte("dot"), 1},
		{"I", []byte{2}, 2},
		{"L.0", []byte{3}, 2},
		{"L.2", []byte{4}, 2},
		{"L.foo", []byte{5}, 2},
		{"M.n", []byte{6}, 2},
		{"M.sub.x", []byte{7}, 2},
		{"P.S", []byte("p"), 1},
		{"Any.0", []byte{8}, 2},
		{"Any.1", []byte{9}, 2},
		{"In.N", []byte{10}, 2},
		{"Nope", []byte("x"), 1},
		{"...", []byte("x"), 1},
	} {
		f.Add(seed.path, seed.raw, seed.mode)
	}

	f.Fuzz(func(t *testing.T, path string, raw []byte, mode uint8) {
		path = clampString(path, 256)
		if len(raw) > 128 {
			raw = raw[:128]
		}
		value := buildFuzzValue(raw, mode)
		root := newFuzzType()

		changed1, err1 := jq.Set(&root, path, value)
		if err1 != nil {
			if !errors.Is(err1, jq.ErrPathNotFound) && !errors.Is(err1, jq.ErrTypeMismatch) {
				t.Fatalf("unexpected Set error for path %q value %#v: %v", path, value, err1)
			}
			return
		}

		changed2, err2 := jq.Set(&root, path, value)
		if err2 != nil {
			t.Fatalf("second Set failed after success for path %q value %#v: %v", path, value, err2)
		}
		if changed2 {
			t.Fatalf("second Set changed state for path %q value %#v (first changed=%v)", path, value, changed1)
		}

		if _, err := jq.Get(&root, path); err != nil {
			t.Fatalf("Get failed after successful Set for path %q value %#v: %v", path, value, err)
		}
	})
}

func FuzzSet_InvalidReceiverContract(f *testing.F) {
	for _, seed := range []struct {
		path string
		v    int
	}{
		{"", 0},
		{"S", 1},
		{"Any.0", 2},
		{"...", 3},
	} {
		f.Add(seed.path, seed.v)
	}

	f.Fuzz(func(t *testing.T, path string, v int) {
		path = clampString(path, 256)

		var nilRoot *fuzzType
		changed, err := jq.Set(nilRoot, path, v)
		if changed || !errors.Is(err, jq.ErrInvalidReceiver) {
			t.Fatalf("nil pointer receiver contract violated for path %q: changed=%v err=%v", path, changed, err)
		}

		root := newFuzzType()
		changed, err = jq.Set(root, path, v)
		if changed || !errors.Is(err, jq.ErrInvalidReceiver) {
			t.Fatalf("non-pointer receiver contract violated for path %q: changed=%v err=%v", path, changed, err)
		}

		var iface any = (*fuzzType)(nil)
		changed, err = jq.Set(iface, path, v)
		if changed || !errors.Is(err, jq.ErrInvalidReceiver) {
			t.Fatalf("typed nil receiver contract violated for path %q: changed=%v err=%v", path, changed, err)
		}
	})
}
