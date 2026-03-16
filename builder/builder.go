package builder

import "os/exec"

func InstallAssessmentDeps(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "install")
	cmd.Dir = assessmentDir

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

func BuildAssessment(assessmentDir string) (string, error) {
	cmd := exec.Command("bun", "run", "build")
	cmd.Dir = assessmentDir

	out, err := cmd.CombinedOutput()

	if err != nil {
		return string(out), err
	}

	return string(out), nil
}
