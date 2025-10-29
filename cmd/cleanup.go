package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	gmail_v1 "google.golang.org/api/gmail/v1"

	"github.com/teemow/inboxfewer/internal/gmail"
)

var githubUser, githubToken string

func readGithubConfig() error {
	file := filepath.Join(gmail.HomeDir(), "keys", "github-inboxfewer.token")
	slurp, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	f := strings.Fields(strings.TrimSpace(string(slurp)))
	if len(f) != 2 {
		return fmt.Errorf("expected two fields (user and token) in %v; got %d fields", file, len(f))
	}
	githubUser, githubToken = f[0], f[1]
	return nil
}

func newCleanupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up Gmail inbox by archiving closed GitHub issue threads",
		Long: `Scan your Gmail inbox for threads related to GitHub issues and pull requests.
If the corresponding GitHub issue or PR is closed, the thread will be archived.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := readGithubConfig(); err != nil {
				return fmt.Errorf("failed to read GitHub config: %w", err)
			}

			ctx := context.Background()
			client, err := gmail.NewClient(ctx)
			if err != nil {
				return fmt.Errorf("failed to create Gmail client: %w", err)
			}

			n := 0
			if err := client.ForeachThread("in:inbox", func(t *gmail_v1.Thread) error {
				if err := client.PopulateThread(t); err != nil {
					return err
				}
				topic := gmail.ClassifyThread(t, githubUser, githubToken)
				n++
				log.Printf("Thread %d (%v) = %T %v", n, t.Id, topic, topic)
				if topic == nil {
					return nil
				}
				if stale, err := topic.IsStale(); err != nil {
					return err
				} else if stale {
					log.Printf("  ... archiving")
					return client.ArchiveThread(t.Id)
				}
				return nil
			}); err != nil {
				return fmt.Errorf("error processing threads: %w", err)
			}

			log.Printf("Processed %d threads", n)
			return nil
		},
	}
}
