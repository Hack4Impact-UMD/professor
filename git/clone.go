package git

import (
	"context"
	"errors"
	"net/url"

	"github.com/Hack4Impact-UMD/professor/sandbox"
)

func githubRepoUrl(repoPath string, pat string) string {
	if pat != "" {
		return "https://" + pat + "@github.com/" + repoPath + ".git"
	} else {
		return "https://github.com/" + repoPath + ".git"
	}
}

func CloneRepo(ctx context.Context, sb sandbox.Sandbox, repoPath string, dest string, pat string) error {
	repo := githubRepoUrl(repoPath, pat)

	url, err := url.Parse(repo)

	if err != nil {
		return err
	}

	if url.Host != "github.com" {
		return errors.New("repo URL host name must be github.com")
	}

	if _, err :=
		sb.Exec(ctx, &sandbox.SandboxCommand{
			Args: []string{
				"git", "clone", "--depth", "1", repo, dest,
			},
		}); err != nil {
		return err
	}

	return nil
}
