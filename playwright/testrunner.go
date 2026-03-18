package playwright

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Hack4Impact-UMD/professor/util"
)

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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "npx", "playwright", "test", "--reporter="+reporterFile.Name())
	cmd.Dir = testDir
	cmd.Env = append(util.SandboxedEnv(), fmt.Sprintf("BASE_URL=http://localhost:%v", port))

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
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	endRecieved := false
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
			{
				endRecieved = true
				reporter.OnTestingEnd(jobId, nil)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error when reading playwright output: %v", err)
		reporter.OnTestingEnd(jobId, err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		if !endRecieved {
			return err
		}
	}

	return nil
}
