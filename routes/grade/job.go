package grade

import (
	"log"
	"os"
	"path/filepath"

	"github.com/Hack4Impact-UMD/professor/builder"
	"github.com/Hack4Impact-UMD/professor/git"
	"github.com/Hack4Impact-UMD/professor/serve"
	"golang.org/x/sync/errgroup"
)

type cloneResult struct {
	GradingDir    string
	AssessmentDir string
	TestDir       string
}

func cloneRepos(jobId string, assessmentRepoURL string, testRepoURL string) (cloneResult, error) {
	pat := os.Getenv("GITHUB_PAT")

	gradingDir, err := os.MkdirTemp("", "job-*")

	if err != nil {
		return cloneResult{}, err
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
		return cloneResult{}, err
	}

	return cloneResult{
		GradingDir:    gradingDir,
		AssessmentDir: assessmentDir,
		TestDir:       testDir,
	}, nil
}

func RunGradingJob(jobId string, assessmentRepoURL string, testRepoURL string, reporter GradingJobReporter) error {
	log.Println("Running grading job", jobId)
	reporter.OnGradeStart(jobId)

	reporter.OnCloneStart(jobId, assessmentRepoURL, testRepoURL)
	clone, err := cloneRepos(jobId, assessmentRepoURL, testRepoURL)
	reporter.OnCloneEnd(jobId, assessmentRepoURL, testRepoURL, err)
	if err != nil {
		return err
	}
	defer os.RemoveAll(clone.GradingDir)

	reporter.OnInstallStart(jobId)
	installOut, err := builder.InstallAssessmentDeps(clone.AssessmentDir)
	reporter.OnInstallEnd(jobId, installOut, err)

	log.Printf("install output: %v", installOut)

	if err != nil {
		log.Fatalf("Install failed: %v", err)
		return err
	}

	reporter.OnBuildStart(jobId)
	buildOut, err := builder.BuildAssessment(clone.AssessmentDir)
	reporter.OnBuildEnd(jobId, buildOut, err)

	log.Printf("build output: %v", buildOut)
	if err != nil {
		log.Fatalf("Build failed: %v", err)
		return err
	}

	port, stop, err := serve.ServeAssessment(filepath.Join(clone.AssessmentDir, "dist"))
	defer stop()

	reporter.OnServe(jobId, err)

	if err != nil {
		log.Fatalf("Serve failed on port %d: %v", port, err)
		return err
	}

	return nil
}
