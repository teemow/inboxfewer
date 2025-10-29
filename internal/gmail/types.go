package gmail

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GitHubIssue represents a GitHub issue referenced in Gmail
type GitHubIssue struct {
	Repo        string // "golang/go"
	Number      string // "123"
	GithubUser  string
	GithubToken string
}

// IsStale checks if the GitHub issue is closed or doesn't exist
func (id *GitHubIssue) IsStale() (bool, error) {
	issueURL := "https://api.github.com/repos/" + id.Repo + "/issues/" + id.Number
	req, _ := http.NewRequest("GET", issueURL, nil)
	req.SetBasicAuth(id.GithubUser, id.GithubToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return true, nil
	}
	if res.StatusCode != 200 {
		return false, fmt.Errorf("fetching %v, http status %s", issueURL, res.Status)
	}
	var issue struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(res.Body).Decode(&issue); err != nil {
		return false, err
	}
	return issue.State == "closed", nil
}

// GitHubPull represents a GitHub pull request referenced in Gmail
type GitHubPull struct {
	Repo        string // "golang/go"
	Number      string // "123"
	GithubUser  string
	GithubToken string
}

// IsStale checks if the GitHub pull request is closed or doesn't exist
func (id *GitHubPull) IsStale() (bool, error) {
	pullURL := "https://api.github.com/repos/" + id.Repo + "/pulls/" + id.Number
	req, _ := http.NewRequest("GET", pullURL, nil)
	req.SetBasicAuth(id.GithubUser, id.GithubToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, nil
	}
	defer res.Body.Close()
	if res.StatusCode == 404 {
		return true, nil
	}
	if res.StatusCode != 200 {
		return false, fmt.Errorf("fetching %v, http status %s", pullURL, res.Status)
	}
	var pull struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(res.Body).Decode(&pull); err != nil {
		return false, err
	}
	return pull.State == "closed", nil
}
