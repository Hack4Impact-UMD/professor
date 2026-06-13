package grade

import (
	"errors"
	"log/slog"
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

const MAX_REPO_SIZE_MB = 10

func cloneWithSizeCheck(path string, dest string, pat string) error {
	size, err := git.GetRepoSizeKB(path)

	if err != nil {
		return errors.New("failed to get repo details")
	}

	if size > MAX_REPO_SIZE_MB*1024 {
		return errors.New("repo too large")
	}

	return git.CloneRepo(path, dest, pat)
}

func cloneRepos(jobId string, assessmentRepoPath string, testRepoPath string) (cloneResult, error) {
	pat := os.Getenv("GITHUB_PAT")

	gradingDir, err := os.MkdirTemp("", "job-*")

	if err != nil {
		return cloneResult{}, err
	}

	wg := errgroup.Group{}

	assessmentDir := filepath.Join(gradingDir, "assessment")
	testDir := filepath.Join(gradingDir, "tests")

	wg.Go(func() error {
		return cloneWithSizeCheck(assessmentRepoPath, assessmentDir, pat)
	})
	wg.Go(func() error {
		return cloneWithSizeCheck(testRepoPath, testDir, pat)
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

func RunGradingJobLocal(jobId string, assessmentRepoDir string, testRepoDir string, reporter playwright.GradingJobReporter, logger *slog.Logger) error {
	return grade(jobId, assessmentRepoDir, testRepoDir, reporter, logger)
}

func RunGradingJob(jobId string, assessmentRepoPath string, testRepoPath string, reporter playwright.GradingJobReporter, logger *slog.Logger) error {
	logger.Info("running grading job", "jobId", jobId)
	reporter.OnGradeStart(jobId)

	reporter.OnCloneStart(jobId, assessmentRepoPath, testRepoPath)
	clone, err := cloneRepos(jobId, assessmentRepoPath, testRepoPath)
	reporter.OnCloneEnd(jobId, assessmentRepoPath, testRepoPath, err)
	if err != nil {
		return err
	}
	defer os.RemoveAll(clone.GradingDir)

	return grade(jobId, clone.AssessmentDir, clone.TestDir, reporter, logger)
}

func grade(jobId string, assessmentDir string, testDir string, reporter playwright.GradingJobReporter, logger *slog.Logger) error {
	reporter.OnInstallStart(jobId)
	installOut, err := builder.InstallAssessmentDeps(assessmentDir)
	reporter.OnInstallEnd(jobId, installOut, err)
	if err != nil {
		return err
	}

	reporter.OnBuildStart(jobId)
	buildOut, err := builder.BuildAssessment(assessmentDir)
	reporter.OnBuildEnd(jobId, buildOut, err)
	if err != nil {
		return err
	}

	port, stop, err := serve.ServeAssessment(filepath.Join(assessmentDir, "dist"))
	if err != nil {
		reporter.OnServe(jobId, err)
		logger.Error("serve failed", "port", port, "err", err)
		return err
	}
	defer stop()

	if err := util.WaitForPort(port, 5*time.Second); err != nil {
		reporter.OnServe(jobId, err)
		logger.Error("file server did not respond within timeout", "port", port)
		return err
	}

	reporter.OnServe(jobId, nil)

	if err := playwright.RunPlaywrightTests(jobId, testDir, port, reporter); err != nil {
		logger.Error("playwright tests failed", "jobId", jobId, "err", err)
		return err
	}

	logger.Info("grading complete", "jobId", jobId)
	return nil
}
