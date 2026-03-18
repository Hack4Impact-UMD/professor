package playwright

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/Hack4Impact-UMD/professor/serve"
	"github.com/Hack4Impact-UMD/professor/util"
)

const testIndexHTML = `<!DOCTYPE html>
<html>
  <head><title>Test Page</title></head>
  <body><h1>Hello</h1></body>
</html>`

type mockReporter struct {
	mu    sync.Mutex
	calls []string

	testingStartSuites []string
	testEnds           []mockTestEnd
}

type mockTestEnd struct {
	suite      string
	testName   string
	passed     bool
	durationMs int64
}

func (r *mockReporter) record(call string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *mockReporter) OnGradeStart(jobId string)                 { r.record("OnGradeStart") }
func (r *mockReporter) OnCloneStart(jobId, a, b string)           { r.record("OnCloneStart") }
func (r *mockReporter) OnCloneEnd(jobId, a, b string, err error)  { r.record("OnCloneEnd") }
func (r *mockReporter) OnInstallStart(jobId string)               { r.record("OnInstallStart") }
func (r *mockReporter) OnInstallEnd(jobId, out string, err error) { r.record("OnInstallEnd") }
func (r *mockReporter) OnBuildStart(jobId string)                 { r.record("OnBuildStart") }
func (r *mockReporter) OnBuildEnd(jobId, out string, err error)   { r.record("OnBuildEnd") }
func (r *mockReporter) OnServe(jobId string, err error)           { r.record("OnServe") }
func (r *mockReporter) OnTestingEnd(jobId string, err error)      { r.record("OnTestingEnd") }
func (r *mockReporter) OnTestStart(jobId, suite, testName string) {
	r.record("OnTestStart:" + suite + "/" + testName)
}

func (r *mockReporter) OnTestingStart(jobId string, suites []string, err error) {
	r.record("OnTestingStart")
	r.mu.Lock()
	r.testingStartSuites = suites
	r.mu.Unlock()
}

func (r *mockReporter) OnTestEnd(jobId, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {
	r.record("OnTestEnd:" + suite + "/" + testName)
	r.mu.Lock()
	r.testEnds = append(r.testEnds, mockTestEnd{suite, testName, passed, durationMs})
	r.mu.Unlock()
}

func (r *mockReporter) hasCall(call string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.calls {
		if c == call {
			return true
		}
	}
	return false
}

func TestRunPlaywrightTests(t *testing.T) {
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}

	// Serve a minimal page that the example spec tests against
	distDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte(testIndexHTML), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}
	port, stop, err := serve.ServeAssessment(distDir)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer stop()

	if err := util.WaitForPort(port, 5*time.Second); err != nil {
		t.Fatalf("file server did not respond: %v", err)
	}

	testDir, err := filepath.Abs("reporter")
	if err != nil {
		t.Fatalf("could not resolve reporter dir: %v", err)
	}

	rep := &mockReporter{}
	if err := RunPlaywrightTests("test-job", testDir, port, rep); err != nil {
		t.Fatalf("RunPlaywrightTests() error = %v", err)
	}

	// OnTestingStart must fire and include the suite from example.spec.ts
	if !rep.hasCall("OnTestingStart") {
		t.Fatal("OnTestingStart was never called")
	}
	foundSuite := false
	for _, s := range rep.testingStartSuites {
		if s == "my suite" {
			foundSuite = true
			break
		}
	}
	if !foundSuite {
		t.Errorf("OnTestingStart: suites %v do not contain \"my suite\"", rep.testingStartSuites)
	}

	// Both tests must produce OnTestStart and OnTestEnd events
	for _, name := range []string{"has title", "has heading"} {
		if !rep.hasCall("OnTestStart:my suite/" + name) {
			t.Errorf("OnTestStart never fired for test %q", name)
		}
		if !rep.hasCall("OnTestEnd:my suite/" + name) {
			t.Errorf("OnTestEnd never fired for test %q", name)
		}
	}

	// All tests must pass against the served page and have non-negative durations
	rep.mu.Lock()
	ends := rep.testEnds
	rep.mu.Unlock()
	for _, e := range ends {
		if !e.passed {
			t.Errorf("test %q/%q failed unexpectedly", e.suite, e.testName)
		}
		if e.durationMs < 0 {
			t.Errorf("test %q/%q: negative durationMs %d", e.suite, e.testName, e.durationMs)
		}
	}

	// OnTestingEnd must fire after OnTestingStart
	if !rep.hasCall("OnTestingEnd") {
		t.Fatal("OnTestingEnd was never called")
	}
	rep.mu.Lock()
	calls := rep.calls
	rep.mu.Unlock()
	startIdx, endIdx := -1, -1
	for i, c := range calls {
		if c == "OnTestingStart" {
			startIdx = i
		}
		if c == "OnTestingEnd" {
			endIdx = i
		}
	}
	if startIdx == -1 || endIdx == -1 || startIdx >= endIdx {
		t.Errorf("expected OnTestingStart before OnTestingEnd, got: %v", calls)
	}
}
