package playwright

import (
	"errors"
	"regexp"
	"strconv"
)

type TestMeta struct {
	Name   string
	Points int
	Public bool
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
