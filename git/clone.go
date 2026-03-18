package git

import (
	"errors"
	"log"
	"net/url"
	"os/exec"
)

func githubRepoUrl(repoPath string, pat string) string {
	if pat != "" {
		return "https://" + pat + "@github.com/" + repoPath + ".git"
	} else {
		return "https://github.com/" + repoPath + ".git"
	}
}

func CloneRepo(repoPath string, dest string, pat string) error {
	repo := githubRepoUrl(repoPath, pat)

	url, err := url.Parse(repo)

	if err != nil {
		return err
	}

	if url.Host != "github.com" {
		return errors.New("repo URL host name must be github.com")
	}

	log.Printf("cloning %s to %s", repo, dest)

	cmd := exec.Command("git", "clone", "--depth", "1", repo, dest)

	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}

	return nil
}
