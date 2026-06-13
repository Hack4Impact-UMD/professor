package firebase

import (
	"context"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
)

func GetFirestoreClient(app *firebase.App) (*firestore.Client, error) {
	return app.Firestore(context.Background())
}

func UpdateDoc(client *firestore.Client, collection, docId string, data map[string]any) error {
	ctx := context.Background()
	_, err := client.Collection(collection).Doc(docId).Set(ctx, data, firestore.MergeAll)
	return err
}

func UpdateDocFields(client *firestore.Client, collection, docId string, updates []firestore.Update) error {
	ctx := context.Background()
	_, err := client.Collection(collection).Doc(docId).Update(ctx, updates)
	return err
}
