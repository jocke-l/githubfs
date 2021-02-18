package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GithubRepoEntity struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	DownloadUrl string `json:"download_url"`
	Url         string `json:"url"`
}

func (entity GithubRepoEntity) GetFile() ([]byte, error) {
	if entity.Type != "file" {
		return nil, fmt.Errorf("Entity is not a file.")
	}

	resp, err := http.Get(entity.DownloadUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func (entity GithubRepoEntity) ListDir() ([]GithubRepoEntity, error) {
	if entity.Type != "dir" {
		return nil, fmt.Errorf("Entity is not a directory")
	}

	resp, err := http.Get(entity.Url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var newEntities []GithubRepoEntity
	json.Unmarshal(body, &newEntities)

	return newEntities, nil
}

func GetRepoRoot(repo string) GithubRepoEntity {
	return GithubRepoEntity{
		Url:  fmt.Sprintf("https://api.github.com/repos/%s/contents", repo),
		Type: "dir",
	}
}
