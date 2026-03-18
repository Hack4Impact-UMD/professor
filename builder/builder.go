package builder

import (
	"os/exec"

	"github.com/Hack4Impact-UMD/professor/util"
)

func InstallAssessmentDeps(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "install")
	cmd.Dir = assessmentDir
	cmd.Env = util.SandboxedEnv()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

func BuildAssessment(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "run", "build")
	cmd.Dir = assessmentDir
	cmd.Env = util.SandboxedEnv()

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
