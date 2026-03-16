package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Hack4Impact-UMD/professor/routes/health"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("warn: could not load .env")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
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
