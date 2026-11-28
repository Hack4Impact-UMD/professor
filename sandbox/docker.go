package sandbox

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
)

// another impl of the sandbox interface with docker for local testing
type DockerSandbox struct {
	Name   string
	Image  string
	Dir    string
	Mounts []MountPoint
	Net    bool
	Env    map[string]string
}

var _ Sandbox = &DockerSandbox{}

func NewDocker(name string, image string, dir string, mounts []MountPoint, netAccess bool) DockerSandbox {
	return DockerSandbox{
		Name:   name,
		Image:  image,
		Dir:    dir,
		Mounts: mounts,
		Net:    netAccess,
	}
}

// hostCwd is the host's working directory, or "" if it can't be determined.
func hostCwd() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func buildDockerStartArgs(s *DockerSandbox, cwd string) []string {
	args := []string{"run", "--detach", "--init", "--name", s.Name, "-w", s.Dir}

	hasMountAt := func(dst string) bool {
		return slices.ContainsFunc(s.Mounts, func(m MountPoint) bool { return m.DestPath == dst })
	}
	if !hasMountAt(s.Dir) {
		args = append(args, (&MountPoint{SourcePath: s.Dir, DestPath: s.Dir, ReadOnly: true}).Arg()...)
	}

	if cwd != "" && cwd != "/" && cwd != s.Dir && !hasMountAt(cwd) {
		args = append(args, (&MountPoint{SourcePath: cwd, DestPath: cwd, ReadOnly: true}).Arg()...)
	}

	for _, mount := range s.Mounts {
		args = append(args, mount.Arg()...)
	}
	for k, v := range s.Env {
		args = append(args, "-e", fmt.Sprintf("%v=%v", k, v))
	}
	if !s.Net {
		args = append(args, "--network", "none")
	}

	args = append(args, s.Image, "sleep", "infinity")
	return args
}

func buildDockerExecArgs(s *DockerSandbox, cmd *SandboxCommand, cwd string) []string {
	dir := cmd.Dir
	if dir == "" {
		dir = cwd
	}
	args := []string{"exec"}
	if dir != "" {
		args = append(args, "-w", dir)
	}
	for k, v := range cmd.Env {
		args = append(args, "-e", fmt.Sprintf("%v=%v", k, v))
	}
	args = append(args, s.Name)
	args = append(args, cmd.Args...)
	return args
}

func (s *DockerSandbox) Start(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "docker", buildDockerStartArgs(s, hostCwd())...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker sandbox start (%s): %w: %s", s.Name, err, out)
	}
	return nil
}

func (s *DockerSandbox) Exec(ctx context.Context, cmd *SandboxCommand) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", buildDockerExecArgs(s, cmd, hostCwd())...).CombinedOutput()
}

func (s *DockerSandbox) ExecPipe(ctx context.Context, cmd *SandboxCommand) (stdout io.ReadCloser, stderr io.ReadCloser, wait func() error, err error) {
	return startPiped(exec.CommandContext(ctx, "docker", buildDockerExecArgs(s, cmd, hostCwd())...))
}

func (s *DockerSandbox) Destroy(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "docker", "rm", "--force", s.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker sandbox destroy (%s): %w: %s", s.Name, err, out)
	}
	return nil
}
