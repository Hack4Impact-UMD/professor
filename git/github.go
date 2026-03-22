package git

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/repos/%s", c.baseURL, path))
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	repoData := githubRepoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return -1, err
	}

	return repoData.Size, nil
}

func GetRepoSizeKB(path string) (int, error) {
	return NewGitHubClient("https://api.github.com", &http.Client{}).GetRepoSizeKB(path)
}
