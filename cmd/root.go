package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "professor",
	Short: "A grading server for web development assessments",
	Long:  `Professor is Hack4Impact-UMD's technical assessment autograder. It runs Playwright tests against assessment submissions and reports results to Firestore (when run as a grading worker) or the terminal (when run locally)`,
}

func Execute() error {
	return rootCmd.Execute()
}
