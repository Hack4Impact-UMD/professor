package db

import "time"

const (
	StatusQueued     = "queued"
	StatusPending    = "pending"
	StatusCloning    = "cloning"
	StatusInstalling = "installing"
	StatusBuilding   = "building"
	StatusServing    = "serving"
	StatusTesting    = "testing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

type TestResult struct {
	Suite      string   `firestore:"suite"`
	TestName   string   `firestore:"testName"`
	Passed     bool     `firestore:"passed"`
	Stdout     string   `firestore:"stdout"`
	Stderr     string   `firestore:"stderr"`
	Errors     []string `firestore:"errors"`
	DurationMs int64    `firestore:"durationMs"`
	Points     int      `firestore:"points"`
}

// only suite-level results are displayed to applicants
type SuiteResult struct {
	SuiteName   string `firestore:"suiteName"`
	Passed      int    `firestore:"passed"`
	Failed      int    `firestore:"failed"`
	Total       int    `firestore:"total"`
	DurationMs  int64  `firestore:"durationMs"`
	Points      int    `firestore:"points"`
	TotalPoints int    `firestore:"totalPoints"`
}

type GradingJobPublic struct {
	Id             string                  `firestore:"id"`
	ResponseId     string                  `firestore:"responseId"`
	RepoURL        string                  `firestore:"repoURL"`
	Status         string                  `firestore:"status"`
	Score          float64                 `firestore:"score"`
	TotalTests     int                     `firestore:"totalTests"`
	CompletedTests int                     `firestore:"completedTests"`
	Error          string                  `firestore:"error,omitempty"`
	Started        time.Time               `firestore:"started"`
	Completed      time.Time               `firestore:"completed,omitempty"`
	Updated        time.Time               `firestore:"updated"`
	SuiteResults   map[string]SuiteResult  `firestore:"suiteResults"` // map of suite name -> public results
	PublicTests    map[string][]TestResult `firestore:"publicTests"`  // results for public tests
}

// other fields can be fetched from GradingJobPublic
type GradingJobDataInternal struct {
	Id            string                  `firestore:"id"` // associated with a grading job id
	TestRepo      string                  `firestore:"testRepo"`
	BuildLog      string                  `firestore:"buildLog"`
	InstallLog    string                  `firestore:"installLog"`
	PlaywrightLog string                  `firestore:"playwrightLog"`
	Error         string                  `firestore:"error,omitempty"`
	Tests         map[string][]TestResult `firestore:"tests"` // map of suite name -> tests
}
