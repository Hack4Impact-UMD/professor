package firebase

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
)

func GetFirebaseApp(useEmulators bool) (*firebase.App, error) {
	if useEmulators {
		// TODO: implement
	}

	app, err := firebase.NewApp(context.Background(), nil)
	if err != nil {
		log.Fatalf("Failed to init firebase app: %v", err)
		return &firebase.App{}, err
	}

	return app, nil
}
