package builder

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Hack4Impact-UMD/professor/util"
)

// buildConfigDirName is the folder in the TRUSTED test repo whose contents are
// overlaid onto the assessment before building. It lets the grader impose a clean
// build config (vite.config, tsconfig, postcss/tailwind config, etc.), replacing
// the submission's own, so the build does not execute attacker-authored config.
const buildConfigDirName = "build_config"

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

	// --ignore-scripts blocks the assessment's own root package.json lifecycle
	// scripts (preinstall/install/postinstall/prepare), which are the primary
	// Node-side arbitrary-code-execution vector for untrusted submissions. pnpm
	// already declines to run dependency build scripts by default, so this mainly
	// closes the root-script hole with minimal impact on legitimate installs.
	return runBuildCommand(ctx, repoDir, util.SandboxedCommandEnv(), "pnpm", "install", "--ignore-scripts")
}

func BuildAssessment(assessmentDir string, testDir string) (string, error) {
	// Overlay the trusted test repo's build_config/ onto the assessment so the
	// build runs against grader-controlled config instead of the submission's
	// own (which would execute as Node code during the build).
	configNote, err := applyTrustedBuildConfig(assessmentDir, testDir)
	if err != nil {
		return configNote, err
	}

	binDir := filepath.Join(assessmentDir, "node_modules", ".bin")

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	tscOut, err := runBuildCommand(ctx, assessmentDir, util.SandboxedEnv(), filepath.Join(binDir, "tsc"), "-b")
	if err != nil {
		return configNote + tscOut, err
	}

	viteOut, err := runBuildCommand(ctx, assessmentDir, util.SandboxedEnv(), filepath.Join(binDir, "vite"), "build")

	out := configNote + tscOut + viteOut

	if err != nil {
		return out, err
	}

	return out, nil
}

// applyTrustedBuildConfig overlays the contents of <testDir>/build_config onto
// assessmentDir when present, returning a note for the build log. If the test
// repo ships no build_config, the submission's own config is used and the note
// records that (production test repos should always provide one).
func applyTrustedBuildConfig(assessmentDir string, testDir string) (string, error) {
	buildConfigDir := filepath.Join(testDir, buildConfigDirName)

	info, err := os.Stat(buildConfigDir)
	if err != nil || !info.IsDir() {
		return fmt.Sprintf("WARNING: test repo has no %s/; building with the submission's own build config\n", buildConfigDirName), nil
	}

	if err := overlayDir(buildConfigDir, assessmentDir); err != nil {
		return "", fmt.Errorf("failed to apply trusted build config: %w", err)
	}

	return fmt.Sprintf("Applied trusted build config from test repo %s/\n", buildConfigDirName), nil
}

// overlayDir copies the regular files under srcDir into dstDir (creating
// directories as needed and overwriting existing files). Symlinks are skipped so
// a config overlay can never redirect a write outside dstDir.
func overlayDir(srcDir string, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(dstDir, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
