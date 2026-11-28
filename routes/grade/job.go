package grade

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/Hack4Impact-UMD/professor/builder"
	"github.com/Hack4Impact-UMD/professor/git"
	"github.com/Hack4Impact-UMD/professor/playwright"
	"github.com/Hack4Impact-UMD/professor/sandbox"
	"github.com/Hack4Impact-UMD/professor/serve"
	"github.com/Hack4Impact-UMD/professor/util"
	"golang.org/x/sync/errgroup"
)

type cloneResult struct {
	// AssessmentDir and TestDir are deliberately created under separate temp
	// roots (not siblings under a shared parent). Untrusted assessment code runs
	// during `pnpm install`/`vite build`; keeping the trusted test tree out of a
	// predictable sibling path removes the naive `../tests` traversal that would
	// let that code overwrite spec files or playwright.config.ts to forge grades.
	// The assessment is only ever exposed to the tests as static files over HTTP,
	// never imported into the test process.
	AssessmentDir string
	TestDir       string
}

func (c cloneResult) cleanup() {
	if c.AssessmentDir != "" {
		os.RemoveAll(c.AssessmentDir)
	}
	if c.TestDir != "" {
		os.RemoveAll(c.TestDir)
	}
}

const MAX_REPO_SIZE_MB = 10

func cloneWithSizeCheck(ctx context.Context, sb sandbox.Sandbox, path string, dest string, pat string) error {
	size, err := git.GetRepoSizeKB(path)

	if err != nil {
		return fmt.Errorf("failed to get repo details: %w", err)
	}

	if size > MAX_REPO_SIZE_MB*1024 {
		return errors.New("repo too large")
	}

	return git.CloneRepo(ctx, sb, path, dest, pat)
}

func cloneRepos(ctx context.Context, sb sandbox.Sandbox, jobId string, assessmentRepoPath string, testRepoPath string, logger *slog.Logger) (cloneResult, error) {
	pat := os.Getenv("GITHUB_PAT")

	assessmentDir := fmt.Sprintf("job-assessment-%v", jobId)
	testDir := fmt.Sprintf("job-tests-%v", jobId)

	result := cloneResult{AssessmentDir: assessmentDir, TestDir: testDir}

	wg := errgroup.Group{}

	logger.Info("cloning repos", "jobId", jobId, "assessmentRepo", assessmentRepoPath, "testRepo", testRepoPath, "assessmentDir", assessmentDir, "testDir", testDir)

	wg.Go(func() error {
		if err := cloneWithSizeCheck(ctx, sb, assessmentRepoPath, assessmentDir, pat); err != nil {
			logger.Error("assessment clone failed", "jobId", jobId, "repo", assessmentRepoPath, "err", err)
			return err
		}
		return nil
	})
	wg.Go(func() error {
		if err := cloneWithSizeCheck(ctx, sb, testRepoPath, testDir, pat); err != nil {
			logger.Error("test repo clone failed", "jobId", jobId, "repo", testRepoPath, "err", err)
			return err
		}
		return nil
	})

	if err := wg.Wait(); err != nil {
		result.cleanup()
		return cloneResult{}, err
	}

	logger.Info("repos cloned", "jobId", jobId, "assessmentDir", assessmentDir, "testDir", testDir)

	return result, nil
}

func RunGradingJobLocal(jobId string, assessmentRepoDir string, testRepoDir string, reporter playwright.GradingJobReporter, logger *slog.Logger) error {
	reporter.OnGradeStart(jobId)
	return grade(jobId, assessmentRepoDir, testRepoDir, reporter, logger)
}

func RunGradingJob(ctx context.Context, jobId string, assessmentRepoPath string, testRepoPath string, reporter playwright.GradingJobReporter, logger *slog.Logger) error {
	logger.Info("running grading job", "jobId", jobId)
	reporter.OnGradeStart(jobId)

	sandboxDir := os.TempDir()
	mount := []sandbox.MountPoint{
		sandbox.MountPoint{
			SourcePath: sandboxDir,
			DestPath:   sandboxDir,
			ReadOnly:   false,
		},
	}

	var sb sandbox.Sandbox = nil
	name := fmt.Sprintf("sandbox-%v", jobId)
	if os.Getenv("DEV") == "true" {
		dockerSandbox := sandbox.NewDocker(name, "h4i-umd/professor:latest", sandboxDir, mount, false)
		sb = &dockerSandbox
	} else {
		crSandbox := sandbox.NewCloudRun(name, sandboxDir, mount, false)
		sb = &crSandbox
	}
	defer sb.Destroy(ctx)

	reporter.OnCloneStart(jobId, assessmentRepoPath, testRepoPath)
	clone, err := cloneRepos(ctx, sb, jobId, assessmentRepoPath, testRepoPath, logger)
	reporter.OnCloneEnd(jobId, assessmentRepoPath, testRepoPath, err)
	if err != nil {
		return err
	}
	defer clone.cleanup()

	return grade(ctx, sb, jobId, clone.AssessmentDir, clone.TestDir, reporter, logger)
}

func installDeps(ctx context.Context, sb sandbox.Sandbox, assessmentDir string, testDir string, logger *slog.Logger) (string, string, error) {
	dirs := [2]string{assessmentDir, testDir}
	outs := [2]string{}

	var wg errgroup.Group
	for i, dir := range dirs {
		wg.Go(func() error {
			logger.Info("installing deps", "dir", dir)
			out, err := builder.InstallDeps(ctx, sb, dir)
			outs[i] = out
			if err != nil {
				logger.Error("dep install failed", "dir", dir, "output", out, "err", err)
			}
			return err
		})
	}

	err := wg.Wait()
	return outs[0], outs[1], err
}

func grade(ctx context.Context, sb sandbox.Sandbox, jobId string, assessmentDir string, testDir string, reporter playwright.GradingJobReporter, logger *slog.Logger) error {
	logger.Info("installing dependencies", "jobId", jobId)
	reporter.OnInstallStart(jobId)

	assessmentInstallOut, testInstallOut, err := installDeps(ctx, sb, assessmentDir, testDir, logger)
	reporter.OnInstallEnd(jobId, fmt.Sprintf("Assessment:\n%s\nTests:\n%s", assessmentInstallOut, testInstallOut), err)
	if err != nil {
		return err
	}
	logger.Info("dependencies installed", "jobId", jobId)

	logger.Info("building assessment", "jobId", jobId)
	reporter.OnBuildStart(jobId)
	buildOut, err := builder.BuildAssessment(ctx, sb, assessmentDir, testDir)
	reporter.OnBuildEnd(jobId, buildOut, err)
	if err != nil {
		logger.Error("build failed", "jobId", jobId, "output", buildOut, "err", err)
		return err
	}
	logger.Info("assessment built", "jobId", jobId)

	port, stop, err := serve.ServeAssessment(ctx, sb, filepath.Join(assessmentDir, "dist"))
	if err != nil {
		reporter.OnServe(jobId, err)
		logger.Error("serve failed", "jobId", jobId, "err", err)
		return err
	}
	defer stop()
	logger.Info("serving assessment", "jobId", jobId, "port", port)

	if err := util.WaitForPort(ctx, sb, port, 5*time.Second); err != nil {
		reporter.OnServe(jobId, err)
		logger.Error("file server did not respond within timeout", "jobId", jobId, "port", port)
		return err
	}
	logger.Info("assessment server ready", "jobId", jobId, "port", port)

	reporter.OnServe(jobId, nil)

	if err := playwright.RunPlaywrightTests(ctx, sb, jobId, testDir, port, reporter, logger); err != nil {
		logger.Error("playwright tests failed", "jobId", jobId, "err", err)
		return err
	}

	logger.Info("grading complete", "jobId", jobId)
	return nil
}
