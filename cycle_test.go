package jq_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime/debug"
	"testing"
	"time"

	"github.com/linkdata/jq"
)

const (
	pointerInterfaceCycleEnv = "JQ_TEST_POINTER_INTERFACE_CYCLE"
	deepPointerChainDepth    = 1024 // exceeds the delayed cycle-tracking threshold
)

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
	if os.Getenv(pointerInterfaceCycleEnv) == "1" {
		// Keep a recursive regression from consuming the runtime's full stack allowance.
		debug.SetMaxStack(16 << 20)
		testPointerInterfaceCycle(t)
		return
	}

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPointerInterfaceCycle$") // #nosec G204,G702 -- re-executes the current test binary with constant arguments
	cmd.Env = append(os.Environ(), pointerInterfaceCycleEnv+"=1")
	output, err := cmd.CombinedOutput()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("pointer/interface cycle subprocess did not terminate within 10 seconds\n%s", output)
	}
	if err != nil {
		t.Fatalf("pointer/interface cycle subprocess: %v\n%s", err, output)
	}
	// The subprocess protects the test process from a recursive regression.
	// Repeating the assertions here records coverage only after that preflight succeeds.
	testPointerInterfaceCycle(t)
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
	changed, err = jq.SetChecked(value, "Next.Next.Value", 3, func() error {
		calls++
		if leaf.Value != 3 {
			t.Fatalf("checker observed value %d, want 3", leaf.Value)
		}
		return nil
	})
	if err != nil || !changed || calls != 1 || leaf.Value != 3 {
		t.Fatalf("SetChecked = (%t, %v), calls/value = %d/%d; want true, nil, 1/3", changed, err, calls, leaf.Value)
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
