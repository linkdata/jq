package jq_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"testing"
	"time"
)

const (
	isolatedTestEnv       = "JQ_TEST_ISOLATED"
	isolatedTestMaxStack  = 16 << 20
	isolatedChildTimeout  = 8 * time.Second
	isolatedParentTimeout = 10 * time.Second
)

func runIsolatedTest(t *testing.T, run func(*testing.T)) {
	t.Helper()
	if os.Getenv(isolatedTestEnv) == t.Name() {
		debug.SetMaxStack(isolatedTestMaxStack)
		run(t)
		return
	}

	ctx, cancel := context.WithTimeout(t.Context(), isolatedParentTimeout)
	defer cancel()
	filter := "^" + regexp.QuoteMeta(t.Name()) + "$"
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run="+filter, "-test.timeout="+isolatedChildTimeout.String()) // #nosec G204,G702 -- re-executes the current test binary with internally supplied arguments
	cmd.Env = append(os.Environ(), isolatedTestEnv+"="+t.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("isolated subprocess did not terminate within %s: %v\n%s", isolatedParentTimeout, err, output)
		}
		t.Fatalf("isolated subprocess: %v\n%s", err, output)
	}

	// A recursive regression can terminate the subprocess. Replay only after its
	// successful preflight so the parent test records coverage safely.
	run(t)
}
