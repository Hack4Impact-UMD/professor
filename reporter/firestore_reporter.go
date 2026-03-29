package reporter

import (
	"errors"
	"sync"

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

func (r *FirestoreReporter) OnGradeStart(jobId string) {
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusPending,
		"updated": firestore.ServerTimestamp,
	})
}

// TODO: implement all of these; here for now to satisfy compiler

func (r *FirestoreReporter) OnCloneStart(jobId string, assessmentRepo string, testRepo string) {}

func (r *FirestoreReporter) OnCloneEnd(jobId string, assessmentRepo string, testRepo string, err error) {}

func (r *FirestoreReporter) OnInstallStart(jobId string) {}

func (r *FirestoreReporter) OnInstallEnd(jobId string, out string, err error) {}

func (r *FirestoreReporter) OnBuildStart(jobId string) {}

func (r *FirestoreReporter) OnBuildEnd(jobId string, out string, err error) {}

func (r *FirestoreReporter) OnServe(jobId string, err error) {}

func (r *FirestoreReporter) OnTestingStart(jobId string, suites []string, err error) {}

func (r *FirestoreReporter) OnTestStart(jobId string, suite string, testName string) {}

func (r *FirestoreReporter) OnTestEnd(jobId string, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {}

func (r *FirestoreReporter) OnTestingEnd(jobId string, err error) {}

var _ util.GradingJobReporter = (*FirestoreReporter)(nil)
