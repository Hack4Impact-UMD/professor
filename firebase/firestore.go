package firebase

import (
	"context"
	"log/slog"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
)

func GetFirestoreClient(app *firebase.App) (*firestore.Client, error) {
	return app.Firestore(context.Background())
}

func UpdateDoc(client *firestore.Client, collection, docId string, data map[string]any) error {
	ctx := context.Background()
	_, err := client.Collection(collection).Doc(docId).Set(ctx, data, firestore.MergeAll)
	if err != nil {
		slog.Error("failed to update firestore document", "collection", collection, "docId", docId, "err", err)
	}
	return err
}

func UpdateDocFields(client *firestore.Client, collection, docId string, updates []firestore.Update) error {
	ctx := context.Background()
	_, err := client.Collection(collection).Doc(docId).Update(ctx, updates)
	if err != nil {
		slog.Error("failed to update firestore document fields", "collection", collection, "docId", docId, "err", err)
	}
	return err
}
