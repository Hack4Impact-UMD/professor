package builder

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Hack4Impact-UMD/professor/util"
)

// Timeouts bound how long untrusted assessment code (dependency lifecycle
// scripts run by `pnpm install`, and `vite.config.ts` executed by the build) is
// allowed to run. Without these, a malicious postinstall/build could hang until
// Cloud Run's request timeout while consuming CPU/memory.
const (
	installTimeout = 5 * time.Minute
	buildTimeout   = 5 * time.Minute
)

// runBuildCommand runs a command bounded by ctx, returning its combined output.
// If ctx's deadline fires, the process is killed (via exec.CommandContext) and a
// clear timeout error is returned instead of the generic "signal: killed".
func runBuildCommand(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env

	out, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("%s timed out after %s", name, deadlineFor(ctx))
	}

	return string(out), err
}

// deadlineFor is a best-effort helper for a readable timeout message.
func deadlineFor(ctx context.Context) time.Duration {
	if dl, ok := ctx.Deadline(); ok {
		return time.Until(dl).Round(time.Second)
	}
	return 0
}

func InstallDeps(repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), installTimeout)
	defer cancel()

	return runBuildCommand(ctx, repoDir, util.SandboxedCommandEnv(), "pnpm", "install")
}

func BuildAssessment(assessmentDir string) (string, error) {
	binDir := filepath.Join(assessmentDir, "node_modules", ".bin")

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	tscOut, err := runBuildCommand(ctx, assessmentDir, util.SandboxedEnv(), filepath.Join(binDir, "tsc"), "-b")
	if err != nil {
		return tscOut, err
	}

	viteOut, err := runBuildCommand(ctx, assessmentDir, util.SandboxedEnv(), filepath.Join(binDir, "vite"), "build")

	out := tscOut + viteOut

	if err != nil {
		return out, err
	}

	return out, nil
}
