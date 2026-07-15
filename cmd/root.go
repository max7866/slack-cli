package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/max7866/slack-cli/internal/config"
	"github.com/spf13/cobra"
)

var workspaceFlag string
var refreshFlag bool

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

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the local directory cache",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Delete cached user/channel directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := config.CacheDir()
		if err != nil {
			return err
		}
		entries, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			return err
		}
		for _, f := range entries {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
		fmt.Printf("Cleared %d cached file(s).\n", len(entries))
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&workspaceFlag, "workspace", "w", "", "Workspace to use (default: your default workspace)")
	rootCmd.PersistentFlags().BoolVar(&refreshFlag, "refresh", false, "Bypass the local cache and fetch directories live")
	cacheCmd.AddCommand(cacheClearCmd)
	rootCmd.AddCommand(cacheCmd)
}
