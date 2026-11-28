package builder

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hack4Impact-UMD/professor/sandbox"
	"github.com/Hack4Impact-UMD/professor/util"
)

const buildConfigDirName = "build_config"

const (
	installTimeout = 5 * time.Minute
	buildTimeout   = 5 * time.Minute
)

func envmap(e []string) map[string]string {
	m := make(map[string]string)
	for _, v := range e {
		pts := strings.Split(v, "=")
		m[pts[0]] = pts[1]
	}
	return m
}

func runBuildCommand(ctx context.Context, sb sandbox.Sandbox, dir string, env []string, name string, args ...string) (string, error) {
	ar := make([]string, 0)
	ar = append(ar, name)
	ar = append(ar, args...)
	out, err := sb.Exec(ctx, &sandbox.SandboxCommand{
		Args: ar,
		Dir:  dir,
		Env:  envmap(env),
	})

	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("%s timed out after %s", name, deadlineFor(ctx))
	}

	return string(out), err
}

func deadlineFor(ctx context.Context) time.Duration {
	if dl, ok := ctx.Deadline(); ok {
		return time.Until(dl).Round(time.Second)
	}
	return 0
}

func InstallDeps(ctx context.Context, sb sandbox.Sandbox, repoDir string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()

	out, err := runBuildCommand(ctx, sb, repoDir, util.SandboxedCommandEnv(), "pnpm", "install", "--ignore-scripts")
	if err != nil {
		return out, err
	}

	tailwindOut, err := ensureTailwind(ctx, sb, repoDir)
	return out + tailwindOut, err
}

func ensureTailwind(ctx context.Context, sb sandbox.Sandbox, repoDir string) (string, error) {
	return runBuildCommand(ctx, sb, repoDir, util.SandboxedCommandEnv(), "pnpm", "add", "-D", "--ignore-scripts", "tailwindcss", "@tailwindcss/vite")
}

func BuildAssessment(ctx context.Context, sb sandbox.Sandbox, assessmentDir string, testDir string) (string, error) {
	configNote, err := applyTrustedBuildConfig(assessmentDir, testDir)
	if err != nil {
		return configNote, err
	}

	binDir := filepath.Join(assessmentDir, "node_modules", ".bin")

	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout)
	defer cancel()

	tscOut, err := runBuildCommand(ctx, sb, assessmentDir, util.SandboxedEnv(), filepath.Join(binDir, "tsc"), "-b")
	if err != nil {
		return configNote + tscOut, err
	}

	viteOut, err := runBuildCommand(ctx, sb, assessmentDir, util.SandboxedEnv(), filepath.Join(binDir, "vite"), "build")

	out := configNote + tscOut + viteOut

	if err != nil {
		return out, err
	}

	return out, nil
}

func applyTrustedBuildConfig(assessmentDir string, testDir string) (string, error) {
	buildConfigDir := filepath.Join(testDir, buildConfigDirName)

	info, err := os.Stat(buildConfigDir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("test repo has no %s/; aborting build\n", buildConfigDirName)
	}

	if err := overlayDir(buildConfigDir, assessmentDir); err != nil {
		return "", fmt.Errorf("failed to apply trusted build config: %w", err)
	}

	return fmt.Sprintf("Applied trusted build config from test repo %s/\n", buildConfigDirName), nil
}

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

		// remove simlinks
		if li, lerr := os.Lstat(target); lerr == nil && li.Mode()&fs.ModeSymlink != 0 {
			if err := os.Remove(target); err != nil {
				return err
			}
		}

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
