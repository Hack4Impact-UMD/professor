package build

import "os/exec"

func BuildAssessment(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "run", "build")
	cmd.Dir = assessmentDir

	out, err := cmd.CombinedOutput()

	if err != nil {
		return "", err
	}

	return string(out), nil
}
