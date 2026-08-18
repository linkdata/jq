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

type fuzzHeldType struct {
	*fuzzSubType
	M map[string]int
	L []int
}

type fuzzType struct {
	S         string
	I         int
	L         []int
	M         map[string]any
	StructMap map[string]fuzzSubType
	P         *fuzzSubType
	Any       any
	Held      any
	In        fuzzSubType
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
		StructMap: map[string]fuzzSubType{
			"key": {S: "map struct", N: 6},
		},
		P:   &fuzzSubType{S: "ptr", N: 9},
		Any: []int{1},
		Held: fuzzHeldType{
			fuzzSubType: &fuzzSubType{S: "held", N: 10},
			M:           map[string]int{"n": 11},
			L:           []int{12},
		},
		In: fuzzSubType{S: "inner", N: 4},
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
		"StructMap.key.S",
		"P.S",
		"P.Missing",
		"Any.0",
		"Any.1",
		"Held.S",
		"Held.M.n",
		"Held.L.0",
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
		{"L.2", nil, 2},
		{"L.2", []byte("wrong"), 1},
		{"L.2.Missing", []byte{4}, 2},
		{"L.foo", []byte{5}, 2},
		{"M.n", []byte{6}, 2},
		{"M.sub.x", []byte{7}, 2},
		{"StructMap.key", []byte("struct map"), 5},
		{"P.S", []byte("p"), 1},
		{"Any.0", []byte{8}, 2},
		{"Any.1", []byte{9}, 2},
		{"Held.S", []byte("updated"), 1},
		{"Held.M.n", []byte{13}, 2},
		{"Held.L.0", []byte{14}, 2},
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
		before := newFuzzType()

		changed1, err1 := jq.Set(&root, path, value)
		if err1 != nil {
			if changed1 {
				t.Fatalf("Set reported a change on error for path %q value %#v: %v", path, value, err1)
			}
			if !reflect.DeepEqual(root, before) {
				t.Fatalf("Set mutated state on error for path %q value %#v: got %#v want %#v", path, value, root, before)
			}
			if !errors.Is(err1, jq.ErrPathNotFound) && !errors.Is(err1, jq.ErrTypeMismatch) {
				t.Fatalf("unexpected Set error for path %q value %#v: %v", path, value, err1)
			}
			return
		}
		if !changed1 && !reflect.DeepEqual(root, before) {
			t.Fatalf("Set mutated state while reporting no change for path %q value %#v: got %#v want %#v", path, value, root, before)
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

var errFuzzCheckRejected = errors.New("fuzz check rejected")

func FuzzSetChecked_AtomicCallbackContract(f *testing.F) {
	for _, seed := range []struct {
		path     string
		raw      []byte
		mode     uint8
		decision uint8
	}{
		{"S", []byte("changed"), 1, 0},
		{"I", []byte{2}, 2, 1},
		{"L.2", nil, 2, 1},
		{"L.2", []byte("wrong"), 1, 1},
		{"P.S", []byte("panic"), 1, 2},
		{"M.n", []byte{6}, 2, 3},
		{"StructMap.key", []byte("struct map"), 5, 0},
		{"StructMap.key", []byte("rollback"), 5, 1},
		{"Held.S", []byte("accept"), 1, 0},
		{"Held.S", []byte("reject"), 1, 1},
		{"Held.S", []byte("panic"), 1, 2},
		{"Nope", []byte("x"), 1, 0},
	} {
		f.Add(seed.path, seed.raw, seed.mode, seed.decision)
	}

	f.Fuzz(func(t *testing.T, path string, raw []byte, mode, decision uint8) {
		path = clampString(path, 256)
		if len(raw) > 128 {
			raw = raw[:128]
		}
		value := buildFuzzValue(raw, mode)
		before := newFuzzType()
		want := newFuzzType()
		wantChanged, wantErr := jq.Set(&want, path, value)
		root := newFuzzType()
		calls := 0
		var changed bool
		var err error

		switch decision % 4 {
		case 0: // accept
			changed, err = jq.SetChecked(&root, path, value, func() error {
				calls++
				return nil
			})
		case 1: // reject
			changed, err = jq.SetChecked(&root, path, value, func() error {
				calls++
				return errFuzzCheckRejected
			})
		case 2: // panic
			panicValue := &struct{}{}
			panicked := false
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						panicked = true
						if recovered != panicValue {
							t.Fatalf("SetChecked repanicked with %#v, want %#v", recovered, panicValue)
						}
					}
				}()
				changed, err = jq.SetChecked(&root, path, value, func() error {
					calls++
					panic(panicValue)
				})
			}()
			if panicked != wantChanged {
				t.Fatalf("SetChecked panic = %t, want %t for path %q value %#v", panicked, wantChanged, path, value)
			}
		case 3: // nil check
			changed, err = jq.SetChecked(&root, path, value, nil)
		}

		if wantErr != nil {
			wantPath := errors.Is(wantErr, jq.ErrPathNotFound)
			wantType := errors.Is(wantErr, jq.ErrTypeMismatch)
			if err == nil || errors.Is(err, jq.ErrPathNotFound) != wantPath || errors.Is(err, jq.ErrTypeMismatch) != wantType {
				t.Fatalf("SetChecked error = %v, want %v for path %q value %#v", err, wantErr, path, value)
			}
			if changed || calls != 0 || !reflect.DeepEqual(root, before) {
				t.Fatalf("SetChecked setter error contract: changed=%t calls=%d root=%#v want=%#v", changed, calls, root, before)
			}
			return
		}

		switch decision % 4 {
		case 0:
			wantCalls := 0
			if wantChanged {
				wantCalls = 1
			}
			if err != nil || changed != wantChanged || calls != wantCalls || !reflect.DeepEqual(root, want) {
				t.Fatalf("SetChecked accept = (%t, %v, %d calls, %#v), want (%t, nil, %d calls, %#v)", changed, err, calls, root, wantChanged, wantCalls, want)
			}
		case 3:
			if err != nil || changed != wantChanged || calls != 0 || !reflect.DeepEqual(root, want) {
				t.Fatalf("SetChecked nil check = (%t, %v, %d calls, %#v), want (%t, nil, 0 calls, %#v)", changed, err, calls, root, wantChanged, want)
			}
		case 1:
			if wantChanged {
				if err != errFuzzCheckRejected || changed || calls != 1 {
					t.Fatalf("SetChecked rejection = (%t, %v, %d calls), want (false, exact rejection, 1 call)", changed, err, calls)
				}
			} else if err != nil || changed || calls != 0 {
				t.Fatalf("SetChecked no-op rejection = (%t, %v, %d calls), want (false, nil, 0 calls)", changed, err, calls)
			}
			if !reflect.DeepEqual(root, before) {
				t.Fatalf("SetChecked rejection mutated state: got %#v want %#v", root, before)
			}
		case 2:
			wantCalls := 0
			if wantChanged {
				wantCalls = 1
			}
			if err != nil || changed || calls != wantCalls {
				t.Fatalf("SetChecked panic = (%t, %v, %d calls), want (false, nil, %d calls)", changed, err, calls, wantCalls)
			}
			if !reflect.DeepEqual(root, before) {
				t.Fatalf("SetChecked panic mutated state: got %#v want %#v", root, before)
			}
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
