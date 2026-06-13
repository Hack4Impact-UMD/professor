package reporter

import (
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/Hack4Impact-UMD/professor/db"
	"github.com/Hack4Impact-UMD/professor/firebase"
	"github.com/Hack4Impact-UMD/professor/playwright"
)

const (
	maxLogBytes        = 50 * 1024 // 50KB
	maxTestOutputBytes = 10 * 1024 // 10KB

	collectionPublic   = "grading-jobs-public"
	collectionInternal = "grading-jobs-internal"
)

type FirestoreReporter struct {
	fsClient      *firestore.Client
	publicTestSet map[string]map[string]bool // suite -> testName -> isPublic
	testPoints    map[string]map[string]int  // suite -> testName -> points
	cloneStart    time.Time
	installStart  time.Time
	buildStart    time.Time
	testingStart  time.Time
}

func NewFirestoreReporter(fsClient *firestore.Client) (*FirestoreReporter, error) {
	if fsClient == nil {
		return nil, errors.New("fsClient argument for reporter is nil")
	}

	return &FirestoreReporter{
		fsClient:      fsClient,
		publicTestSet: make(map[string]map[string]bool),
		testPoints:    make(map[string]map[string]int),
	}, nil
}

func suiteTotalPoints(suite playwright.TestSuite) int {
	sum := 0
	for _, t := range suite.Tests {
		sum += t.Points
	}

	return sum
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
	r.cloneStart = time.Now()
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusCloning,
		"updated": firestore.ServerTimestamp,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"testRepo": testRepo,
	})
}

func (r *FirestoreReporter) OnCloneEnd(jobId, assessmentRepo, testRepo string, err error) {
	cloneDurationMs := time.Since(r.cloneStart).Milliseconds()
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":          db.StatusFailed,
			"error":           err.Error(),
			"cloneDurationMs": cloneDurationMs,
			"completed":       firestore.ServerTimestamp,
			"updated":         firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error": err.Error(),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"cloneDurationMs": cloneDurationMs,
		"updated":         firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnInstallStart(jobId string) {
	r.installStart = time.Now()
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusInstalling,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnInstallEnd(jobId, out string, err error) {
	installDurationMs := time.Since(r.installStart).Milliseconds()
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":            db.StatusFailed,
			"error":             err.Error(),
			"installDurationMs": installDurationMs,
			"completed":         firestore.ServerTimestamp,
			"updated":           firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error":      err.Error(),
			"installLog": truncateLog(out, maxLogBytes),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"installDurationMs": installDurationMs,
		"updated":           firestore.ServerTimestamp,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"installLog": truncateLog(out, maxLogBytes),
	})
}

func (r *FirestoreReporter) OnBuildStart(jobId string) {
	r.buildStart = time.Now()
	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":  db.StatusBuilding,
		"updated": firestore.ServerTimestamp,
	})
}

func (r *FirestoreReporter) OnBuildEnd(jobId, out string, err error) {
	buildDurationMs := time.Since(r.buildStart).Milliseconds()
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":          db.StatusFailed,
			"error":           err.Error(),
			"buildDurationMs": buildDurationMs,
			"completed":       firestore.ServerTimestamp,
			"updated":         firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error":    err.Error(),
			"buildLog": truncateLog(out, maxLogBytes),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"buildDurationMs": buildDurationMs,
		"updated":         firestore.ServerTimestamp,
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

func (r *FirestoreReporter) OnTestingStart(jobId string, repo playwright.TestRepo, err error) {
	r.testingStart = time.Now()
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

	publicTestMap := make(map[string]map[string]db.TestResult)
	testMap := make(map[string]map[string]db.TestResult)
	suiteResults := make(map[string]db.SuiteResult)

	r.publicTestSet = make(map[string]map[string]bool)
	r.testPoints = make(map[string]map[string]int)

	for _, suite := range repo.Suites {
		sr := db.SuiteResult{
			SuiteName:   suite.Name,
			Passed:      0,
			Failed:      0,
			Total:       len(suite.Tests),
			DurationMs:  0,
			TotalPoints: suiteTotalPoints(suite),
		}

		suiteResults[suite.Name] = sr
		publicTestMap[suite.Name] = make(map[string]db.TestResult)
		testMap[suite.Name] = make(map[string]db.TestResult)
		r.publicTestSet[suite.Name] = make(map[string]bool)
		r.testPoints[suite.Name] = make(map[string]int)

		for _, test := range suite.Tests {
			result := db.TestResult{
				Suite:      suite.Name,
				TestName:   test.Name,
				Passed:     false,
				Pending:    true,
				Stdout:     "",
				Stderr:     "",
				Errors:     make([]string, 0),
				DurationMs: 0,
				Points:     test.Points,
			}

			if test.Public {
				publicTestMap[suite.Name][test.Name] = result
				r.publicTestSet[suite.Name][test.Name] = true
			}

			testMap[suite.Name][test.Name] = result
			r.testPoints[suite.Name][test.Name] = test.Points
		}
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":       db.StatusTesting,
		"updated":      firestore.ServerTimestamp,
		"publicTests":  publicTestMap,
		"suiteResults": suiteResults,
	})

	_ = r.updateInternalDoc(jobId, map[string]any{
		"status":  db.StatusTesting,
		"updated": firestore.ServerTimestamp,
		"tests":   testMap,
	})
}

func (r *FirestoreReporter) OnTestStart(jobId, suite, testName string) {
	// No-op rn as it would incur a lot of extra writes, but could update timestamp later
}

func (r *FirestoreReporter) OnTestEnd(jobId, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {
	var points int
	if suitePoints, ok := r.testPoints[suite]; ok {
		points = suitePoints[testName] // defaults to 0 if testName not found
	}

	result := db.TestResult{
		Suite:      suite,
		TestName:   testName,
		Passed:     passed,
		Pending:    false,
		Stdout:     truncateLog(stdout, maxTestOutputBytes),
		Stderr:     truncateLog(stderr, maxTestOutputBytes),
		Errors:     testErrors,
		DurationMs: durationMs,
		Points:     points,
	}

	_ = firebase.UpdateDocFields(r.fsClient, collectionInternal, jobId, []firestore.Update{
		{FieldPath: firestore.FieldPath{"tests", suite, testName}, Value: result},
	})

	publicUpdates := []firestore.Update{
		{Path: "completedTests", Value: firestore.Increment(1)},
		{Path: "updated", Value: firestore.ServerTimestamp},
		{FieldPath: firestore.FieldPath{"suiteResults", suite, "durationMs"}, Value: firestore.Increment(durationMs)},
	}

	if passed {
		publicUpdates = append(publicUpdates,
			firestore.Update{FieldPath: firestore.FieldPath{"suiteResults", suite, "passed"}, Value: firestore.Increment(1)},
			firestore.Update{FieldPath: firestore.FieldPath{"suiteResults", suite, "points"}, Value: firestore.Increment(points)},
		)
	} else {
		publicUpdates = append(publicUpdates,
			firestore.Update{FieldPath: firestore.FieldPath{"suiteResults", suite, "failed"}, Value: firestore.Increment(1)},
		)
	}

	if suitePublic, ok := r.publicTestSet[suite]; ok && suitePublic[testName] {
		publicUpdates = append(publicUpdates,
			firestore.Update{FieldPath: firestore.FieldPath{"publicTests", suite, testName}, Value: result},
		)
	}

	_ = firebase.UpdateDocFields(r.fsClient, collectionPublic, jobId, publicUpdates)
}

func (r *FirestoreReporter) OnTestingEnd(jobId string, err error) {
	testingDurationMs := time.Since(r.testingStart).Milliseconds()
	if err != nil {
		_ = r.updatePublicDoc(jobId, map[string]any{
			"status":            db.StatusFailed,
			"error":             err.Error(),
			"testingDurationMs": testingDurationMs,
			"completed":         firestore.ServerTimestamp,
			"updated":           firestore.ServerTimestamp,
		})

		_ = r.updateInternalDoc(jobId, map[string]any{
			"error": err.Error(),
		})

		return
	}

	_ = r.updatePublicDoc(jobId, map[string]any{
		"status":            db.StatusCompleted,
		"testingDurationMs": testingDurationMs,
		"completed":         firestore.ServerTimestamp,
		"updated":           firestore.ServerTimestamp,
	})
}

var _ playwright.GradingJobReporter = (*FirestoreReporter)(nil)
