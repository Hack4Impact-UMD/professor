package reporter

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/Hack4Impact-UMD/professor/db"
	"github.com/Hack4Impact-UMD/professor/firebase"
)

func setupTest(t *testing.T) (context.Context, *firestore.Client, *FirestoreReporter) {
	ctx := context.Background()

	app, err := firebase.GetFirebaseApp(true)
	if err != nil {
		t.Fatalf("Failed to create Firebase app: %v", err)
	}

	client, err := firebase.GetFirestoreClient(app)
	if err != nil {
		t.Fatalf("Failed to create Firestore client: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	reporter, err := NewFirestoreReporter(client)
	if err != nil {
		t.Fatalf("Failed to create reporter: %v", err)
	}

	return ctx, client, reporter
}

func setupJobDocs(t *testing.T, ctx context.Context, client *firestore.Client, jobId string) {
	_, err := client.Collection(collectionPublic).Doc(jobId).Set(ctx, map[string]any{
		"id":     jobId,
		"status": db.StatusPending,
	})
	if err != nil {
		t.Fatalf("Failed to create test gradingJobs document: %v", err)
	}

	_, err = client.Collection(collectionInternal).Doc(jobId).Set(ctx, map[string]any{
		"id": jobId,
	})
	if err != nil {
		t.Fatalf("Failed to create test gradingJobsInternal document: %v", err)
	}
}

func cleanupJobDocs(ctx context.Context, client *firestore.Client, jobId string) {
	client.Collection(collectionPublic).Doc(jobId).Delete(ctx)
	client.Collection(collectionInternal).Doc(jobId).Delete(ctx)
}

func assertField(t *testing.T, ctx context.Context, client *firestore.Client, jobId, collection, field string, expected any) {
	t.Helper()
	doc, err := client.Collection(collection).Doc(jobId).Get(ctx)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	actual, ok := doc.Data()[field]
	if !ok {
		t.Errorf("Field %q does not exist", field)
		return
	}

	if actual != expected {
		t.Errorf("Field %q: expected %q, got %q", field, expected, actual)
	}
}

func assertFieldExists(t *testing.T, ctx context.Context, client *firestore.Client, jobId, collection, field string) {
	t.Helper()
	doc, err := client.Collection(collection).Doc(jobId).Get(ctx)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	if _, exists := doc.Data()[field]; !exists {
		t.Errorf("Field %q should exist but doesn't", field)
	}
}

func TestFirestoreReporter(t *testing.T) {
	ctx, client, reporter := setupTest(t)

	jobId := "firestore_reporter_test"
	setupJobDocs(t, ctx, client, jobId)
	
	defer cleanupJobDocs(ctx, client, jobId)

	t.Run("OnGradeStart updates status", func(t *testing.T) {
		reporter.OnGradeStart(jobId)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusPending)
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "updated")
	})

	t.Run("OnCloneStart updates status and stores testRepo", func(t *testing.T) {
		reporter.OnCloneStart(jobId, "user/assessment", "h4i/tests")

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusCloning)
		assertField(t, ctx, client, jobId, collectionInternal, "testRepo", "h4i/tests")
	})

	t.Run("OnCloneEnd with no error does nothing", func(t *testing.T) {
		reporter.OnCloneEnd(jobId, "user/assessment", "h4i/tests", nil)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusCloning)
	})

	t.Run("OnCloneEnd with error marks job as failed", func(t *testing.T) {
		testErr := errors.New("git clone failed: repository not found")
		reporter.OnCloneEnd(jobId, "user/assessment", "h4i/tests", testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
	})

	t.Run("OnInstallStart updates status", func(t *testing.T) {
		reporter.OnInstallStart(jobId)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusInstalling)
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "updated")
	})

	t.Run("OnInstallEnd with success stores log", func(t *testing.T) {
		installOutput := "npm install successful\ninstalled 42 packages"
		reporter.OnInstallEnd(jobId, installOutput, nil)

		assertField(t, ctx, client, jobId, collectionInternal, "installLog", installOutput)
	})

	t.Run("OnInstallEnd with error marks job as failed", func(t *testing.T) {
		testErr := errors.New("npm install failed: ENOENT package.json")
		installOutput := "Error: cannot find package.json"
		reporter.OnInstallEnd(jobId, installOutput, testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertField(t, ctx, client, jobId, collectionInternal, "installLog", installOutput)
	})
}
