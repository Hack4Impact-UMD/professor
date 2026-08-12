package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/Hack4Impact-UMD/professor/reporter"
	"github.com/Hack4Impact-UMD/professor/routes/grade"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var gradeLogFile string

var gradeCmd = &cobra.Command{
	Use:   "grade <assessment_repo_path> <test_repo_path>",
	Short: "Run a grading job from the CLI",
	Long: `Runs Playwright tests defined in the specified test repo against the specified assessment repo locally. Results are printed to the terminal.

Both arguments are local filesystem paths to directories containing the assessment and test repositories.

Example:
  professor grade ./student-submission ./assignment-tests`,
	Args: cobra.ExactArgs(2),
	RunE: runGrade,
}

func init() {
	gradeCmd.Flags().StringVar(&gradeLogFile, "log-file", "", "Write logs to this file instead of discarding them")
	rootCmd.AddCommand(gradeCmd)
}

func runGrade(cmd *cobra.Command, args []string) error {
	assessmentRepoPath := args[0]
	testRepoPath := args[1]

	jobId := uuid.New().String()

	rep := &reporter.CLIReporter{}
	defer rep.Wait()

	var logger *slog.Logger
	if gradeLogFile != "" {
		f, err := os.OpenFile(gradeLogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("could not open log file: %w", err)
		}
		defer f.Close()
		logger = slog.New(slog.NewTextHandler(f, nil))
	} else {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	if err := grade.RunGradingJobLocal(jobId, assessmentRepoPath, testRepoPath, rep, logger); err != nil {
		return fmt.Errorf("grading failed: %w", err)
	}

	return nil
}
