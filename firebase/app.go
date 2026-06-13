package firebase

import (
	"context"
	"log/slog"
	"os"

	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/option"
)

func GetFirebaseApp(useEmulators bool) (*firebase.App, error) {
	if os.Getenv("PROJECT_ID") == "" {
		slog.Error("PROJECT_ID not found in env")
		os.Exit(1)
	}

	cfg := firebase.Config{
		ProjectID: os.Getenv("PROJECT_ID"),
	}

	if useEmulators {
		slog.Info("using emulators")
		opt := option.WithoutAuthentication()
		os.Setenv("FIRESTORE_EMULATOR_HOST", "localhost:8080")
		app, err := firebase.NewApp(context.Background(), &cfg, opt)
		if err != nil {
			slog.Error("failed to init firebase app", "err", err)
			os.Exit(1)
		}
		return app, nil
	}

	app, err := firebase.NewApp(context.Background(), &cfg)
	if err != nil {
		slog.Error("failed to init firebase app", "err", err)
		os.Exit(1)
	}

	return app, nil
}
