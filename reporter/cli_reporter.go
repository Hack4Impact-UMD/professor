package reporter

import (
	"fmt"
	"strings"

	"github.com/Hack4Impact-UMD/professor/playwright"
	"github.com/pterm/pterm"
)

type CLIReporter struct {
	multi          *pterm.MultiPrinter
	testSpinners   map[string]*pterm.SpinnerPrinter // key: suite+"\x00"+rawTestName
	suiteSpinners  map[string]*pterm.SpinnerPrinter // key: suite name
	suitePassed    map[string]int
	suiteFailed    map[string]int
	suiteTotal     map[string]int
	pendingFailures []pendingFailure
}

type pendingFailure struct {
	label  string
	errors string
	stdout string
	stderr string
}

func printSubprocessOutput(output string) {
	if output == "" {
		return
	}
	style := pterm.NewStyle(pterm.FgDarkGray)
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		pterm.Println(style.Sprint("  │ " + line))
	}
}

func rawTestName(meta playwright.TestMeta) string {
	if meta.Public {
		return fmt.Sprintf("[%d*] - %s", meta.Points, meta.Name)
	}
	return fmt.Sprintf("[%d] - %s", meta.Points, meta.Name)
}

func (r *CLIReporter) OnGradeStart(jobId string) {
	pterm.Info.Printf("grading started [job ID = %s]", jobId)
}
func (r *CLIReporter) OnCloneStart(jobId, assessmentRepo, testRepo string) {
	pterm.Info.Print("cloning...")
}
func (r *CLIReporter) OnCloneEnd(jobId, assessmentRepo, testRepo string, err error) {
	if err != nil {
		pterm.FgRed.Println("error!")
		printSubprocessOutput(err.Error())
	} else {
		pterm.FgCyan.Println("done!")
	}
}
func (r *CLIReporter) OnInstallStart(jobId string) {
	pterm.Info.Print("installing dependencies...")
}
func (r *CLIReporter) OnInstallEnd(jobId, out string, err error) {
	if err != nil {
		pterm.FgRed.Println("error!")
		printSubprocessOutput(out)
	} else {
		pterm.FgCyan.Println("done!")
	}
}
func (r *CLIReporter) OnBuildStart(jobId string) {
	pterm.Info.Print("building...")
}
func (r *CLIReporter) OnBuildEnd(jobId, out string, err error) {
	if err != nil {
		pterm.FgRed.Println("error!")
		printSubprocessOutput(out)
	} else {
		pterm.FgCyan.Println("done!")
	}
}
func (r *CLIReporter) OnServe(jobId string, err error) {
	if err != nil {
		pterm.Error.Printf("failed to serve assessment: %v", err)
	}
}

func (r *CLIReporter) OnTestingStart(jobId string, repo playwright.TestRepo, err error) {
	if err != nil {
		pterm.Error.Printf("failed to start tests: %v", err)
		return
	}

	multi := pterm.DefaultMultiPrinter
	r.multi = &multi
	r.testSpinners = make(map[string]*pterm.SpinnerPrinter)
	r.suiteSpinners = make(map[string]*pterm.SpinnerPrinter)
	r.suitePassed = make(map[string]int)
	r.suiteFailed = make(map[string]int)
	r.suiteTotal = make(map[string]int)
	r.pendingFailures = nil

	for _, suite := range repo.Suites {
		suiteSpinner, _ := pterm.DefaultSpinner.WithWriter(r.multi.NewWriter()).Start(suite.Name)
		r.suiteSpinners[suite.Name] = suiteSpinner
		r.suiteTotal[suite.Name] = len(suite.Tests)

		for _, test := range suite.Tests {
			label := fmt.Sprintf("  [%dpts] %s", test.Points, test.Name)
			spinner, _ := pterm.DefaultSpinner.WithWriter(r.multi.NewWriter()).Start(label)
			r.testSpinners[suite.Name+"\x00"+rawTestName(test)] = spinner
		}
	}

	r.multi.Start()
}

func (r *CLIReporter) OnTestStart(jobId, suite, testName string) {}

func (r *CLIReporter) OnTestEnd(jobId, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {
	key := suite + "\x00" + testName
	spinner, ok := r.testSpinners[key]
	if !ok {
		status := "PASS"
		if !passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s %s/%s (%dms)\n", jobId, status, suite, testName, durationMs)
		return
	}

	meta, parseErr := playwright.ParseTestName(testName)
	displayName := testName
	if parseErr == nil {
		displayName = fmt.Sprintf("[%dpts] %s", meta.Points, meta.Name)
	}
	label := fmt.Sprintf("  %s (%dms)", displayName, durationMs)

	if passed {
		spinner.Success(label)
		r.suitePassed[suite]++
	} else {
		spinner.Fail(label)
		r.suiteFailed[suite]++
		r.pendingFailures = append(r.pendingFailures, pendingFailure{
			label:  fmt.Sprintf("%s / %s", suite, displayName),
			errors: strings.Join(testErrors, "\n"),
			stdout: stdout,
			stderr: stderr,
		})
	}

	done := r.suitePassed[suite] + r.suiteFailed[suite]
	if done == r.suiteTotal[suite] {
		if suiteSpinner, ok := r.suiteSpinners[suite]; ok {
			allPassed := r.suiteFailed[suite] == 0
			suiteLabel := fmt.Sprintf("%s (%d/%d passed)", suite, r.suitePassed[suite], r.suiteTotal[suite])
			if allPassed {
				suiteSpinner.Success(suiteLabel)
			} else {
				suiteSpinner.Fail(suiteLabel)
			}
			delete(r.suiteSpinners, suite)
		}
	}
}

func (r *CLIReporter) OnTestingEnd(jobId string, err error) {
	for name, spinner := range r.suiteSpinners {
		spinner.Info(name)
	}
	if r.multi != nil {
		r.multi.Stop()
	}
	for _, f := range r.pendingFailures {
		pterm.Error.Println(f.label)
		printSubprocessOutput(f.errors)
		printSubprocessOutput(f.stdout)
		printSubprocessOutput(f.stderr)
	}
	if err != nil {
		pterm.Error.Printf("testing failed: %v", err)
	}
}

var _ playwright.GradingJobReporter = (*CLIReporter)(nil)
