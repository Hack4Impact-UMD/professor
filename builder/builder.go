package builder

import (
	"os"
	"os/exec"
)

func InstallAssessmentDeps(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "install")
	cmd.Dir = assessmentDir
	cmd.Env = []string{"HOME=" + os.Getenv("HOME"), "PATH=" + os.Getenv("PATH")}

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

func BuildAssessment(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "run", "build")
	cmd.Dir = assessmentDir
	cmd.Env = []string{"HOME=" + os.Getenv("HOME"), "PATH=" + os.Getenv("PATH")}

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
