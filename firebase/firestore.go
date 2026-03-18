package firebase

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
)

func GetFirestoreClient(app *firebase.App) (*firestore.Client, error) {
	client, err := app.Firestore(context.Background())

	if err != nil {
		log.Println("Faild to get firestore client instance:", err)
		return nil, err
	}

	return client, nil
}
