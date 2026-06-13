package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Hack4Impact-UMD/professor/firebase"
	"github.com/Hack4Impact-UMD/professor/routes/grade"
	"github.com/Hack4Impact-UMD/professor/routes/health"
	"github.com/spf13/cobra"
)

var servePort string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP grading server",
	Long: `Starts the HTTP server that listens for grading job requests.
Results are reported to Firestore. Requires Firebase credentials to be configured.

The port can be set via --port, the PORT environment variable, or defaults to 8000.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringVarP(&servePort, "port", "p", "", "Port to listen on (overrides $PORT, default 8000)")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	port := servePort
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8000"
		log.Printf("Defaulting to port %s", port)
	}

	app, err := firebase.GetFirebaseApp(os.Getenv("DEV") == "true")
	if err != nil {
		return fmt.Errorf("could not init firebase app: %w", err)
	}

	fsClient, err := firebase.GetFirestoreClient(app)
	if err != nil {
		return fmt.Errorf("could not get firestore client: %w", err)
	}
	defer fsClient.Close()

	mux := http.NewServeMux()
	health.RegisterRoutes(mux)
	grade.RegisterHandlers(mux, fsClient)

	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Listening on port %s", port)
	return server.ListenAndServe()
}
