package jq_test

import (
	"os"
	"os/exec"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/linkdata/jq"
)

const largeEmptyComponentPathEnv = "JQ_TEST_LARGE_EMPTY_COMPONENT_PATH"

func TestLargeEmptyComponentPath(t *testing.T) {
	if os.Getenv(largeEmptyComponentPathEnv) == "1" {
		// Keep a recursive regression from consuming the runtime's full stack allowance.
		debug.SetMaxStack(64 << 20)
		testLargeEmptyComponentPath(t)
		return
	}

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLargeEmptyComponentPath$") // #nosec G204,G702 -- re-executes the current test binary with constant arguments
	cmd.Env = append(os.Environ(), largeEmptyComponentPathEnv+"=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("large empty component path subprocess: %v\n%s", err, output)
	}
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
