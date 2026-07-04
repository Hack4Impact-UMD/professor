package builder

import (
	"os/exec"
	"path/filepath"

	"github.com/Hack4Impact-UMD/professor/util"
)

func InstallDeps(repoDir string) (string, error) {
	cmd := exec.Command("pnpm", "install")
	cmd.Dir = repoDir
	cmd.Env = util.SandboxedEnv()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

func BuildAssessment(assessmentDir string) (string, error) {
	binDir := filepath.Join(assessmentDir, "node_modules", ".bin")

	tscCmd := exec.Command(filepath.Join(binDir, "tsc"), "-b")
	tscCmd.Dir = assessmentDir
	tscCmd.Env = util.SandboxedEnv()

	tscOut, err := tscCmd.CombinedOutput()
	if err != nil {
		return string(tscOut), err
	}

	viteCmd := exec.Command(filepath.Join(binDir, "vite"), "build")
	viteCmd.Dir = assessmentDir
	viteCmd.Env = util.SandboxedEnv()

	viteOut, err := viteCmd.CombinedOutput()

	out := string(tscOut) + string(viteOut)

	if err != nil {
		return out, err
	}

	return out, nil
}
