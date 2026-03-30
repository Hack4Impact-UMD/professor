package reporter

import (
	"context"
	"errors"
	"testing"
	"time"

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

func assertUpdateTimeChanged(t *testing.T, ctx context.Context, client *firestore.Client, jobId, collection string, before time.Time) {
	t.Helper()
	doc, err := client.Collection(collection).Doc(jobId).Get(ctx)
	if err != nil {
		t.Fatalf("Failed to read document: %v", err)
	}

	updatedField, ok := doc.Data()["updated"]
	if !ok {
		t.Error("Field 'updated' does not exist")
		return
	}

	updatedTime, ok := updatedField.(time.Time)
	if !ok {
		t.Errorf("Field 'updated' is not a time.Time, got %T", updatedField)
		return
	}

	if !updatedTime.After(before) {
		t.Errorf("'updated' field should have changed: before=%v, after=%v", before, updatedTime)
	}
}

func TestFirestoreReporter(t *testing.T) {
	ctx, client, reporter := setupTest(t)

	jobId := "firestore_reporter_test"
	setupJobDocs(t, ctx, client, jobId)
	
	defer cleanupJobDocs(ctx, client, jobId)

	t.Run("OnGradeStart updates status", func(t *testing.T) {
		before := time.Now()

		reporter.OnGradeStart(jobId)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusPending)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnCloneStart updates status and stores testRepo", func(t *testing.T) {
		before := time.Now()

		reporter.OnCloneStart(jobId, "user/assessment", "h4i/tests")

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusCloning)
		assertField(t, ctx, client, jobId, collectionInternal, "testRepo", "h4i/tests")
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnCloneEnd with no error only updates timestamp", func(t *testing.T) {
		before := time.Now()

		reporter.OnCloneEnd(jobId, "user/assessment", "h4i/tests", nil)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusCloning)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnCloneEnd with error marks job as failed", func(t *testing.T) {
		before := time.Now()
		testErr := errors.New("git clone failed: repository not found")

		reporter.OnCloneEnd(jobId, "user/assessment", "h4i/tests", testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnInstallStart updates status", func(t *testing.T) {
		before := time.Now()

		reporter.OnInstallStart(jobId)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusInstalling)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnInstallEnd with success stores log", func(t *testing.T) {
		before := time.Now()
		installOutput := "npm install successful\ninstalled 42 packages"

		reporter.OnInstallEnd(jobId, installOutput, nil)

		assertField(t, ctx, client, jobId, collectionInternal, "installLog", installOutput)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnInstallEnd with error marks job as failed", func(t *testing.T) {
		before := time.Now()
		testErr := errors.New("npm install failed: ENOENT package.json")
		installOutput := "Error: cannot find package.json"

		reporter.OnInstallEnd(jobId, installOutput, testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertField(t, ctx, client, jobId, collectionInternal, "installLog", installOutput)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnBuildStart updates status", func(t *testing.T) {
		before := time.Now()

		reporter.OnBuildStart(jobId)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusBuilding)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnBuildEnd with success stores log", func(t *testing.T) {
		before := time.Now()
		buildOutput := "Build successful\nCompiled 15 files"

		reporter.OnBuildEnd(jobId, buildOutput, nil)

		assertField(t, ctx, client, jobId, collectionInternal, "buildLog", buildOutput)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnBuildEnd with error marks job as failed", func(t *testing.T) {
		before := time.Now()
		testErr := errors.New("build failed: syntax error")
		buildOutput := "Error: unexpected token"

		reporter.OnBuildEnd(jobId, buildOutput, testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertField(t, ctx, client, jobId, collectionInternal, "buildLog", buildOutput)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnServe with no error updates status to serving", func(t *testing.T) {
		before := time.Now()

		reporter.OnServe(jobId, nil)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusServing)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnServe with error marks job as failed", func(t *testing.T) {
		before := time.Now()
		testErr := errors.New("failed to start server: port already in use")

		reporter.OnServe(jobId, testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnTestingStart with no error updates status to testing", func(t *testing.T) {
		before := time.Now()

		reporter.OnTestingStart(jobId, []string{"suite1", "suite2"}, nil)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusTesting)
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnTestingStart with error marks job as failed", func(t *testing.T) {
		before := time.Now()
		testErr := errors.New("failed to discover tests")

		reporter.OnTestingStart(jobId, nil, testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnTestStart doesn't panic", func(t *testing.T) {
		reporter.OnTestStart(jobId, "suite1", "test1")
	})

	t.Run("OnTestEnd with passing test writes result and updates aggregations", func(t *testing.T) {
		before := time.Now()

		reporter.OnTestEnd(jobId, "suite1", "test1", true, "stdout output", "stderr output", []string{}, 150, nil)

		doc, err := client.Collection(collectionInternal).Doc(jobId).Get(ctx)
		if err != nil {
			t.Fatalf("Failed to read internal document: %v", err)
		}

		tests := doc.Data()["tests"].(map[string]any)
		suite1Tests := tests["suite1"].([]any)
		if len(suite1Tests) != 1 {
			t.Fatalf("Expected 1 test in suite1, got %d", len(suite1Tests))
		}

		testResult := suite1Tests[0].(map[string]any)
		if testResult["testName"] != "test1" || testResult["passed"] != true {
			t.Errorf("Test result incorrect: %v", testResult)
		}

		assertField(t, ctx, client, jobId, collectionPublic, "completedTests", int64(1))
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)

		publicDoc, err := client.Collection(collectionPublic).Doc(jobId).Get(ctx)
		if err != nil {
			t.Fatalf("Failed to read public document: %v", err)
		}

		publicTests := publicDoc.Data()["publicTests"].(map[string]any)
		suite1 := publicTests["suite1"].(map[string]any)

		if suite1["passed"] != int64(1) || suite1["total"] != int64(1) || suite1["durationMs"] != int64(150) {
			t.Errorf("Suite aggregations incorrect: %v", suite1)
		}
	})

	t.Run("OnTestEnd with failing test updates aggregations correctly", func(t *testing.T) {
		before := time.Now()

		reporter.OnTestEnd(jobId, "suite1", "test2", false, "stdout", "stderr", []string{"error1"}, 200, nil)

		doc, err := client.Collection(collectionInternal).Doc(jobId).Get(ctx)
		if err != nil {
			t.Fatalf("Failed to read internal document: %v", err)
		}

		tests := doc.Data()["tests"].(map[string]any)
		suite1Tests := tests["suite1"].([]any)
		if len(suite1Tests) != 2 {
			t.Fatalf("Expected 2 tests in suite1, got %d", len(suite1Tests))
		}

		testResult := suite1Tests[1].(map[string]any)
		if testResult["testName"] != "test2" || testResult["passed"] != false {
			t.Errorf("Test result incorrect: %v", testResult)
		}

		assertField(t, ctx, client, jobId, collectionPublic, "completedTests", int64(2))
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)

		publicDoc, err := client.Collection(collectionPublic).Doc(jobId).Get(ctx)
		if err != nil {
			t.Fatalf("Failed to read public document: %v", err)
		}

		publicTests := publicDoc.Data()["publicTests"].(map[string]any)
		suite1 := publicTests["suite1"].(map[string]any)

		if suite1["passed"] != int64(1) || suite1["failed"] != int64(1) || suite1["total"] != int64(2) || suite1["durationMs"] != int64(350) {
			t.Errorf("Suite aggregations incorrect: %v", suite1)
		}
	})

	t.Run("OnTestingEnd with no error marks job as completed", func(t *testing.T) {
		before := time.Now()

		reporter.OnTestingEnd(jobId, nil)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusCompleted)
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})

	t.Run("OnTestingEnd with error marks job as failed", func(t *testing.T) {
		before := time.Now()
		testErr := errors.New("playwright tests crashed")

		reporter.OnTestingEnd(jobId, testErr)

		assertField(t, ctx, client, jobId, collectionPublic, "status", db.StatusFailed)
		assertField(t, ctx, client, jobId, collectionPublic, "error", testErr.Error())
		assertFieldExists(t, ctx, client, jobId, collectionPublic, "completed")
		assertField(t, ctx, client, jobId, collectionInternal, "error", testErr.Error())
		assertUpdateTimeChanged(t, ctx, client, jobId, collectionPublic, before)
	})
}
