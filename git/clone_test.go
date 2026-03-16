package git

import (
	"log"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func loadTestEnv(t *testing.T) {
	t.Helper()
	if err := godotenv.Load("../.env.test"); err != nil {
		log.Fatal("Could not load test env!")
	}
}

func TestCloneRepo(t *testing.T) {
	loadTestEnv(t)

	repo := "Hack4Impact-UMD/professor"
	dest := t.TempDir()

	if err := CloneRepo(repo, dest, ""); err != nil {
		t.Error("Error when cloning:", err)
		return
	}

	files, err := os.ReadDir(dest)
	if err != nil {
		t.Error("Error when reading dest dir:", err)
	} else if len(files) <= 0 {
		t.Error("Dest dir is empty")
	}
}

func TestCloneRepoPrivateWithPAT(t *testing.T) {
	loadTestEnv(t)

	repo := "rk234/RamyKaddouri-h4i-assessment-Spring25"
	dest := t.TempDir()

	if err := CloneRepo(repo, dest, os.Getenv("GITHUB_PAT")); err != nil {
		t.Error("Error when cloning:", err)
		return
	}

	files, err := os.ReadDir(dest)
	if err != nil {
		t.Error("Error when reading dest dir:", err)
	} else if len(files) <= 0 {
		t.Error("Dest dir is empty")
	}
}

func TestCloneRepoPrivateWithBadPAT(t *testing.T) {
	loadTestEnv(t)

	repo := "rk234/RamyKaddouri-h4i-assessment-Spring25"
	dest := t.TempDir()

	if err := CloneRepo(repo, dest, "abc:abc"); err != nil {
		return
	}

	t.Error("expected error when cloning private repo with bad PAT")
}

func TestCloneRepoBadPath(t *testing.T) {
	loadTestEnv(t)

	dest := t.TempDir()

	if err := CloneRepo("", dest, ""); err != nil {
		return
	}

	t.Error("expected error when cloning with empty repo path")
}
