package grade

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/Hack4Impact-UMD/professor/builder"
	"github.com/Hack4Impact-UMD/professor/git"
	"github.com/Hack4Impact-UMD/professor/playwright"
	"github.com/Hack4Impact-UMD/professor/serve"
	"github.com/Hack4Impact-UMD/professor/util"
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
		os.RemoveAll(gradingDir)
		return cloneResult{}, err
	}

	return cloneResult{
		GradingDir:    gradingDir,
		AssessmentDir: assessmentDir,
		TestDir:       testDir,
	}, nil
}

func RunGradingJob(jobId string, assessmentRepoURL string, testRepoURL string, reporter util.GradingJobReporter) error {
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
		log.Printf("Install failed: %v", err)
		return err
	}

	reporter.OnBuildStart(jobId)
	buildOut, err := builder.BuildAssessment(clone.AssessmentDir)
	reporter.OnBuildEnd(jobId, buildOut, err)

	log.Printf("build output: %v", buildOut)
	if err != nil {
		log.Printf("Build failed: %v", err)
		return err
	}

	port, stop, err := serve.ServeAssessment(filepath.Join(clone.AssessmentDir, "dist"))

	if err != nil {
		reporter.OnServe(jobId, err)
		log.Printf("Serve failed on port %d: %v", port, err)
		return err
	}

	defer stop()

	if err := util.WaitForPort(port, 5*time.Second); err != nil {
		reporter.OnServe(jobId, err)
		log.Printf("File server did not respond in 5 seconds!")
		return err
	}

	reporter.OnServe(jobId, nil)

	if err := playwright.RunPlaywrightTests(jobId, clone.TestDir, port, reporter); err != nil {
		log.Printf("Failed to run playwright tests %v", err)
		return err
	}

	log.Printf("Tests run successfully for job %v", jobId)

	return nil
}
