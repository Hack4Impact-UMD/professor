package reporter

import (
	"errors"

	"cloud.google.com/go/firestore"
	"github.com/Hack4Impact-UMD/professor/db"
	"github.com/Hack4Impact-UMD/professor/firebase"
	"github.com/Hack4Impact-UMD/professor/util"
)

const (
	maxLogBytes        = 50 * 1024 // 50KB 
	maxTestOutputBytes = 10 * 1024 // 10KB 

	collectionPublic   = "gradingJobs"
	collectionInternal = "gradingJobsInternal"
)

type FirestoreReporter struct {
	fsClient *firestore.Client
	tests    map[string][]db.TestResult
}

func NewFirestoreReporter(fsClient *firestore.Client) (*FirestoreReporter, error) {
	if fsClient == nil {
		return nil, errors.New("fsClient argument for reporter is nil")
	}

	return &FirestoreReporter{
		fsClient: fsClient,
		tests:    make(map[string][]db.TestResult),
	}, nil
}

func (r *FirestoreReporter) updatePublicDoc(jobId string, data map[string]any) error {
	return firebase.UpdateDoc(r.fsClient, collectionPublic, jobId, data)
}

func (r *FirestoreReporter) updateInternalDoc(jobId string, data map[string]any) error {
	return firebase.UpdateDoc(r.fsClient, collectionInternal, jobId, data)
}

func truncateLog(log string, maxBytes int) string {
	if len(log) <= maxBytes {
		return log
	}
	return "Tail:\n" + log[len(log)-maxBytes:]
}

func (r *FirestoreReporter) OnGradeStart(jobId string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusPending,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnCloneStart(jobId, assessmentRepo, testRepo string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusCloning,
		"updated": firestore.ServerTimestamp,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"testRepo": testRepo,
	})
}

func (r *FirestoreReporter) OnCloneEnd(jobId, assessmentRepo, testRepo string, err error) {
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":    db.StatusFailed,
			"error":     err.Error(),
			"completed": firestore.ServerTimestamp,
			"updated":   firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error": err.Error(),
		})

		return
	}
	
	_ = r.updatePublicDoc(jobId, map[string]any{
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnInstallStart(jobId string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusInstalling,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnInstallEnd(jobId, out string, err error) {
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":    db.StatusFailed,
			"error":     err.Error(),
			"completed": firestore.ServerTimestamp,
			"updated":   firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error":      err.Error(),
			"installLog": truncateLog(out, maxLogBytes),
		})

		return
	}
	
	_ = r.updatePublicDoc(jobId, map[string]any{
		"updated": firestore.ServerTimestamp,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"installLog": truncateLog(out, maxLogBytes),
	})
}

func (r *FirestoreReporter) OnBuildStart(jobId string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusBuilding,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnBuildEnd(jobId, out string, err error) {
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":    db.StatusFailed,
			"error":     err.Error(),
			"completed": firestore.ServerTimestamp,
			"updated":   firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error":    err.Error(),
			"buildLog": truncateLog(out, maxLogBytes),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"updated": firestore.ServerTimestamp,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"buildLog": truncateLog(out, maxLogBytes),
	})
}

func (r *FirestoreReporter) OnServe(jobId string, err error) {
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":    db.StatusFailed,
			"error":     err.Error(),
			"completed": firestore.ServerTimestamp,
			"updated":   firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error": err.Error(),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusServing,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnTestingStart(jobId string, suites []string, err error) {
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":    db.StatusFailed,
			"error":     err.Error(),
			"completed": firestore.ServerTimestamp,
			"updated":   firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error": err.Error(),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusTesting,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnTestStart(jobId, suite, testName string) {
	// No-op rn as it would incur a lot of extra writes, but could update timestamp later
}

func (r *FirestoreReporter) OnTestEnd(jobId, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {
	result := db.TestResult{
		Suite:      suite,
		TestName:   testName,
		Passed:     passed,
		Stdout:     truncateLog(stdout, maxTestOutputBytes),
		Stderr:     truncateLog(stderr, maxTestOutputBytes),
		Errors:     testErrors,
		DurationMs: durationMs,
		Points:     0, // TODO: implement point extraction to fill this
	}

	_ = r.updateInternalDoc(jobId, map[string]any{
		"tests": map[string]any{
			suite: firestore.ArrayUnion(result),
		},
	})

	publicTestUpdates := map[string]any{
		"suiteName": suite,
		"total":     firestore.Increment(1),
		"durationMs": firestore.Increment(durationMs),
	}

	if passed {
		publicTestUpdates["passed"] = firestore.Increment(1)
	} else {
		publicTestUpdates["failed"] = firestore.Increment(1)
	}

	publicTestUpdates["points"] = firestore.Increment(result.Points)
	publicTestUpdates["totalPoints"] = firestore.Increment(result.Points)

	_ = r.updatePublicDoc(jobId, map[string]any{
		"completedTests": firestore.Increment(1),
		"updated":        firestore.ServerTimestamp,
		"publicTests": map[string]any{
			suite: publicTestUpdates,
		},
	})
}

func (r *FirestoreReporter) OnTestingEnd(jobId string, err error) {
	data := map[string]any{
		"updated":   firestore.ServerTimestamp,
		"completed": firestore.ServerTimestamp,
	}

	if err != nil {
		data["status"] = db.StatusFailed
		data["error"] = err.Error()

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error": err.Error(),
		})
	} else {
		data["status"] = db.StatusCompleted
	}

	_ = r.updatePublicDoc(jobId, data)
}

var _ util.GradingJobReporter = (*FirestoreReporter)(nil)
