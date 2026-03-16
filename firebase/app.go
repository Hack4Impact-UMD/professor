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

	if useEmulators {
		opt = option.WithoutAuthentication()
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
	}

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatalf("Failed to init firebase app: %v", err)
		return &firebase.App{}, err
	}

	return app, nil
}
