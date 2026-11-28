package sandbox

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type SandboxCommand struct {
	Args []string
	Dir  string
	Env  map[string]string
}

func SandboxCmd(cmd string, arg ...string) SandboxCommand {
	args := make([]string, 0)
	args = append(args, cmd)
	args = append(args, arg...)
	return SandboxCommand{
		Args: args,
	}
}

type MountPoint struct {
	SourcePath string
	DestPath   string
	ReadOnly   bool
}

func (p *MountPoint) Arg() []string {
	//type=bind,src=<abs-host-path>,dst=<abs-path>[,readonly]
	mountDesc := fmt.Sprintf("type=bind,src=%v,dst=%v", p.SourcePath, p.DestPath)
	if p.ReadOnly {
		mountDesc += ",readonly"
	}

	return []string{"--mount", mountDesc}
}

type Sandbox interface {
	Start(ctx context.Context) error
	Exec(ctx context.Context, cmd *SandboxCommand) ([]byte, error)
	ExecPipe(ctx context.Context, cmd *SandboxCommand) (stdout io.ReadCloser, stderr io.ReadCloser, wait func() error, err error)
	Destroy(ctx context.Context) error
}

func startPiped(c *exec.Cmd) (stdout io.ReadCloser, stderr io.ReadCloser, wait func() error, err error) {
	stdout, err = c.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stderr, err = c.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	if err := c.Start(); err != nil {
		return nil, nil, nil, err
	}
	return stdout, stderr, c.Wait, nil
}
