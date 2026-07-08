package builder

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// A hanging build step (e.g. a malicious postinstall) must be killed by the
// context deadline rather than running until Cloud Run's request timeout.
func TestRunBuildCommand_Timeout(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skip("sleep not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := runBuildCommand(ctx, "", nil, "sleep", "30")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected a timeout error, got: %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("command was not killed promptly; ran for %s", elapsed)
	}
}

func TestRunBuildCommand_Success(t *testing.T) {
	if _, err := exec.LookPath("echo"); err != nil {
		t.Skip("echo not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := runBuildCommand(ctx, "", nil, "echo", "hello")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected output to contain %q, got: %q", "hello", out)
	}
}
