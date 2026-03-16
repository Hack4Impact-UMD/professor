package git

import (
	"log"
	"os/exec"
)

func githubRepoUrl(repoPath string, pat string) string {
	if pat != "" {
		return "https://" + pat + "@://github.com/" + repoPath + ".git"
	} else {
		return "https://github.com/" + repoPath + ".git"
	}
}

func CloneRepo(repoPath string, dest string, pat string) error {
	repo := githubRepoUrl(repoPath, pat)

	log.Printf("cloning %s to %s", repo, dest)

	cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)

	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}

	return nil
}
