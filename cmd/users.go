package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Manage users",
}

var usersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspace users",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := config.Load(workspaceFlag)
		if err != nil {
			return err
		}
		client := api.NewClient(ws)

		users, err := getAllUsers(client)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "USERNAME\tNAME\tEMAIL\tADMIN\tBOT")
		for _, u := range users {
			if u.Deleted {
				continue
			}
			admin := ""
			if u.IsAdmin {
				admin = "yes"
			}
			bot := ""
			if u.IsBot {
				bot = "yes"
			}
			fmt.Fprintf(w, "@%s\t%s\t%s\t%s\t%s\n", u.Name, u.RealName, u.Profile.Email, admin, bot)
		}
		w.Flush()
		return nil
	},
}

var usersInfoCmd = &cobra.Command{
	Use:   "info [@username]",
	Short: "Show user details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := config.Load(workspaceFlag)
		if err != nil {
			return err
		}
		client := api.NewClient(ws)
		name := strings.TrimPrefix(args[0], "@")

		users, err := getAllUsers(client)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		for _, u := range users {
			if u.Name == name || u.RealName == name || u.Profile.DisplayName == name {
				fmt.Printf("Username:  @%s\n", u.Name)
				fmt.Printf("Name:      %s\n", u.RealName)
				fmt.Printf("Title:     %s\n", u.Profile.Title)
				fmt.Printf("Email:     %s\n", u.Profile.Email)
				fmt.Printf("Phone:     %s\n", u.Profile.Phone)
				fmt.Printf("Status:    %s %s\n", u.Profile.StatusEmoji, u.Profile.StatusText)
				fmt.Printf("Admin:     %v\n", u.IsAdmin)
				fmt.Printf("Bot:       %v\n", u.IsBot)
				fmt.Printf("ID:        %s\n", u.ID)
				return nil
			}
		}
		return fmt.Errorf("user @%s not found", name)
	},
}

var usersSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search users by partial name, display name, or email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := config.Load(workspaceFlag)
		if err != nil {
			return err
		}
		client := api.NewClient(ws)
		query := strings.ToLower(strings.TrimPrefix(args[0], "@"))

		users, err := getAllUsers(client)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "USERNAME\tNAME\tDISPLAY\tEMAIL\tSTATUS")
		matches := 0
		for _, u := range users {
			if u.Deleted {
				continue
			}
			if !userMatches(u, query) {
				continue
			}
			matches++
			status := strings.TrimSpace(u.Profile.StatusEmoji + " " + u.Profile.StatusText)
			fmt.Fprintf(w, "@%s\t%s\t%s\t%s\t%s\n",
				u.Name, u.RealName, u.Profile.DisplayName, u.Profile.Email, status)
		}
		w.Flush()
		if matches == 0 {
			return fmt.Errorf("no users matching %q", args[0])
		}
		return nil
	},
}

// userMatches reports whether the (already lower-cased) query appears in any of
// the user's searchable fields.
func userMatches(u slack.User, query string) bool {
	fields := []string{
		u.Name,
		u.RealName,
		u.Profile.RealName,
		u.Profile.DisplayName,
		u.Profile.Email,
	}
	for _, f := range fields {
		if f != "" && strings.Contains(strings.ToLower(f), query) {
			return true
		}
	}
	return false
}

func init() {
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersInfoCmd)
	usersCmd.AddCommand(usersSearchCmd)
	rootCmd.AddCommand(usersCmd)
}
