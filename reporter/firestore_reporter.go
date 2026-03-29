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

func (r *FirestoreReporter) OnCloneStart(jobId string, assessmentRepo string, testRepo string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusCloning,
		"updated": firestore.ServerTimestamp,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"testRepo": testRepo,
	})
}

func (r *FirestoreReporter) OnCloneEnd(jobId string, assessmentRepo string, testRepo string, err error) {
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
	}
}

func (r *FirestoreReporter) OnInstallStart(jobId string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusInstalling,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnInstallEnd(jobId string, out string, err error) {
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
	} else {
		_ = r.updateInternalDoc(jobId, map[string]any{
			"installLog": truncateLog(out, maxLogBytes),
		})
	}
}

func (r *FirestoreReporter) OnBuildStart(jobId string) {}

func (r *FirestoreReporter) OnBuildEnd(jobId string, out string, err error) {}

func (r *FirestoreReporter) OnServe(jobId string, err error) {}

func (r *FirestoreReporter) OnTestingStart(jobId string, suites []string, err error) {}

func (r *FirestoreReporter) OnTestStart(jobId string, suite string, testName string) {}

func (r *FirestoreReporter) OnTestEnd(jobId string, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {}

func (r *FirestoreReporter) OnTestingEnd(jobId string, err error) {}

var _ util.GradingJobReporter = (*FirestoreReporter)(nil)
