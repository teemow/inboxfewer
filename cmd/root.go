package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command for the inboxfewer application
var rootCmd = &cobra.Command{
	Use:   "inboxfewer",
	Short: "Archives Gmail threads for closed GitHub issues and PRs",
	Long: `inboxfewer is a tool that archives messages in your Gmail inbox if the
corresponding GitHub issue or pull request has been closed.

It can run as:
  - A standalone CLI tool (default)
  - An MCP (Model Context Protocol) server for AI assistants`,
	SilenceUsage: true,
}

// version will be set by main
var version = "dev"

// SetVersion sets the version for the root command
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

// Execute is the main entry point for the CLI application
func Execute() {
	rootCmd.SetVersionTemplate(`{{printf "inboxfewer version %s\n" .Version}}`)

	// If no subcommand is provided, run the cleanup command by default
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "cleanup")
	}

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(newCleanupCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newVersionCmd())
}
