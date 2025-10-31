// Package github provides types and utilities for interacting with GitHub issues and pull requests.
//
// This package contains types for representing GitHub issues and pull requests,
// along with functionality to check their staleness (whether they are closed or no longer exist).
//
// The staleness checking is used by the Gmail cleanup functionality to determine
// whether email threads related to GitHub issues/PRs should be archived.
//
// Example usage:
//
//	issue := &github.GithubIssue{
//	    Repo:        "owner/repo",
//	    Number:      "123",
//	    GithubUser:  "username",
//	    GithubToken: "token",
//	}
//
//	isStale, err := issue.IsStale()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if isStale {
//	    fmt.Println("Issue is closed or doesn't exist")
//	}
package github
