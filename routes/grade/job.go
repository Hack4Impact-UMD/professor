package grade

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Hack4Impact-UMD/professor/builder"
	"github.com/Hack4Impact-UMD/professor/git"
	"golang.org/x/sync/errgroup"
)

type CloneResult struct {
	GradingDir    string
	AssessmentDir string
	TestDir       string
}

func cloneRepos(jobId string, assessmentRepoURL string, testRepoURL string) (CloneResult, error) {
	pat := os.Getenv("GITHUB_PAT")

	gradingDir, err := os.MkdirTemp("", "job-*")

	if err != nil {
		return CloneResult{}, err
	}

	wg := errgroup.Group{}

	assessmentDir := filepath.Join(gradingDir, "assessment")
	testDir := filepath.Join(gradingDir, "tests")

	wg.Go(func() error {
		return git.CloneRepo(assessmentRepoURL, assessmentDir, pat)
	})
	wg.Go(func() error {
		return git.CloneRepo(testRepoURL, testDir, pat)
	})

	if err := wg.Wait(); err != nil {
		return CloneResult{}, err
	}

	return CloneResult{
		GradingDir:    gradingDir,
		AssessmentDir: assessmentDir,
		TestDir:       testDir,
	}, nil
}

func RunGradingJob(jobId string, assessmentRepoURL string, testRepoURL string) error {
	log.Println("Running grading job", jobId)

	clone, err := cloneRepos(jobId, assessmentRepoURL, testRepoURL)
	if err != nil {
		return err
	}
	defer os.RemoveAll(clone.GradingDir)

	installOut, err := builder.InstallAssessmentDeps(clone.AssessmentDir)

	log.Printf("install output: %v", installOut)

	if err != nil {
		// failed on install
		// TODO: update job status, add logs
		return nil
	}

	buildOut, err := builder.BuildAssessment(clone.AssessmentDir)

	log.Printf("build output: %v", buildOut)
	if err != nil {
		// failed on build
		// TODO: update job status, add logs
		return nil
	}

	// TODO: run tests

	if err != nil {
		return err
	}

	return nil
}
