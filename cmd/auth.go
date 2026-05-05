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
	Short: "Configure Slack credentials (xoxc token + xoxd cookie)",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("Slack CLI Auth Setup")
		fmt.Println("====================")
		fmt.Println()
		fmt.Println("To get your credentials:")
		fmt.Println("  1. Open Slack in your browser and log in")
		fmt.Println("  2. Open DevTools (F12) -> Application -> Cookies -> app.slack.com")
		fmt.Println("  3. Copy the 'd' cookie value (starts with xoxd-...)")
		fmt.Println("  4. In DevTools -> Console, run: window.prompt('token', document.body.innerHTML.match(/\"api_token\":\"(xox[cs]-[^\"]+)\"/)[1])")
		fmt.Println("     Or check Network tab for any API call and find the token parameter")
		fmt.Println()

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

		cfg := &config.Config{Token: token, Cookie: cookie}

		// Validate credentials
		fmt.Println("\nValidating credentials...")
		client := api.NewClient(cfg)
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nAuthenticated as %s in %s\n", resp.User, resp.Team)
		fmt.Println("Config saved to ~/.slack-cli/config.json")
		return nil
	},
}

var authTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test current credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client := api.NewClient(cfg)
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

func init() {
	authCmd.AddCommand(authSetupCmd)
	authCmd.AddCommand(authTestCmd)
	rootCmd.AddCommand(authCmd)
}
