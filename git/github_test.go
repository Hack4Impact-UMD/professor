package git

import "testing"

func TestGetRepoSizeKB(t *testing.T) {
	path := "Hack4Impact-UMD/professor"
	size, err := GetRepoSizeKB(path)

	if err != nil {
		t.Fail()
	}

	if size < 0 {
		t.Fail()
	}
}
