package reporter

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Hack4Impact-UMD/professor/playwright"
)

type (
	msgCloneStart   struct{}
	msgCloneEnd     struct{ err error }
	msgInstallStart struct{}
	msgInstallEnd   struct {
		out      string
		err      error
		duration time.Duration
	}
	msgBuildStart struct{}
	msgBuildEnd   struct {
		out      string
		err      error
		duration time.Duration
	}
	msgServeEnd     struct{ err error }
	msgTestingStart struct {
		repo playwright.TestRepo
		err  error
	}
	msgTestEnd struct {
		suite, name    string
		passed         bool
		stdout, stderr string
		testErrors     []string
		durationMs     int64
	}
	msgTestingEnd struct{ err error }
	tickMsg       struct{}
)

type stepStatus int

const (
	stepPending stepStatus = iota
	stepRunning
	stepDone
	stepFailed
)

type testStatus int

const (
	testPending testStatus = iota
	testPassed
	testFailed
)

type step struct {
	label    string
	status   stepStatus
	output   string
	duration time.Duration
}

type testEntry struct {
	displayName string
	rawName     string
	points      int
	status      testStatus
	durationMs  int64
	errors      string
	stdout      string
	stderr      string
}

type suiteEntry struct {
	name   string
	tests  []*testEntry
	passed int
	failed int
}

type gradingModel struct {
	jobId       string
	steps       [4]step
	suites      []*suiteEntry
	testIndex   map[string]*testEntry // key: suite+"\x00"+rawName
	frame       int
	testingDone bool
}

const (
	iClone   = 0
	iInstall = 1
	iBuild   = 2
	iServe   = 3
)

func newGradingModel(jobId string) gradingModel {
	return gradingModel{
		jobId: jobId,
		steps: [4]step{
			iClone:   {label: "Clone"},
			iInstall: {label: "Install dependencies"},
			iBuild:   {label: "Build"},
			iServe:   {label: "Serve"},
		},
		testIndex: make(map[string]*testEntry),
	}
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m gradingModel) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m gradingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.frame++
		return m, tick()

	case msgCloneStart:
		m.steps[iClone].status = stepRunning

	case msgCloneEnd:
		if msg.err != nil {
			m.steps[iClone].status = stepFailed
			m.steps[iClone].output = msg.err.Error()
			return m, tea.Quit
		}
		m.steps[iClone].status = stepDone

	case msgInstallStart:
		m.steps[iInstall].status = stepRunning

	case msgInstallEnd:
		m.steps[iInstall].output = msg.out
		m.steps[iInstall].duration = msg.duration
		if msg.err != nil {
			m.steps[iInstall].status = stepFailed
			return m, tea.Quit
		}
		m.steps[iInstall].status = stepDone

	case msgBuildStart:
		m.steps[iBuild].status = stepRunning

	case msgBuildEnd:
		m.steps[iBuild].output = msg.out
		m.steps[iBuild].duration = msg.duration
		if msg.err != nil {
			m.steps[iBuild].status = stepFailed
			return m, tea.Quit
		}
		m.steps[iBuild].status = stepDone
		m.steps[iServe].status = stepRunning // serve starts immediately after build

	case msgServeEnd:
		if msg.err != nil {
			m.steps[iServe].status = stepFailed
			m.steps[iServe].output = msg.err.Error()
			return m, tea.Quit
		}
		m.steps[iServe].status = stepDone

	case msgTestingStart:
		if msg.err != nil {
			return m, tea.Quit
		}
		m.suites = make([]*suiteEntry, 0, len(msg.repo.Suites))
		for _, s := range msg.repo.Suites {
			suite := &suiteEntry{name: s.Name, tests: make([]*testEntry, 0, len(s.Tests))}
			for _, t := range s.Tests {
				raw := rawTestName(t)
				entry := &testEntry{
					displayName: fmt.Sprintf("[%dpts] %s", t.Points, t.Name),
					rawName:     raw,
					points:      t.Points,
				}
				suite.tests = append(suite.tests, entry)
				m.testIndex[s.Name+"\x00"+raw] = entry
			}
			m.suites = append(m.suites, suite)
		}

	case msgTestEnd:
		key := msg.suite + "\x00" + msg.name
		entry, ok := m.testIndex[key]
		if !ok {
			return m, nil
		}
		entry.durationMs = msg.durationMs
		if msg.passed {
			entry.status = testPassed
			for _, s := range m.suites {
				if s.name == msg.suite {
					s.passed++
					break
				}
			}
		} else {
			entry.status = testFailed
			entry.errors = strings.Join(msg.testErrors, "\n")
			entry.stdout = msg.stdout
			entry.stderr = msg.stderr
			for _, s := range m.suites {
				if s.name == msg.suite {
					s.failed++
					break
				}
			}
		}

	case msgTestingEnd:
		m.testingDone = true
		return m, tea.Quit
	}

	return m, nil
}

var (
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleErr    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleBanner = lipgloss.NewStyle().Background(lipgloss.Color("2")).Foreground(lipgloss.Color("0")).Bold(true).Padding(0, 1)
)

func (m gradingModel) View() string {
	var sb strings.Builder
	sp := spinnerFrames[m.frame%len(spinnerFrames)]

	if m.jobId != "" {
		sb.WriteString(styleDim.Render("job "+m.jobId) + "\n")
	}

	for _, s := range m.steps {
		switch s.status {
		case stepPending:
			// not yet started — don't show
		case stepRunning:
			sb.WriteString(sp + " " + s.label + "\n")
		case stepDone:
			sb.WriteString(styleOK.Render("✓") + " " + s.label)

			if s.duration.Milliseconds() > 0 {
				sb.WriteString(styleDim.Render(fmt.Sprintf(" (%dms)", s.duration.Milliseconds())) + "\n")
			} else {
				sb.WriteString("\n")
			}
			renderLines(&sb, s.output)
		case stepFailed:
			sb.WriteString(styleErr.Render("✗") + " " + s.label + "\n")
			renderLines(&sb, s.output)
		}
	}

	for _, suite := range m.suites {
		total := len(suite.tests)
		done := suite.passed + suite.failed

		var prefix string
		switch {
		case done < total:
			prefix = sp
		case suite.failed == 0:
			prefix = styleOK.Render("✓")
		default:
			prefix = styleErr.Render("✗")
		}

		label := suite.name
		if done > 0 {
			label = fmt.Sprintf("%s (%d/%d passed)", suite.name, suite.passed, total)
		}
		sb.WriteString(prefix + " " + styleBold.Render(label) + "\n")

		for _, t := range suite.tests {
			switch t.status {
			case testPending:
				sb.WriteString("  " + styleDim.Render("○ "+t.displayName) + "\n")
			case testPassed:
				sb.WriteString("  " + styleOK.Render("✓") + " " + t.displayName +
					styleDim.Render(fmt.Sprintf(" (%dms)", t.durationMs)) + "\n")
			case testFailed:
				sb.WriteString("  " + styleErr.Render("✗") + " " + t.displayName +
					styleDim.Render(fmt.Sprintf(" (%dms)", t.durationMs)) + "\n")
				sb.WriteString("  errors:\n")
				renderLines(&sb, t.errors)
				if t.stdout != "" {
					sb.WriteString("  stdout:\n")
					renderLines(&sb, t.stdout)
				}
				if t.stderr != "" {
					sb.WriteString("  stderr:\n")
					renderLines(&sb, t.stderr)
				}
			}
		}
	}

	if m.testingDone {
		var grandEarned, grandTotal int
		sb.WriteString("\n")
		sb.WriteString(styleBanner.Render("Grading Finished") + "\n")
		for _, suite := range m.suites {
			var earned, total int
			for _, t := range suite.tests {
				total += t.points
				if t.status == testPassed {
					earned += t.points
				}
			}
			grandEarned += earned
			grandTotal += total
			suiteStr := fmt.Sprintf("  %s: %d / %d pts", suite.name, earned, total)
			if earned == total {
				sb.WriteString(styleOK.Render(suiteStr) + "\n")
			} else {
				sb.WriteString(styleDim.Render(suiteStr) + "\n")
			}
		}
		totalStr := fmt.Sprintf("Total: %d / %d points", grandEarned, grandTotal)
		if grandEarned == grandTotal {
			sb.WriteString(styleOK.Render(styleBold.Render(totalStr)) + "\n")
		} else {
			sb.WriteString(styleBold.Render(totalStr) + "\n")
		}
	}

	return sb.String()
}

func renderLines(sb *strings.Builder, output string) {
	if output == "" {
		return
	}
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		sb.WriteString(styleDim.Render("  │ "+line) + "\n")
	}
}
