package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var workspaceFlag string

var rootCmd = &cobra.Command{
	Use:   "slack-cli",
	Short: "A CLI for Slack using browser session tokens",
	Long:  "Interact with Slack from the terminal using your browser session (xoxc token + xoxd cookie). No bot or app setup required.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&workspaceFlag, "workspace", "w", "", "Workspace to use (default: your default workspace)")
}
