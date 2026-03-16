package grade

import (
	"github.com/Hack4Impact-UMD/professor/build"
	"log"
	"os"
	"path/filepath"

	"github.com/Hack4Impact-UMD/professor/git"
	"golang.org/x/sync/errgroup"
)

func cloneRepos(jobId string, assessmentRepoURL string, testRepoURL string) (string, error) {
	pat := os.Getenv("GITHUB_PAT")

	gradingDir, err := os.MkdirTemp("", "job-*")

	if err != nil {
		return "", err
	}

	wg := errgroup.Group{}

	wg.Go(func() error {
		return git.CloneRepo(assessmentRepoURL, filepath.Join(gradingDir, "assessment"), pat)
	})
	wg.Go(func() error {
		return git.CloneRepo(testRepoURL, filepath.Join(gradingDir, "tests"), pat)
	})

	if err := wg.Wait(); err != nil {
		return "", err
	}

	return gradingDir, nil
}

func RunGradingJob(jobId string, assessmentRepoURL string, testRepoURL string) error {
	log.Println("Running grading job", jobId)

	gradingDir, err := cloneRepos(jobId, assessmentRepoURL, testRepoURL)
	if err != nil {
		return err
	}
	defer os.RemoveAll(gradingDir)

	buildOut, err := build.BuildAssessment(gradingDir)

	if err != nil {
		return err
	}

	return nil
}
