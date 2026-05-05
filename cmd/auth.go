package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure Slack credentials manually (xoxc token + xoxd cookie)",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Workspace name (e.g., mycompany): ")
		name, _ := reader.ReadString('\n')
		name = strings.TrimSpace(name)

		fmt.Print("Enter xoxc token: ")
		token, _ := reader.ReadString('\n')
		token = strings.TrimSpace(token)

		fmt.Print("Enter xoxd cookie: ")
		cookie, _ := reader.ReadString('\n')
		cookie = strings.TrimSpace(cookie)

		if !strings.HasPrefix(token, "xoxc-") {
			return fmt.Errorf("token should start with 'xoxc-'")
		}
		if !strings.HasPrefix(cookie, "xoxd-") {
			return fmt.Errorf("cookie should start with 'xoxd-'")
		}

		ws := &config.Workspace{Token: token, Cookie: cookie}

		fmt.Println("\nValidating credentials...")
		client := api.NewClient(ws)
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if err := config.SaveWorkspace(name, ws); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nAuthenticated as %s in %s\n", resp.User, resp.Team)
		fmt.Printf("Workspace '%s' saved.\n", name)
		return nil
	},
}

var authTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test current credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := config.Load(workspaceFlag)
		if err != nil {
			return err
		}
		client := api.NewClient(ws)
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}
		fmt.Printf("User:      %s\n", resp.User)
		fmt.Printf("Team:      %s\n", resp.Team)
		fmt.Printf("User ID:   %s\n", resp.UserID)
		fmt.Printf("Team ID:   %s\n", resp.TeamID)
		fmt.Printf("URL:       %s\n", resp.URL)
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured workspaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		names, defaultName, err := config.ListWorkspaces()
		if err != nil {
			return err
		}
		if len(names) == 0 {
			fmt.Println("No workspaces configured. Run 'slack-cli auth login' to add one.")
			return nil
		}
		fmt.Println("Configured workspaces:")
		for _, name := range names {
			marker := "  "
			if name == defaultName {
				marker = "* "
			}
			fmt.Printf("  %s%s\n", marker, name)
		}
		fmt.Printf("\n* = default (use 'slack-cli auth switch <name>' to change)\n")
		return nil
	},
}

var authSwitchCmd = &cobra.Command{
	Use:   "switch <workspace>",
	Short: "Set the default workspace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := config.SetDefault(name); err != nil {
			return err
		}
		fmt.Printf("Default workspace set to '%s'\n", name)
		return nil
	},
}

func init() {
	authCmd.AddCommand(authSetupCmd)
	authCmd.AddCommand(authTestCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authSwitchCmd)
	rootCmd.AddCommand(authCmd)
}
