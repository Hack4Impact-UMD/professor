package build

import "os/exec"

func BuildAssessment(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "run", "build")
	cmd.Dir = assessmentDir

	if err := cmd.Run(); err != nil {
		return "", err
	}

	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}

	return string(out), nil
}
