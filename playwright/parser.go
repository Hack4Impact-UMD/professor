package playwright

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

type TestMeta struct {
	Name   string
	Points int
	Public bool
}

type TestSuite struct {
	Name  string
	Tests []TestMeta
}

type TestRepo struct {
	Suites []TestSuite
}

// Ex: [5] - A 5 pt private test
// Ex: [10*] - A 10 pt public test
func ParseTestName(name string) (TestMeta, error) {
	re := regexp.MustCompile(`\[(\d+)(\*)?\] \- (.*)`)
	matches := re.FindStringSubmatch(name)

	if len(matches) != 4 {
		return TestMeta{}, errors.New("Did not match test name regex")
	}

	pts, err := strconv.Atoi(matches[1])

	public := false

	if matches[2] == "*" {
		public = true
	}

	testName := matches[3]

	if err != nil {
		return TestMeta{}, err
	}

	return TestMeta{
		Name:   testName,
		Points: pts,
		Public: public,
	}, nil
}

func ParseTestRepo(data []byte) (TestRepo, error) {
	var raw struct {
		Suites []struct {
			Name  string   `json:"name"`
			Tests []string `json:"tests"`
		} `json:"suites"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return TestRepo{}, fmt.Errorf("parse test repo: %w", err)
	}

	repo := TestRepo{Suites: make([]TestSuite, 0, len(raw.Suites))}
	for _, s := range raw.Suites {
		suite := TestSuite{Name: s.Name, Tests: make([]TestMeta, 0, len(s.Tests))}
		for _, testName := range s.Tests {
			meta, err := ParseTestName(testName)
			if err != nil {
				return TestRepo{}, fmt.Errorf("suite %q: %w", s.Name, err)
			}
			suite.Tests = append(suite.Tests, meta)
		}
		repo.Suites = append(repo.Suites, suite)
	}
	return repo, nil
}
