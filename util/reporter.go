package util

import (
	"cloud.google.com/go/firestore"
)

type GradingJobReporter interface {
	OnGradeStart(jobId string)
	OnCloneStart(jobId string, assessmentRepo string, testRepo string)
	OnCloneEnd(jobId string, assessmentRepo string, testRepo string, err error)
	OnInstallStart(jobId string)
	OnInstallEnd(jobId string, out string, err error)
	OnBuildStart(jobId string)
	OnBuildEnd(jobId string, out string, err error)
	OnServe(jobId string, err error)
	OnTestingStart(jobId string, suites []string, err error)
	OnTestStart(jobId string, suite string, testName string)
	OnTestEnd(jobId string, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error)
	OnTestingEnd(jobId string, err error)
}

type FirestoreReporter struct {
	fsClient *firestore.Client
}
