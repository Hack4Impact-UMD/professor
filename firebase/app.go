package firebase

import (
	"context"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func GetFirebaseApp(useEmulators bool) (*firebase.App, error) {
	var opt option.ClientOption = nil

	if os.Getenv("PROJECT_ID") == "" {
		log.Fatalf("PROJECT_ID not found in env")
	}

	if useEmulators {
		opt = option.WithoutAuthentication()
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	}

	cfg := firebase.Config{
		ProjectID: os.Getenv("PROJECT_ID"),
	}

	app, err := firebase.NewApp(context.Background(), &cfg, opt)
	if err != nil {
		log.Fatalf("Failed to init firebase app: %v", err)
	}

	return app, nil
}
