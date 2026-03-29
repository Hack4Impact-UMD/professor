package reporter

import (
	"context"
	"testing"

	"github.com/Hack4Impact-UMD/professor/db"
	"github.com/Hack4Impact-UMD/professor/firebase"
)

func TestFirestoreReporter(t *testing.T) {

	// === Init Firebase stuff ===

	ctx := context.Background()

	app, err := firebase.GetFirebaseApp(true)
	if err != nil {
		t.Fatalf("Failed to create Firebase app: %v", err)
	}

	client, err := firebase.GetFirestoreClient(app)
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	defer client.Close()

	reporter := NewFirestoreReporter(client)
	jobId := "firestore_reporter_test"
	initialData := map[string]any{
		"id":         jobId,
		"status":     db.StatusQueued,
		"repoURL":    "https://github.com/test/repo",
		"responseId": "abc123",
	}

	// === Mock job creation by Express backend ===

	_, err = client.Collection(collectionPublic).Doc(jobId).Set(ctx, initialData)
	if err != nil {
		t.Fatalf("Failed to create initial test document: %v", err)
	}

	// === TESTS ===

	t.Run("OnGradeStart updates status", func(t *testing.T) {
		reporter.OnGradeStart(jobId)

		doc, err := client.Collection(collectionPublic).Doc(jobId).Get(ctx)
		if err != nil {
			t.Fatalf("Failed to read document: %v", err)
		}
		data := doc.Data()

		status, ok := data["status"].(string)
		if !ok {
			t.Fatalf("status field missing or wrong type")
		}
		if status != db.StatusPending {
			t.Errorf("Expected status %q, got %q", db.StatusPending, status)
		}
		
		if _, ok := data["updated"]; !ok {
			t.Error("updated timestamp not set")
		}
	})

	// === Validate unchanged fields after all tests ===

	finalDoc, err := client.Collection(collectionPublic).Doc(jobId).Get(ctx)
	if err != nil {
		t.Fatalf("Failed to read document for invariants: %v", err)
	}
	finalData := finalDoc.Data()

	if initialData["id"] != finalData["id"] {
		t.Error("id field was lost")
	}
	if initialData["repoURL"] != finalData["repoURL"] {
		t.Error("repoURL field was lost")
	}
	if initialData["responseId"] != finalData["responseId"] {
		t.Error("responseId field was lost")
	}

	// === Cleanup ===

	client.Collection(collectionPublic).Doc(jobId).Delete(ctx)
}
