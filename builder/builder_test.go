package builder

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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

// InstallDeps must not run the submission's postinstall script (the primary
// Node-side RCE vector). A postinstall that writes a marker file must not run.
func TestInstallDeps_IgnoresScripts(t *testing.T) {
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skip("pnpm not available")
	}

	dir := t.TempDir()
	pkg := `{
  "name": "malicious-submission",
  "version": "1.0.0",
  "scripts": {
    "postinstall": "node -e \"require('fs').writeFileSync('PWNED','1')\""
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	out, err := InstallDeps(dir)
	if err != nil {
		t.Fatalf("InstallDeps failed: %v\noutput:\n%s", err, out)
	}

	if _, err := os.Stat(filepath.Join(dir, "PWNED")); err == nil {
		t.Fatal("postinstall script executed despite --ignore-scripts (PWNED marker was created)")
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking marker: %v", err)
	}
}

// overlayDir must copy trusted config over the submission's own files.
func TestOverlayDir(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	// Trusted config to overlay, including a nested file.
	if err := os.WriteFile(filepath.Join(src, "vite.config.ts"), []byte("clean"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(src, "cfg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "cfg", "extra.ts"), []byte("nested"), 0644); err != nil {
		t.Fatal(err)
	}

	// Submission's own (untrusted) config that must be overwritten.
	if err := os.WriteFile(filepath.Join(dst, "vite.config.ts"), []byte("malicious"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := overlayDir(src, dst); err != nil {
		t.Fatalf("overlayDir failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "vite.config.ts"))
	if err != nil || string(got) != "clean" {
		t.Errorf("vite.config.ts = %q (err %v), want %q", got, err, "clean")
	}
	nested, err := os.ReadFile(filepath.Join(dst, "cfg", "extra.ts"))
	if err != nil || string(nested) != "nested" {
		t.Errorf("cfg/extra.ts = %q (err %v), want %q", nested, err, "nested")
	}
}
