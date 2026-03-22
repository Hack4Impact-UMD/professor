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

func GetRepoSizeKB(path string) (int, error) {
	client := http.Client{}

	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%v", path))

	if err != nil {
		return -1, err
	}

	repoData := githubRepoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&repoData); err != nil {
		return -1, err
	}

	return repoData.Size, nil
}
