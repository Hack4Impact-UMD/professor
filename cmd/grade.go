package cmd

import (
	"fmt"
	"github.com/Hack4Impact-UMD/professor/reporter"
	"github.com/Hack4Impact-UMD/professor/routes/grade"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var gradeCmd = &cobra.Command{
	Use:   "grade <assessment_repo_path> <test_repo_path>",
	Short: "Run a grading job from the CLI",
	Long: `Clones the given GitHub repositories, builds the assessment, and runs
Playwright tests against it. Results are printed to the terminal.

Both arguments are GitHub repository paths in the form "owner/repo".

Example:
  professor grade my-org/student-submission my-org/assignment-tests`,
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

	// TODO: replace with a real CLIReporter once implemented
	rep := &reporter.CLIReporter{}

	if err := grade.RunGradingJob(jobId, assessmentRepoPath, testRepoPath, rep); err != nil {
		return fmt.Errorf("grading failed: %w", err)
	}

	return nil
}
