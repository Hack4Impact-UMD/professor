package sandbox

import (
	"context"
	"fmt"
)

type SandboxCommand struct {
	Args []string
	Dir  string
	Env  map[string]string
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
	Destroy() error
}
