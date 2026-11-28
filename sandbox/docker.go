package sandbox

import (
	"context"
	"fmt"
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

func buildDockerStartArgs(s *DockerSandbox) []string {
	args := []string{"run", "--detach", "--init", "--name", s.Name, "-w", s.Dir}

	if !slices.ContainsFunc(s.Mounts, func(m MountPoint) bool { return m.DestPath == s.Dir }) {
		args = append(args, (&MountPoint{SourcePath: s.Dir, DestPath: s.Dir, ReadOnly: true}).Arg()...)
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

func buildDockerExecArgs(s *DockerSandbox, cmd *SandboxCommand) []string {
	dir := cmd.Dir
	if dir == "" {
		dir = s.Dir
	}
	args := []string{"exec", "-w", dir}
	for k, v := range cmd.Env {
		args = append(args, "-e", fmt.Sprintf("%v=%v", k, v))
	}
	args = append(args, s.Name)
	args = append(args, cmd.Args...)
	return args
}

func (s *DockerSandbox) Start(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "docker", buildDockerStartArgs(s)...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker sandbox start (%s): %w: %s", s.Name, err, out)
	}
	return nil
}

func (s *DockerSandbox) Exec(ctx context.Context, cmd *SandboxCommand) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", buildDockerExecArgs(s, cmd)...).CombinedOutput()
}

func (s *DockerSandbox) Destroy() error {
	out, err := exec.Command("docker", "rm", "--force", s.Name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker sandbox destroy (%s): %w: %s", s.Name, err, out)
	}
	return nil
}
