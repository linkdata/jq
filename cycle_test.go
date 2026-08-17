package jq_test

import (
	"testing"

	"github.com/linkdata/jq"
)

const deepPointerChainDepth = 1024

type cyclicNode struct {
	Next  any
	Value int
}

func newSelfPointerInterfaceCycle() any {
	var value any
	value = &value
	return value
}

func newTwoNodePointerInterfaceCycle() any {
	var first any
	var second any
	first = &second
	second = &first
	return first
}

func wrapPointerInterfaces(value any, depth int) any {
	for range depth {
		next := value
		value = &next
	}
	return value
}

func requireSelfPointerInterfaceCycle(t *testing.T, value any) {
	t.Helper()
	pointer, ok := value.(*any)
	if !ok {
		t.Fatalf("cycle root has type %T, want *any", value)
	}
	next, ok := (*pointer).(*any)
	if !ok || next != pointer {
		t.Fatal("pointer/interface self-cycle was changed")
	}
}

func TestPointerInterfaceCycle(t *testing.T) {
	runIsolatedTest(t, testPointerInterfaceCycle)
}

func testPointerInterfaceCycle(t *testing.T) {
	t.Helper()

	t.Run("Get self-cycle", func(t *testing.T) {
		value := newSelfPointerInterfaceCycle()
		got, err := jq.Get(value, "missing")
		if got != nil {
			t.Fatalf("Get value = %#v, want nil", got)
		}
		requirePathNotFound(t, err)
		requireSelfPointerInterfaceCycle(t, value)
	})

	t.Run("Get two-node cycle", func(t *testing.T) {
		got, err := jq.Get(newTwoNodePointerInterfaceCycle(), "missing")
		if got != nil {
			t.Fatalf("Get value = %#v, want nil", got)
		}
		requirePathNotFound(t, err)
	})

	t.Run("GetAs", func(t *testing.T) {
		got, err := jq.GetAs[int](newSelfPointerInterfaceCycle(), "missing")
		if got != 0 {
			t.Fatalf("GetAs value = %d, want 0", got)
		}
		requirePathNotFound(t, err)
	})

	t.Run("Set", func(t *testing.T) {
		value := newSelfPointerInterfaceCycle()
		changed, err := jq.Set(value, "missing", 1)
		if changed {
			t.Fatal("Set reported a change")
		}
		requirePathNotFound(t, err)
		requireSelfPointerInterfaceCycle(t, value)
	})

	t.Run("SetChecked", func(t *testing.T) {
		value := newSelfPointerInterfaceCycle()
		calls := 0
		changed, err := jq.SetChecked(value, "missing", 1, func() error {
			calls++
			return nil
		})
		if changed || calls != 0 {
			t.Fatalf("SetChecked = %t with %d checker calls, want false with 0 calls", changed, calls)
		}
		requirePathNotFound(t, err)
		requireSelfPointerInterfaceCycle(t, value)
	})
}

func TestPointerInterfaceCycleEmptyPath(t *testing.T) {
	value := newSelfPointerInterfaceCycle()
	pointer := value.(*any)
	got, err := jq.Get(value, "")
	if err != nil || got != value {
		t.Fatalf("Get = (%#v, %v), want cycle root, nil", got, err)
	}

	changed, err := jq.Set(value, "", 1)
	if err != nil || !changed || *pointer != 1 {
		t.Fatalf("Set = (%t, %v), pointed value = %#v; want true, nil, 1", changed, err, *pointer)
	}
}

func TestPointerRevisitAfterPathProgress(t *testing.T) {
	leaf := &cyclicNode{Value: 1}
	value := wrapPointerInterfaces(leaf, deepPointerChainDepth)
	leaf.Next = value

	got, err := jq.Get(value, "Next.Next.Value")
	if err != nil || got != 1 {
		t.Fatalf("Get = (%#v, %v), want 1, nil", got, err)
	}

	changed, err := jq.Set(value, "Next.Next.Value", 2)
	if err != nil || !changed || leaf.Value != 2 {
		t.Fatalf("Set = (%t, %v), value = %d; want true, nil, 2", changed, err, leaf.Value)
	}

	calls := 0
	observed := 0
	changed, err = jq.SetChecked(value, "Next.Next.Value", 3, func() error {
		calls++
		observed = leaf.Value
		return nil
	})
	if err != nil || !changed || calls != 1 || observed != 3 || leaf.Value != 3 {
		t.Fatalf("SetChecked = (%t, %v), calls/observed/value = %d/%d/%d; want true, nil, 1/3/3", changed, err, calls, observed, leaf.Value)
	}
}

func TestAcyclicInterfacePointerChain(t *testing.T) {
	leaf := &cyclicNode{Value: 1}
	value := wrapPointerInterfaces(leaf, deepPointerChainDepth)

	got, err := jq.Get(value, "Value")
	if err != nil || got != 1 {
		t.Fatalf("Get = (%#v, %v), want 1, nil", got, err)
	}

	changed, err := jq.Set(value, "Value", 2)
	if err != nil || !changed || leaf.Value != 2 {
		t.Fatalf("Set = (%t, %v), value = %d; want true, nil, 2", changed, err, leaf.Value)
	}
}
