package cmd

import (
	"fmt"
	"os"

	"github.com/Hack4Impact-UMD/professor/serve"
	"github.com/spf13/cobra"
)

var serveStaticPort int = -1

var serveStatic = &cobra.Command{
	Use:   "serve-static",
	Short: "Start the HTTP SPA server",
	Long:  `Starts the HTTP SPA server that served the built assessment SPA files. Used within the grading container`,
	RunE:  runServeStatic,
}

func init() {
	serveStatic.Flags().IntVarP(&serveStaticPort, "port", "p", -1, "Port to listen on")
	rootCmd.AddCommand(serveStatic)
}

func runServeStatic(cmd *cobra.Command, args []string) error {
	port := serveStaticPort
	if port == -1 {
		p, err := serve.GetFreePort()
		if err != nil {
			return err
		} else {
			port = p
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", port)
	err = serve.ServeAssessmentBlocking(port, wd)

	return err
}
