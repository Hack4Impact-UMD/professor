package sandbox

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestBuildDockerStartArgs(t *testing.T) {
	tests := []struct {
		name     string
		sb       DockerSandbox
		cwd      string // host cwd; defaults to sb.Dir
		wantRest []string
		wantEnv  []string
	}{
		{
			name: "minimal has no network",
			sb:   DockerSandbox{Name: "sb-1", Image: "img", Dir: "/workspace"},
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/workspace,dst=/workspace,readonly",
				"--network", "none",
				"img", "sleep", "infinity",
			},
		},
		{
			name: "network access",
			sb:   DockerSandbox{Name: "sb-1", Image: "img", Dir: "/workspace", Net: true},
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/workspace,dst=/workspace,readonly",
				"img", "sleep", "infinity",
			},
		},
		{
			name: "caller mount at dir replaces the read-only dir mount",
			sb: DockerSandbox{
				Name:   "sb-1",
				Image:  "img",
				Dir:    "/workspace",
				Mounts: []MountPoint{{SourcePath: "/elsewhere", DestPath: "/workspace"}},
			},
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/elsewhere,dst=/workspace",
				"--network", "none",
				"img", "sleep", "infinity",
			},
		},
		{
			name: "mounts and env",
			sb: DockerSandbox{
				Name:  "sb-1",
				Image: "img",
				Dir:   "/workspace",
				Mounts: []MountPoint{
					{SourcePath: "/a", DestPath: "/workspace/a"},
					{SourcePath: "/b", DestPath: "/workspace/b", ReadOnly: true},
				},
				Env: map[string]string{"PATH": "/usr/bin:/bin", "CI": "1"},
			},
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/workspace,dst=/workspace,readonly",
				"--mount", "type=bind,src=/a,dst=/workspace/a",
				"--mount", "type=bind,src=/b,dst=/workspace/b,readonly",
				"--network", "none",
				"img", "sleep", "infinity",
			},
			wantEnv: []string{"CI=1", "PATH=/usr/bin:/bin"},
		},
		{
			name: "host cwd outside dir is mounted read-only",
			sb:   DockerSandbox{Name: "sb-1", Image: "img", Dir: "/workspace"},
			cwd:  "/host/cwd",
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/workspace,dst=/workspace,readonly",
				"--mount", "type=bind,src=/host/cwd,dst=/host/cwd,readonly",
				"--network", "none",
				"img", "sleep", "infinity",
			},
		},
		{
			name: "host cwd of / is not mounted",
			sb:   DockerSandbox{Name: "sb-1", Image: "img", Dir: "/workspace"},
			cwd:  "/",
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/workspace,dst=/workspace,readonly",
				"--network", "none",
				"img", "sleep", "infinity",
			},
		},
		{
			name: "caller mount at host cwd replaces the cwd mount",
			sb: DockerSandbox{
				Name:   "sb-1",
				Image:  "img",
				Dir:    "/workspace",
				Mounts: []MountPoint{{SourcePath: "/elsewhere", DestPath: "/host/cwd"}},
			},
			cwd: "/host/cwd",
			wantRest: []string{
				"run", "--detach", "--init", "--name", "sb-1", "-w", "/workspace",
				"--mount", "type=bind,src=/workspace,dst=/workspace,readonly",
				"--mount", "type=bind,src=/elsewhere,dst=/host/cwd",
				"--network", "none",
				"img", "sleep", "infinity",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cwd := tt.cwd
			if cwd == "" {
				cwd = tt.sb.Dir
			}
			rest, env := splitEnv(t, buildDockerStartArgs(&tt.sb, cwd))
			if !reflect.DeepEqual(rest, tt.wantRest) {
				t.Errorf("args (excluding env) = %q, want %q", rest, tt.wantRest)
			}
			if !reflect.DeepEqual(env, tt.wantEnv) {
				t.Errorf("env = %q, want %q", env, tt.wantEnv)
			}
		})
	}
}

func TestBuildDockerExecArgs(t *testing.T) {
	sb := &DockerSandbox{Name: "sb-1", Image: "img", Dir: "/workspace"}

	tests := []struct {
		name     string
		cmd      SandboxCommand
		cwd      string
		wantRest []string
		wantEnv  []string
	}{
		{
			name:     "no dir defaults to host cwd",
			cmd:      SandboxCommand{Args: []string{"echo", "hi"}},
			cwd:      "/host/cwd",
			wantRest: []string{"exec", "-w", "/host/cwd", "sb-1", "echo", "hi"},
		},
		{
			name:     "command dir overrides host cwd",
			cmd:      SandboxCommand{Args: []string{"ls"}, Dir: "/tmp"},
			cwd:      "/host/cwd",
			wantRest: []string{"exec", "-w", "/tmp", "sb-1", "ls"},
		},
		{
			name:     "no dir and unknown cwd omits -w",
			cmd:      SandboxCommand{Args: []string{"ls"}},
			wantRest: []string{"exec", "sb-1", "ls"},
		},
		{
			name:     "env is passed as KEY=VALUE",
			cmd:      SandboxCommand{Args: []string{"env"}, Env: map[string]string{"CI": "1"}},
			cwd:      "/host/cwd",
			wantRest: []string{"exec", "-w", "/host/cwd", "sb-1", "env"},
			wantEnv:  []string{"CI=1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildDockerExecArgs(sb, &tt.cmd, tt.cwd)

			// The user's command comes after the container name; only parse flags before it.
			nameIdx := slices.Index(args, sb.Name)
			if got := args[nameIdx+1:]; !reflect.DeepEqual(got, tt.cmd.Args) {
				t.Errorf("command = %q, want %q", got, tt.cmd.Args)
			}

			rest, env := splitEnv(t, args[:nameIdx])
			rest = append(rest, args[nameIdx:]...)
			if !reflect.DeepEqual(rest, tt.wantRest) {
				t.Errorf("args (excluding env) = %q, want %q", rest, tt.wantRest)
			}
			if !reflect.DeepEqual(env, tt.wantEnv) {
				t.Errorf("env = %q, want %q", env, tt.wantEnv)
			}
		})
	}
}

// dockerTestImage returns the image for the Docker integration tests, skipping
// the test if Docker isn't usable. Set SANDBOX_TEST_IMAGE to override it.
// Set REQUIRE_DOCKER_TESTS=true (as CI does) to fail instead of skipping, so a
// broken Docker setup can't silently pass.
func dockerTestImage(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Docker integration test in -short mode")
	}

	skip := t.Skipf
	if os.Getenv("REQUIRE_DOCKER_TESTS") == "true" {
		skip = t.Fatalf
	}
	if _, err := exec.LookPath("docker"); err != nil {
		skip("docker not available")
	}
	if out, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
		skip("docker daemon not reachable: %v: %s", err, out)
	}
	image := os.Getenv("SANDBOX_TEST_IMAGE")
	if image == "" {
		image = "busybox:1.37"
	}
	if err := exec.Command("docker", "image", "inspect", image).Run(); err != nil {
		if out, err := exec.Command("docker", "pull", image).CombinedOutput(); err != nil {
			skip("could not pull %s: %v: %s", image, err, out)
		}
	}
	return image
}

func startDockerSandbox(t *testing.T, sb *DockerSandbox) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := sb.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := sb.Destroy(ctx); err != nil {
			t.Errorf("Destroy() error = %v", err)
		}
	})
}

func TestDockerSandbox_Lifecycle(t *testing.T) {
	image := dockerTestImage(t)

	// workDir is exposed read-only at the same path; outDir is mounted writable
	// inside it, at a mountpoint that has to exist in workDir.
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "input.txt"), []byte("from-host"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(workDir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	sb := NewDocker(
		fmt.Sprintf("professor-test-%d", time.Now().UnixNano()),
		image,
		workDir,
		[]MountPoint{{SourcePath: outDir, DestPath: filepath.Join(workDir, "out")}},
		false,
	)
	sb.Env = map[string]string{"SANDBOX_VAR": "sandbox-level"}
	startDockerSandbox(t, &sb)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// The test's cwd is this package's dir, which is outside workDir, so this
	// also checks that the host cwd gets mounted.
	t.Run("no dir defaults to host cwd", func(t *testing.T) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"sh", "-c", "pwd && test -f docker_test.go"}})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := strings.TrimSpace(string(out)); got != cwd {
			t.Errorf("pwd = %q, want %q", got, cwd)
		}
	})

	t.Run("command dir overrides", func(t *testing.T) {
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"pwd"}, Dir: "/tmp"})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := strings.TrimSpace(string(out)); got != "/tmp" {
			t.Errorf("pwd = %q, want /tmp", got)
		}
	})

	t.Run("sandbox and command env", func(t *testing.T) {
		out, err := sb.Exec(ctx, &SandboxCommand{
			Args: []string{"sh", "-c", "echo $SANDBOX_VAR $CMD_VAR"},
			Env:  map[string]string{"CMD_VAR": "cmd-level"},
		})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := strings.TrimSpace(string(out)); got != "sandbox-level cmd-level" {
			t.Errorf("env = %q, want %q", got, "sandbox-level cmd-level")
		}
	})

	t.Run("working dir is readable but not writable", func(t *testing.T) {
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"cat", "input.txt"}, Dir: workDir})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := string(out); got != "from-host" {
			t.Errorf("cat = %q, want from-host", got)
		}
		if out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"touch", "new"}, Dir: workDir}); err == nil {
			t.Errorf("write to read-only working dir succeeded: %s", out)
		}
	})

	t.Run("writable mount nested in working dir", func(t *testing.T) {
		if out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"sh", "-c", "echo result > out/result.txt"}, Dir: workDir}); err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		got, err := os.ReadFile(filepath.Join(outDir, "result.txt"))
		if err != nil {
			t.Fatalf("reading result on host: %v", err)
		}
		if strings.TrimSpace(string(got)) != "result" {
			t.Errorf("result.txt = %q, want result", got)
		}
	})

	t.Run("non-zero exit is an error", func(t *testing.T) {
		if _, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"false"}}); err == nil {
			t.Error("expected error from failing command")
		}
	})

	t.Run("no egress by default", func(t *testing.T) {
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"sh", "-c", "ip -o link | grep -v ' lo:' | wc -l"}})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := strings.TrimSpace(string(out)); got != "0" {
			t.Errorf("expected only loopback interface, found %s other(s)", got)
		}
	})
}

// A server started by one Exec in a goroutine must be reachable over loopback
// by a later Exec in the same sandbox: the serve-in-a-goroutine plan relies on it.
func TestDockerSandbox_BackgroundExecSharesLoopback(t *testing.T) {
	image := dockerTestImage(t)

	// The site is written on the host and served from the read-only working dir,
	// like the built dist/ will be.
	workDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(workDir, "www"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "www", "index.html"), []byte("ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	sb := NewDocker(fmt.Sprintf("professor-test-%d", time.Now().UnixNano()), image, workDir, nil, false)
	startDockerSandbox(t, &sb)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serveErr := make(chan error, 1)
	go func() {
		_, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"httpd", "-f", "-p", "127.0.0.1:4173", "-h", "www"}, Dir: workDir})
		serveErr <- err
	}()

	deadline := time.Now().Add(10 * time.Second)
	for {
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"wget", "-qO-", "http://127.0.0.1:4173/"}})
		if err == nil {
			if got := strings.TrimSpace(string(out)); got != "ok" {
				t.Errorf("body = %q, want ok", got)
			}
			return
		}
		select {
		case err := <-serveErr:
			t.Fatalf("server exited early: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became reachable: %v: %s", err, out)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// drainPiped reads both ExecPipe streams to EOF concurrently, then calls wait,
// the way callers are supposed to.
func drainPiped(t *testing.T, stdout, stderr io.Reader, wait func() error) (string, string, error) {
	t.Helper()
	errCh := make(chan []byte, 1)
	go func() {
		b, _ := io.ReadAll(stderr)
		errCh <- b
	}()
	outBytes, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("reading stdout: %v", err)
	}
	errBytes := <-errCh
	return string(outBytes), string(errBytes), wait()
}

func TestDockerSandbox_ExecPipe(t *testing.T) {
	image := dockerTestImage(t)

	sb := NewDocker(fmt.Sprintf("professor-test-%d", time.Now().UnixNano()), image, t.TempDir(), nil, false)
	startDockerSandbox(t, &sb)

	t.Run("stdout and stderr are separate", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stdout, stderr, wait, err := sb.ExecPipe(ctx, &SandboxCommand{
			Args: []string{"sh", "-c", "echo to-stdout; echo to-stderr >&2"},
		})
		if err != nil {
			t.Fatalf("ExecPipe() error = %v", err)
		}

		out, errOut, err := drainPiped(t, stdout, stderr, wait)
		if err != nil {
			t.Errorf("wait() error = %v", err)
		}
		if got := strings.TrimSpace(out); got != "to-stdout" {
			t.Errorf("stdout = %q, want to-stdout", got)
		}
		if got := strings.TrimSpace(errOut); got != "to-stderr" {
			t.Errorf("stderr = %q, want to-stderr", got)
		}
	})

	t.Run("wait reports a non-zero exit", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		stdout, stderr, wait, err := sb.ExecPipe(ctx, &SandboxCommand{
			Args: []string{"sh", "-c", "echo partial; exit 3"},
		})
		if err != nil {
			t.Fatalf("ExecPipe() error = %v", err)
		}

		out, _, err := drainPiped(t, stdout, stderr, wait)
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("wait() error = %v, want *exec.ExitError", err)
		}
		// Only check non-zero: the Cloud Run CLI collapses every failure to 1.
		if exitErr.ExitCode() == 0 {
			t.Errorf("exit code = 0, want non-zero")
		}
		if got := strings.TrimSpace(out); got != "partial" {
			t.Errorf("stdout = %q, want partial (output before the failure)", got)
		}
	})

	t.Run("wait reports context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		stdout, stderr, wait, err := sb.ExecPipe(ctx, &SandboxCommand{Args: []string{"sleep", "30"}})
		if err != nil {
			cancel()
			t.Fatalf("ExecPipe() error = %v", err)
		}
		cancel()

		start := time.Now()
		if _, _, err := drainPiped(t, stdout, stderr, wait); err == nil {
			t.Error("wait() error = nil, want an error after cancellation")
		}
		if elapsed := time.Since(start); elapsed > 10*time.Second {
			t.Errorf("cancelled command took %s to stop", elapsed)
		}
	})

	// The Playwright reporter's NDJSON events have to arrive while tests are
	// still running, not all at once when the command exits.
	t.Run("output streams before the command exits", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		start := time.Now()
		stdout, stderr, wait, err := sb.ExecPipe(ctx, &SandboxCommand{
			Args: []string{"sh", "-c", "echo first; sleep 2; echo second"},
		})
		if err != nil {
			t.Fatalf("ExecPipe() error = %v", err)
		}

		r := bufio.NewReader(stdout)
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("reading first line: %v", err)
		}
		if got := strings.TrimSpace(line); got != "first" {
			t.Errorf("first line = %q, want first", got)
		}
		if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
			t.Errorf("first line took %s; output is not streaming", elapsed)
		}

		rest, _, err := drainPiped(t, r, stderr, wait)
		if err != nil {
			t.Errorf("wait() error = %v", err)
		}
		if got := strings.TrimSpace(rest); got != "second" {
			t.Errorf("rest of stdout = %q, want second", got)
		}
	})
}
