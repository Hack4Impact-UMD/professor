package playwright

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Hack4Impact-UMD/professor/util"
)

var safeEnvPrefixes = []string{
	"PATH=",
	"HOME=",
	"TMPDIR=",
	"TEMP=",
	"TMP=",
	"USER=",
	"LANG=",
	"LC_",
	"NODE_",
	"npm_",
	"PLAYWRIGHT_",
}

func sandboxedEnv() []string {
	var filtered []string
	for _, kv := range os.Environ() {
		for _, prefix := range safeEnvPrefixes {
			if strings.HasPrefix(kv, prefix) {
				filtered = append(filtered, kv)
				break
			}
		}
	}
	return filtered
}

//go:embed reporter/reporter.ts
var reporterTS []byte

// ndjsonEvent mirrors the union type emitted by reporter.ts.
type ndjsonEvent struct {
	Type       string   `json:"type"`
	Suites     []string `json:"suites"`
	Suite      string   `json:"suite"`
	Test       string   `json:"test"`
	Passed     bool     `json:"passed"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	Errors     []string `json:"errors"`
	DurationMs int64    `json:"durationMs"`
}

func RunPlaywrightTests(jobId string, testDir string, port int, reporter util.GradingJobReporter) error {
	reporterFile, err := os.CreateTemp("", "pw-reporter-*.ts")
	if err != nil {
		reporter.OnTestingStart(jobId, nil, err)
		return err
	}
	defer os.Remove(reporterFile.Name())

	if _, err := reporterFile.Write(reporterTS); err != nil {
		reporter.OnTestingStart(jobId, nil, err)
		return err
	}
	reporterFile.Close()

	cmd := exec.Command("npx", "playwright", "test", "--reporter="+reporterFile.Name())
	cmd.Dir = testDir
	cmd.Env = append(sandboxedEnv(), fmt.Sprintf("BASE_URL=http://localhost:%v", port))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		reporter.OnTestingStart(jobId, nil, err)
		return err
	}

	if err := cmd.Start(); err != nil {
		reporter.OnTestingStart(jobId, nil, err)
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		var event ndjsonEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}

		switch event.Type {
		case "begin":
			reporter.OnTestingStart(jobId, event.Suites, nil)
		case "testBegin":
			reporter.OnTestStart(jobId, event.Suite, event.Test)
		case "testEnd":
			reporter.OnTestEnd(
				jobId,
				event.Suite,
				event.Test,
				event.Passed,
				event.Stdout,
				event.Stderr,
				event.Errors,
				event.DurationMs,
				nil,
			)
		case "end":
			reporter.OnTestingEnd(jobId)
		}
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return err
		}
	}

	return nil
}
