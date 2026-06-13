package reporter

import (
	"fmt"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Hack4Impact-UMD/professor/playwright"
)

type CLIReporter struct {
	program *tea.Program
	once    sync.Once
	wg      sync.WaitGroup
}

func (r *CLIReporter) start(jobId string) {
	r.once.Do(func() {
		r.program = tea.NewProgram(newGradingModel(jobId))
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.program.Run() //nolint:errcheck
		}()
	})
}

func (r *CLIReporter) send(msg tea.Msg) {
	if r.program != nil {
		r.program.Send(msg)
	}
}

// Wait blocks until the TUI has finished rendering. Call this after the
// grading job returns to ensure the final frame is visible before exit.
func (r *CLIReporter) Wait() {
	r.wg.Wait()
}

// rawTestName reconstructs the original test name string from a parsed TestMeta.
func rawTestName(meta playwright.TestMeta) string {
	if meta.Public {
		return fmt.Sprintf("[%d*] - %s", meta.Points, meta.Name)
	}
	return fmt.Sprintf("[%d] - %s", meta.Points, meta.Name)
}

func (r *CLIReporter) OnGradeStart(jobId string) {
	r.start(jobId)
}
func (r *CLIReporter) OnCloneStart(jobId, _, _ string) {
	r.start(jobId)
	r.send(msgCloneStart{})
}
func (r *CLIReporter) OnCloneEnd(_, _, _ string, err error) {
	r.send(msgCloneEnd{err: err})
	if err != nil {
		r.wg.Wait()
	}
}
func (r *CLIReporter) OnInstallStart(jobId string) {
	r.start(jobId) // no-op if OnGradeStart already ran; safety net for local mode
	r.send(msgInstallStart{})
}
func (r *CLIReporter) OnInstallEnd(_, out string, err error) {
	r.send(msgInstallEnd{out: out, err: err})
	if err != nil {
		r.wg.Wait()
	}
}
func (r *CLIReporter) OnBuildStart(_ string) {
	r.send(msgBuildStart{})
}
func (r *CLIReporter) OnBuildEnd(_, out string, err error) {
	r.send(msgBuildEnd{out: out, err: err})
	if err != nil {
		r.wg.Wait()
	}
}
func (r *CLIReporter) OnServe(_ string, err error) {
	r.send(msgServeEnd{err: err})
	if err != nil {
		r.wg.Wait()
	}
}
func (r *CLIReporter) OnTestingStart(_ string, repo playwright.TestRepo, err error) {
	r.send(msgTestingStart{repo: repo, err: err})
	if err != nil {
		r.wg.Wait()
	}
}
func (r *CLIReporter) OnTestStart(_, _, _ string) {}
func (r *CLIReporter) OnTestEnd(_ string, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, _ error) {
	r.send(msgTestEnd{
		suite:      suite,
		name:       testName,
		passed:     passed,
		stdout:     stdout,
		stderr:     stderr,
		testErrors: testErrors,
		durationMs: durationMs,
	})
}
func (r *CLIReporter) OnTestingEnd(_ string, err error) {
	r.send(msgTestingEnd{err: err})
	r.wg.Wait()
}

var _ playwright.GradingJobReporter = (*CLIReporter)(nil)
