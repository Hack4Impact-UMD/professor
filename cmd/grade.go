package cmd

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/Hack4Impact-UMD/professor/reporter"
	"github.com/Hack4Impact-UMD/professor/routes/grade"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var gradeCmd = &cobra.Command{
	Use:   "grade <assessment_repo_path> <test_repo_path>",
	Short: "Run a grading job from the CLI",
	Long: `Runs Playwright tests defined in the specified test repo against the specified assessment repo locally. Results are printed to the terminal.

Both arguments are GitHub repository paths in the form "owner/repo".

Example:
  professor grade ./student-submission ./assignment-tests`,
	Args: cobra.ExactArgs(2),
	RunE: runGrade,
}

func init() {
	rootCmd.AddCommand(gradeCmd)
}

func runGrade(cmd *cobra.Command, args []string) error {
	assessmentRepoPath := args[0]
	testRepoPath := args[1]

	jobId := uuid.New().String()

	rep := &reporter.CLIReporter{}
	silent := slog.New(slog.NewTextHandler(io.Discard, nil))

	if err := grade.RunGradingJobLocal(jobId, assessmentRepoPath, testRepoPath, rep, silent); err != nil {
		return fmt.Errorf("grading failed: %w", err)
	}

	return nil
}
