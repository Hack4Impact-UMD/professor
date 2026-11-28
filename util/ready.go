package util

import (
	"context"
	"fmt"
	"time"

	"github.com/Hack4Impact-UMD/professor/sandbox"
)

func WaitForPort(ctx context.Context, sb sandbox.Sandbox, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf(":%d", port)

	for time.Now().Before(deadline) {
		cmd := sandbox.SandboxCmd("curl", url)
		_, err := sb.Exec(ctx, &cmd)
		if err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("port %d did not respond in %v", port, timeout)
}
