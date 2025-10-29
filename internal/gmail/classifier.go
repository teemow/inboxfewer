package gmail

import (
	"regexp"
	"strings"

	gmail "google.golang.org/api/gmail/v1"
)

var (
	githubIssueID = regexp.MustCompile(`^<([\w-]+/[\w-]+)/issues?/(\d+).*@github\.com>$`)
	githubPullID  = regexp.MustCompile(`^<([\w-]+/[\w-]+)/pull/(\d+).*@github\.com>$`)
)

// ThreadType represents a classified thread type
type ThreadType interface {
	IsStale() (bool, error)
}

// ClassifyThread classifies a Gmail thread based on its message headers
func ClassifyThread(t *gmail.Thread, githubUser, githubToken string) ThreadType {
	for _, m := range t.Messages {
		mpart := m.Payload
		if mpart == nil {
			continue
		}
		for _, mph := range mpart.Headers {
			// <golang/go/issue/3665/100642466@github.com>
			if mph.Name == "Message-ID" &&
				(strings.Contains(mph.Value, "/issues/") || strings.Contains(mph.Value, "/issue/")) &&
				strings.Contains(mph.Value, "@github.com>") {
				m := githubIssueID.FindStringSubmatch(mph.Value)
				if m != nil {
					return &GitHubIssue{
						Repo:        m[1],
						Number:      m[2],
						GithubUser:  githubUser,
						GithubToken: githubToken,
					}
				}
			}
			if mph.Name == "Message-ID" &&
				strings.Contains(mph.Value, "/pull/") &&
				strings.Contains(mph.Value, "@github.com>") {
				m := githubPullID.FindStringSubmatch(mph.Value)
				if m != nil {
					return &GitHubPull{
						Repo:        m[1],
						Number:      m[2],
						GithubUser:  githubUser,
						GithubToken: githubToken,
					}
				}
			}
		}
	}
	return nil
}

// HeaderValue extracts a header value from a Gmail message
func HeaderValue(m *gmail.Message, header string) string {
	mpart := m.Payload
	if mpart == nil {
		return ""
	}
	for _, mph := range mpart.Headers {
		if mph.Name == header {
			return mph.Value
		}
	}
	return ""
}
