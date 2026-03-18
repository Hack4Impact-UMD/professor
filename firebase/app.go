package firebase

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
)

func GetFirebaseApp(useEmulators bool) (*firebase.App, error) {
	if os.Getenv("PROJECT_ID") == "" {
		log.Fatalf("PROJECT_ID not found in env")
	}

	if useEmulators {
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	}

	cfg := firebase.Config{
		ProjectID: os.Getenv("PROJECT_ID"),
	}

	app, err := firebase.NewApp(context.Background(), &cfg)
	if err != nil {
		log.Fatalf("Failed to init firebase app: %v", err)
	}

	return app, nil
}
