package reporter

import (
	"fmt"
	"log"

	"github.com/Hack4Impact-UMD/professor/playwright"
)

type CLIReporter struct{}

func (r *CLIReporter) OnGradeStart(jobId string) {
	log.Printf("[%s] grade start", jobId)
}
func (r *CLIReporter) OnCloneStart(jobId, assessmentRepo, testRepo string) {
	log.Printf("[%s] clone start", jobId)
}
func (r *CLIReporter) OnCloneEnd(jobId, assessmentRepo, testRepo string, err error) {
	log.Printf("[%s] clone end (err=%v)", jobId, err)
}
func (r *CLIReporter) OnInstallStart(jobId string) { log.Printf("[%s] install start", jobId) }
func (r *CLIReporter) OnInstallEnd(jobId, out string, err error) {
	log.Printf("[%s] install end (err=%v)", jobId, err)
}
func (r *CLIReporter) OnBuildStart(jobId string) { log.Printf("[%s] build start", jobId) }
func (r *CLIReporter) OnBuildEnd(jobId, out string, err error) {
	log.Printf("[%s] build end (err=%v)", jobId, err)
}
func (r *CLIReporter) OnServe(jobId string, err error) {
	log.Printf("[%s] serve (err=%v)", jobId, err)
}
func (r *CLIReporter) OnTestingStart(jobId string, repo playwright.TestRepo, err error) {
	log.Printf("[%s] testing start (err=%v)", jobId, err)
}
func (r *CLIReporter) OnTestStart(jobId, suite, testName string) {}
func (r *CLIReporter) OnTestEnd(jobId, suite, testName string, passed bool, stdout, stderr string, testErrors []string, durationMs int64, err error) {
	status := "PASS"
	if !passed {
		status = "FAIL"
	}
	fmt.Printf("[%s] %s %s/%s (%dms)\n", jobId, status, suite, testName, durationMs)
}
func (r *CLIReporter) OnTestingEnd(jobId string, err error) {
	log.Printf("[%s] testing end (err=%v)", jobId, err)
}

var _ playwright.GradingJobReporter = (*CLIReporter)(nil)
