package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type githubRepoResponse struct {
	Id       int    `json:"id"`
	FullName string `json:"full_name"`
	Size     int    `json:"size"`
}

type GitHubClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewGitHubClient(baseURL string, httpClient *http.Client) *GitHubClient {
	return &GitHubClient{baseURL: baseURL, httpClient: httpClient}
}

func (c *GitHubClient) GetRepoSizeKB(path string) (int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/repos/%s", c.baseURL, path), nil)

	if err != nil {
		return -1, err
	}

	if pat := os.Getenv("GITHUB_PAT"); pat != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", pat))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	// Fail closed: a non-200 (e.g. 404 for a missing/private repo, or 403 when
	// rate-limited) returns a body without a "size" field, which would decode to
	// 0 and silently pass the size gate. Reject anything that isn't a clean 200.
	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("github api returned status %d for repo %q", resp.StatusCode, path)
	}

	repoData := githubRepoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return -1, err
	}

	return repoData.Size, nil
}

func GetRepoSizeKB(path string) (int, error) {
	return NewGitHubClient("https://api.github.com", &http.Client{}).GetRepoSizeKB(path)
}
