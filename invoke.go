package main

import (
	"github.com/Hack4Impact-UMD/professor/routes/health"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	handler := http.NewServeMux()
	health.RegisterRoutes(handler)

	server := http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	log.Printf("Listening on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
