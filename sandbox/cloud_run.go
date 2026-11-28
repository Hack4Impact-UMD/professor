package sandbox

import (
	"context"
	"fmt"
	"os/exec"
)

type CloudRunSandbox struct {
	Name   string
	Dir    string
	Mounts []MountPoint
	Net    bool
	Env    map[string]string
}

var _ Sandbox = &CloudRunSandbox{}

func buildStartArgs(s *CloudRunSandbox) []string {
	mountArgs := make([]string, 0)

	for _, mount := range s.Mounts {
		mountArgs = append(mountArgs, mount.Arg()...)
	}

	args := []string{
		"run", "--detach", "--stdin=false", "-w", s.Dir,
	}

	args = append(args, mountArgs...)
	for k, v := range s.Env {
		args = append(args, "-e", fmt.Sprintf("%v=%v", k, v))
	}
	if s.Net {
		args = append(args, "--allow-egress")
	}

	args = append(args, s.Name)

	return args
}

func buildExecArgs(s *CloudRunSandbox, cmd *SandboxCommand) []string {
	dir := cmd.Dir
	if dir == "" {
		dir = s.Dir
	}
	args := []string{"exec", "--stdin=false", "-w", dir}
	for k, v := range cmd.Env {
		args = append(args, "--env", fmt.Sprintf("%v=%v", k, v))
	}
	args = append(args, s.Name, "--")
	args = append(args, cmd.Args...)
	return args
}

func NewCloudRun(name string, dir string, mounts []MountPoint, netAccess bool) CloudRunSandbox {
	return CloudRunSandbox{
		Name:   name,
		Dir:    dir,
		Net:    netAccess,
		Mounts: mounts,
	}
}

func (s *CloudRunSandbox) Start(ctx context.Context) error {
	args := buildStartArgs(s)
	out, err := exec.Command("sandbox", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("sandbox start (%s): %w: %s", s.Name, err, out)
	}

	return nil
}

func (s *CloudRunSandbox) Exec(ctx context.Context, cmd *SandboxCommand) ([]byte, error) {
	args := buildExecArgs(s, cmd)
	return exec.Command("sandbox", args...).CombinedOutput()
}

func (s *CloudRunSandbox) Destroy() error {
	out, err := exec.Command("sandbox", "delete", "--force", s.Name).CombinedOutput()

	if err != nil {
		return fmt.Errorf("sandbox destory (%s): %w: %s", s.Name, err, out)
	}

	return nil
}
