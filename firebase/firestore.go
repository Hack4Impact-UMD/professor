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
		log.Println("ERROR: Failed to get firestore client instance:", err)
		return nil, err
	}

	return client, nil
}

func UpdateDoc(client *firestore.Client, collection, docId string, data map[string]any) error {
	ctx := context.Background()
	_, err := client.Collection(collection).Doc(docId).Set(ctx, data, firestore.MergeAll)

	if err != nil {
		log.Printf("ERROR: Failed to update %s/%s: %v", collection, docId, err)
		return err
	}

	return nil
}
