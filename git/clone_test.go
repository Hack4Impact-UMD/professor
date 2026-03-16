package git

import (
	"os"
	"testing"
)

func TestCloneRepo(t *testing.T) {
	t.Run("should clone the repo in the dest directory", func(t *testing.T) {
		repo := "Hack4Impact-UMD/professor"
		dest := t.TempDir()

		t.Cleanup(func() {
			os.RemoveAll(dest)
		})

		if err := CloneRepo(repo, dest, ""); err != nil {
			t.Error("Error when cloning:", err)
			t.Fail()
			return
		}

		files, err := os.ReadDir(dest)

		if err != nil {
			t.Error("Error when reading dest dir:", err)
			t.Fail()
			return
		} else if len(files) <= 0 {
			t.Error("Dest dir is empty")
			t.Fail()
			return
		}
	})

}
