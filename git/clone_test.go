package git

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestCloneRepo(t *testing.T) {
	err := godotenv.Load("../.env.test")

	if err != nil {
		log.Fatal("Could not load test env!")
		return
	}

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

	t.Run("should clone private repo with PAT in the dest directory", func(t *testing.T) {
		repo := "rk234/RamyKaddouri-h4i-assessment-Spring25"
		dest := t.TempDir()

		t.Cleanup(func() {
			os.RemoveAll(dest)
		})

		if err := CloneRepo(repo, dest, os.Getenv("GITHUB_PAT")); err != nil {
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

	t.Run("should error when cloning private repo with bad PAT", func(t *testing.T) {
		repo := "rk234/RamyKaddouri-h4i-assessment-Spring25"
		dest := t.TempDir()

		t.Cleanup(func() {
			os.RemoveAll(dest)
		})

		if err := CloneRepo(repo, dest, "abc:abc"); err != nil {
			return
		}

		t.Fail()
	})

	t.Run("should error when cloning bad repo path", func(t *testing.T) {
		repo := ""
		dest := t.TempDir()

		t.Cleanup(func() {
			os.RemoveAll(dest)
		})

		if err := CloneRepo(repo, dest, ""); err != nil {
			return
		}

		t.Fail()
	})
}
