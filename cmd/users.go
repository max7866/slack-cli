package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client := api.NewClient(cfg)

		users, err := client.GetUsers()
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client := api.NewClient(cfg)
		name := strings.TrimPrefix(args[0], "@")

		users, err := client.GetUsers()
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		for _, u := range users {
			if u.Name == name || u.RealName == name {
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

func init() {
	usersCmd.AddCommand(usersListCmd)
	usersCmd.AddCommand(usersInfoCmd)
	rootCmd.AddCommand(usersCmd)
}
