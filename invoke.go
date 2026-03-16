package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Hack4Impact-UMD/professor/firebase"
	"github.com/Hack4Impact-UMD/professor/routes/grade"
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

	app, err := firebase.GetFirebaseApp(os.Getenv("DEV") == "true")
	if err != nil {
		log.Fatalf("Could not init firebase app: %v", err)
		return
	}

	fsClient, err := firebase.GetFirestoreClient(app)
	if err != nil {
		log.Fatalf("Could not get firestore client instance: %v", err)
		return
	}

	handler := http.NewServeMux()
	health.RegisterRoutes(handler)
	grade.RegisterHandlers(handler, fsClient)

	server := http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	log.Printf("Listening on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
