package sandbox

import (
	"reflect"
	"slices"
	"testing"
)

// splitEnv pulls every "-e X" / "--env X" pair out of args. Env comes from a
// map, so its order is random; the tests check env values as a sorted set
// and check everything else in order.
func splitEnv(t *testing.T, args []string) (rest, env []string) {
	t.Helper()
	for i := 0; i < len(args); i++ {
		if args[i] == "-e" || args[i] == "--env" {
			if i+1 >= len(args) {
				t.Fatalf("%s flag with no value at end of args: %q", args[i], args)
			}
			env = append(env, args[i+1])
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	slices.Sort(env)
	return rest, env
}

func TestMountPointArg(t *testing.T) {
	tests := []struct {
		name  string
		mount MountPoint
		want  []string
	}{
		{
			name:  "read-write",
			mount: MountPoint{SourcePath: "/host/src", DestPath: "/workspace/src"},
			want:  []string{"--mount", "type=bind,src=/host/src,dst=/workspace/src"},
		},
		{
			name:  "read-only",
			mount: MountPoint{SourcePath: "/host/tests", DestPath: "/workspace/tests", ReadOnly: true},
			want:  []string{"--mount", "type=bind,src=/host/tests,dst=/workspace/tests,readonly"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mount.Arg(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Arg() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildStartArgs(t *testing.T) {
	tests := []struct {
		name     string
		sb       CloudRunSandbox
		wantRest []string
		wantEnv  []string
	}{
		{
			name:     "minimal",
			sb:       CloudRunSandbox{Name: "sb-1", Dir: "/workspace"},
			wantRest: []string{"run", "--detach", "--stdin=false", "-w", "/workspace", "sb-1"},
		},
		{
			name: "mounts keep their order",
			sb: CloudRunSandbox{
				Name: "sb-1",
				Dir:  "/workspace",
				Mounts: []MountPoint{
					{SourcePath: "/a", DestPath: "/workspace/a"},
					{SourcePath: "/b", DestPath: "/workspace/b", ReadOnly: true},
				},
			},
			wantRest: []string{
				"run", "--detach", "--stdin=false", "-w", "/workspace",
				"--mount", "type=bind,src=/a,dst=/workspace/a",
				"--mount", "type=bind,src=/b,dst=/workspace/b,readonly",
				"sb-1",
			},
		},
		{
			name:     "network access",
			sb:       CloudRunSandbox{Name: "sb-1", Dir: "/workspace", Net: true},
			wantRest: []string{"run", "--detach", "--stdin=false", "-w", "/workspace", "--allow-egress", "sb-1"},
		},
		{
			name: "env",
			sb: CloudRunSandbox{
				Name: "sb-1",
				Dir:  "/workspace",
				Env:  map[string]string{"PATH": "/usr/bin:/bin", "CI": "1"},
			},
			wantRest: []string{"run", "--detach", "--stdin=false", "-w", "/workspace", "sb-1"},
			wantEnv:  []string{"CI=1", "PATH=/usr/bin:/bin"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildStartArgs(&tt.sb)

			// No "--" or command: run starts an idle sandbox.
			if slices.Contains(args, "--") {
				t.Errorf("start args must not contain \"--\" (no initial command): %q", args)
			}
			// The sandbox name has to be the last arg.
			if last := args[len(args)-1]; last != tt.sb.Name {
				t.Errorf("last arg = %q, want sandbox name %q", last, tt.sb.Name)
			}

			rest, env := splitEnv(t, args)
			if !reflect.DeepEqual(rest, tt.wantRest) {
				t.Errorf("args (excluding env) = %q, want %q", rest, tt.wantRest)
			}
			if !reflect.DeepEqual(env, tt.wantEnv) {
				t.Errorf("env = %q, want %q", env, tt.wantEnv)
			}
		})
	}
}

func TestBuildExecArgs(t *testing.T) {
	sb := &CloudRunSandbox{Name: "sb-1", Dir: "/workspace"}

	tests := []struct {
		name     string
		cmd      SandboxCommand
		wantRest []string
		wantEnv  []string
	}{
		{
			name:     "no dir omits -w so the CLI uses the host cwd",
			cmd:      SandboxCommand{Args: []string{"echo", "hi"}},
			wantRest: []string{"exec", "--stdin=false", "sb-1", "--", "echo", "hi"},
		},
		{
			name:     "command dir is passed with -w",
			cmd:      SandboxCommand{Args: []string{"ls"}, Dir: "/tmp"},
			wantRest: []string{"exec", "--stdin=false", "-w", "/tmp", "sb-1", "--", "ls"},
		},
		{
			name: "command args are passed through verbatim",
			cmd:  SandboxCommand{Args: []string{"bash", "-c", "echo $HOME && -w --env"}},
			wantRest: []string{
				"exec", "--stdin=false", "sb-1", "--",
				"bash", "-c", "echo $HOME && -w --env",
			},
		},
		{
			name: "env is passed as KEY=VALUE",
			cmd: SandboxCommand{
				Args: []string{"env"},
				Env:  map[string]string{"PATH": "/usr/bin:/bin", "CI": "1"},
			},
			wantRest: []string{"exec", "--stdin=false", "sb-1", "--", "env"},
			wantEnv:  []string{"CI=1", "PATH=/usr/bin:/bin"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildExecArgs(sb, &tt.cmd)

			// Only parse flags before "--"; everything after is the user's command.
			sep := slices.Index(args, "--")
			if sep == -1 {
				t.Fatalf("exec args missing \"--\" separator: %q", args)
			}
			if got := args[sep+1:]; !reflect.DeepEqual(got, tt.cmd.Args) {
				t.Errorf("command after \"--\" = %q, want %q", got, tt.cmd.Args)
			}

			rest, env := splitEnv(t, args[:sep])
			rest = append(rest, args[sep:]...)
			if !reflect.DeepEqual(rest, tt.wantRest) {
				t.Errorf("args (excluding env) = %q, want %q", rest, tt.wantRest)
			}
			if !reflect.DeepEqual(env, tt.wantEnv) {
				t.Errorf("env = %q, want %q", env, tt.wantEnv)
			}
		})
	}
}
