// go test ./... -run=^$ -fuzz=FuzzOnDemand_ComprehensivePathTypeMatrix -fuzztime=30s
package jq_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/linkdata/jq"
)

type onDemandSub struct {
	S string
	I int
	F float64
}

type onDemandRoot struct {
	Str     string         `json:"str"`
	Num     int            `json:"num"`
	U8      uint8          `json:"u8"`
	F64     float64        `json:"f64"`
	Bool    bool           `json:"bool"`
	Arr     [2]int         `json:"arr"`
	Slice   []int          `json:"slice"`
	SliceS  []string       `json:"sliceStr"`
	MapInt  map[string]int `json:"mapInt"`
	MapAny  map[string]any `json:"mapAny"`
	Sub     onDemandSub    `json:"sub"`
	PSub    *onDemandSub   `json:"pSub"`
	Any     any            `json:"any"`
	AnyNil  any            `json:"anyNil"`
	Ignored int            `json:"-"`
	hidden  string
}

func onDemandClamp(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func onDemandBytesToInt(raw []byte) int {
	var n int
	for i, b := range raw {
		if i == 8 {
			break
		}
		n = (n << 4) + int(b&0x0f)
	}
	return n
}

func onDemandValue(raw []byte, kind uint8) any {
	if len(raw) > 64 {
		raw = raw[:64]
	}
	n := onDemandBytesToInt(raw)
	if len(raw) == 0 {
		raw = []byte("x")
	}
	s := string(raw)
	sub := onDemandSub{S: s, I: n, F: float64(n) + 0.5}
	switch kind % 18 {
	case 0:
		return nil
	case 1:
		return n%2 == 0
	case 2:
		return n
	case 3:
		return int64(n)
	case 4:
		return uint(n)
	case 5:
		return float64(n) / 10.0
	case 6:
		return s
	case 7:
		return []int{n, n + 1}
	case 8:
		return []string{s, s + "_x"}
	case 9:
		return map[string]any{"S": s, "I": n}
	case 10:
		return map[string]int{"I": n}
	case 11:
		return sub
	case 12:
		return &sub
	case 13:
		return map[string]any{"str": s, "num": n}
	case 14:
		return map[string]any{"pSub": nil}
	case 15:
		return map[string]any{"slice": []int{n, n + 1}}
	case 16:
		return map[string]*onDemandSub{"pSub": nil}
	default:
		return []byte(raw)
	}
}

func newOnDemandRoot(mode uint8) onDemandRoot {
	r := onDemandRoot{
		Str:    "root",
		Num:    7,
		U8:     5,
		F64:    4.2,
		Bool:   true,
		Arr:    [2]int{11, 22},
		Slice:  []int{1, 2},
		SliceS: []string{"a", "b"},
		MapInt: map[string]int{
			"a": 1,
			"b": 2,
		},
		MapAny: map[string]any{
			"s":      "map",
			"n":      3,
			"sub":    map[string]any{"S": "nested", "I": 1},
			"nested": map[string]any{"x": 9},
		},
		Sub:     onDemandSub{S: "sub", I: 2, F: 1.5},
		PSub:    &onDemandSub{S: "ptr", I: 3, F: 2.5},
		Ignored: 99,
		hidden:  "hidden",
	}
	switch mode % 5 {
	case 0:
		r.Any = onDemandSub{S: "iface", I: 4, F: 3.5}
	case 1:
		r.Any = &onDemandSub{S: "ifacePtr", I: 5, F: 4.5}
	case 2:
		r.Any = []int{9}
	case 3:
		r.Any = map[string]any{"k": "v", "n": 6, "nested": map[string]any{"x": 7}}
	default:
		r.Any = nil
	}
	if mode&0x20 != 0 {
		r.PSub = nil
	}
	return r
}

func onDemandPaths() []string {
	base := []string{
		"",
		"str", "num", "u8", "f64", "bool",
		"arr", "arr.0", "arr.1", "arr.2", "arr.-1", "arr.x",
		"slice", "slice.0", "slice.1", "slice.2", "slice.-1", "slice.x",
		"sliceStr", "sliceStr.0", "sliceStr.2",
		"mapInt", "mapInt.a", "mapInt.b", "mapInt.missing", "mapInt.0",
		"mapAny", "mapAny.s", "mapAny.n", "mapAny.sub.S", "mapAny.nested.x", "mapAny.missing",
		"sub", "sub.S", "sub.I", "sub.F", "sub.Missing",
		"pSub", "pSub.S", "pSub.I", "pSub.F", "pSub.Missing",
		"any", "any.S", "any.I", "any.0", "any.1", "any.k", "any.nested.x",
		"anyNil", "anyNil.S",
		"nope", "Nope",
	}
	set := map[string]struct{}{
		".":   {},
		"..":  {},
		"...": {},
	}
	for _, path := range base {
		set[path] = struct{}{}
		if path != "" {
			set["."+path] = struct{}{}
			set[path+"."] = struct{}{}
			if strings.Contains(path, ".") {
				set[strings.ReplaceAll(path, ".", "..")] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for path := range set {
		out = append(out, path)
	}
	return out
}

func onDemandValueKinds() []uint8 {
	kinds := make([]uint8, 18)
	for i := range kinds {
		kinds[i] = uint8(i)
	}
	return kinds
}

func assertOnDemandSetGetCase(t *testing.T, path string, raw []byte, ifaceMode, valueKind uint8) {
	t.Helper()
	path = onDemandClamp(path, 256)
	root := newOnDemandRoot(ifaceMode)
	value := onDemandValue(raw, valueKind)

	changed1, err1 := jq.Set(&root, path, value)
	if err1 != nil {
		if changed1 {
			t.Fatalf("Set changed state on error path=%q valueKind=%d err=%v", path, valueKind, err1)
		}
		if !errors.Is(err1, jq.ErrPathNotFound) && !errors.Is(err1, jq.ErrTypeMismatch) {
			t.Fatalf("unexpected Set error path=%q valueKind=%d err=%v", path, valueKind, err1)
		}
		return
	}

	changed2, err2 := jq.Set(&root, path, value)
	if err2 != nil {
		t.Fatalf("second Set failed path=%q valueKind=%d err=%v", path, valueKind, err2)
	}
	if changed2 {
		t.Fatalf("second Set must be idempotent path=%q valueKind=%d firstChanged=%v", path, valueKind, changed1)
	}

	got, err := jq.Get(&root, path)
	if err != nil {
		t.Fatalf("Get failed after successful Set path=%q valueKind=%d err=%v", path, valueKind, err)
	}

	gotAny, err := jq.GetAs[any](&root, path)
	if got == nil {
		if !errors.Is(err, jq.ErrTypeMismatch) {
			t.Fatalf("GetAs[any] expected type mismatch for nil path=%q valueKind=%d err=%v", path, valueKind, err)
		}
	} else {
		if err != nil {
			t.Fatalf("GetAs[any] failed after successful Set path=%q valueKind=%d err=%v", path, valueKind, err)
		}
		if !reflect.DeepEqual(got, gotAny) {
			t.Fatalf("Get/GetAs mismatch path=%q valueKind=%d got=%#v gotAny=%#v", path, valueKind, got, gotAny)
		}
	}

	gotInt, errInt := jq.GetAs[int](&root, path)
	if expect, ok := got.(int); ok {
		if errInt != nil {
			t.Fatalf("GetAs[int] expected success path=%q valueKind=%d got=%#v err=%v", path, valueKind, got, errInt)
		}
		if gotInt != expect {
			t.Fatalf("GetAs[int] wrong value path=%q valueKind=%d got=%d want=%d", path, valueKind, gotInt, expect)
		}
	} else if !errors.Is(errInt, jq.ErrTypeMismatch) {
		t.Fatalf("GetAs[int] expected type mismatch path=%q valueKind=%d gotType=%T err=%v", path, valueKind, got, errInt)
	}
}

func assertOnDemandGetContract(t *testing.T, path string, ifaceMode uint8) {
	t.Helper()
	path = onDemandClamp(path, 256)
	root := newOnDemandRoot(ifaceMode)

	for _, obj := range []any{root, &root, nil} {
		v, err := jq.Get(obj, path)
		if err != nil {
			if !errors.Is(err, jq.ErrPathNotFound) {
				t.Fatalf("unexpected Get error obj=%T path=%q err=%v", obj, path, err)
			}
			continue
		}
		gotAny, err := jq.GetAs[any](obj, path)
		if v == nil {
			if !errors.Is(err, jq.ErrTypeMismatch) {
				t.Fatalf("GetAs[any] expected type mismatch for nil obj=%T path=%q err=%v", obj, path, err)
			}
		} else {
			if err != nil {
				t.Fatalf("GetAs[any] failed obj=%T path=%q err=%v", obj, path, err)
			}
			if !reflect.DeepEqual(v, gotAny) {
				t.Fatalf("Get/GetAs mismatch obj=%T path=%q got=%#v gotAny=%#v", obj, path, v, gotAny)
			}
		}
	}
}

func assertOnDemandInvalidReceiverContract(t *testing.T, path string, value any) {
	t.Helper()
	path = onDemandClamp(path, 256)
	var nilPtr *onDemandRoot
	var typedNil any = (*onDemandRoot)(nil)
	receivers := []any{
		nil,
		onDemandRoot{},
		17,
		map[string]int{"k": 1},
		nilPtr,
		typedNil,
	}
	for _, receiver := range receivers {
		changed, err := jq.Set(receiver, path, value)
		if changed || !errors.Is(err, jq.ErrInvalidReceiver) {
			t.Fatalf("invalid receiver contract violated receiver=%T path=%q changed=%v err=%v", receiver, path, changed, err)
		}
	}
}

// FuzzOnDemand_ComprehensivePathTypeMatrix runs in seed mode during `go test`.
// Example fuzz run: go test ./... -run=^$ -fuzz=FuzzOnDemand_ComprehensivePathTypeMatrix -fuzztime=30s
func FuzzOnDemand_ComprehensivePathTypeMatrix(f *testing.F) {
	seeds := []struct {
		raw       []byte
		ifaceMode uint8
		path      string
		valueKind uint8
	}{
		{[]byte("seed"), 0, "slice.2", 2},
		{[]byte{0, 1, 2, 3}, 1, "pSub.S", 6},
		{[]byte("map"), 2, "mapAny.sub.S", 6},
		{[]byte("iface"), 3, "any.0", 2},
		{[]byte("nil"), 4, "anyNil.S", 6},
		{[]byte("dots"), 0x20, "...", 1},
	}
	for _, seed := range seeds {
		f.Add(seed.raw, seed.ifaceMode, seed.path, seed.valueKind)
	}

	paths := onDemandPaths()
	valueKinds := onDemandValueKinds()

	f.Fuzz(func(t *testing.T, raw []byte, ifaceMode uint8, extraPath string, extraValueKind uint8) {
		if len(raw) > 256 {
			raw = raw[:256]
		}

		for _, path := range paths {
			assertOnDemandGetContract(t, path, ifaceMode)
			for _, valueKind := range valueKinds {
				assertOnDemandSetGetCase(t, path, raw, ifaceMode, valueKind)
			}
		}

		assertOnDemandGetContract(t, extraPath, ifaceMode)
		assertOnDemandSetGetCase(t, extraPath, raw, ifaceMode, extraValueKind)
		assertOnDemandInvalidReceiverContract(t, extraPath, onDemandValue(raw, extraValueKind))
	})
}
