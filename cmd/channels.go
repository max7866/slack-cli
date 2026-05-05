package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var includeArchived bool

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Manage channels",
}

var channelsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List channels",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client := api.NewClient(cfg)

		params := &slack.GetConversationsParameters{
			Types:           []string{"public_channel", "private_channel"},
			Limit:           200,
			ExcludeArchived: !includeArchived,
		}

		channels, _, err := client.GetConversations(params)
		if err != nil {
			return fmt.Errorf("failed to list channels: %w", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tMEMBERS\tTOPIC\tPRIVATE")
		for _, ch := range channels {
			private := ""
			if ch.IsPrivate {
				private = "yes"
			}
			topic := ch.Topic.Value
			if len(topic) > 50 {
				topic = topic[:50] + "..."
			}
			fmt.Fprintf(w, "#%s\t%d\t%s\t%s\n", ch.Name, ch.NumMembers, topic, private)
		}
		w.Flush()
		return nil
	},
}

func init() {
	channelsListCmd.Flags().BoolVarP(&includeArchived, "archived", "a", false, "Include archived channels")
	channelsCmd.AddCommand(channelsListCmd)
	rootCmd.AddCommand(channelsCmd)
}
