package builder

import (
	"os/exec"

	"github.com/Hack4Impact-UMD/professor/util"
)

func InstallDeps(repoDir string) (string, error) {
	cmd := exec.Command("pnpm", "install")
	cmd.Dir = repoDir
	cmd.Env = util.SandboxedCommandEnv()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

func BuildAssessment(assessmentDir string) (string, error) {
	cmd := exec.Command("pnpm", "run", "build")
	cmd.Dir = assessmentDir
	cmd.Env = util.SandboxedCommandEnv()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
