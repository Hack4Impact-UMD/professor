package sandbox

import (
	"context"
	"fmt"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rest, env := splitEnv(t, buildDockerStartArgs(&tt.sb))
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
		wantRest []string
		wantEnv  []string
	}{
		{
			name:     "defaults to sandbox dir",
			cmd:      SandboxCommand{Args: []string{"echo", "hi"}},
			wantRest: []string{"exec", "-w", "/workspace", "sb-1", "echo", "hi"},
		},
		{
			name:     "command dir overrides sandbox dir",
			cmd:      SandboxCommand{Args: []string{"ls"}, Dir: "/tmp"},
			wantRest: []string{"exec", "-w", "/tmp", "sb-1", "ls"},
		},
		{
			name:     "env is passed as KEY=VALUE",
			cmd:      SandboxCommand{Args: []string{"env"}, Env: map[string]string{"CI": "1"}},
			wantRest: []string{"exec", "-w", "/workspace", "sb-1", "env"},
			wantEnv:  []string{"CI=1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildDockerExecArgs(sb, &tt.cmd)

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
		if err := sb.Destroy(); err != nil {
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

	t.Run("working dir defaults to sandbox dir", func(t *testing.T) {
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"pwd"}})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := strings.TrimSpace(string(out)); got != workDir {
			t.Errorf("pwd = %q, want %q", got, workDir)
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
		out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"cat", "input.txt"}})
		if err != nil {
			t.Fatalf("Exec() error = %v: %s", err, out)
		}
		if got := string(out); got != "from-host" {
			t.Errorf("cat = %q, want from-host", got)
		}
		if out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"touch", "new"}}); err == nil {
			t.Errorf("write to read-only working dir succeeded: %s", out)
		}
	})

	t.Run("writable mount nested in working dir", func(t *testing.T) {
		if out, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"sh", "-c", "echo result > out/result.txt"}}); err != nil {
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
		_, err := sb.Exec(ctx, &SandboxCommand{Args: []string{"httpd", "-f", "-p", "127.0.0.1:4173", "-h", "www"}})
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
