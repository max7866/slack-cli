package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/max7866/slack-cli/internal/api"
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
		wsName, ws, err := loadWorkspace()
		if err != nil {
			return err
		}
		client := api.NewClient(ws)

		var channels []slack.Channel
		if includeArchived {
			// Archived channels are a different, rarely-needed set — always
			// fetch live rather than caching them alongside active channels.
			channels, err = fetchArchivedInclusive(client)
		} else {
			channels, err = getAllConversations(client, wsName, []string{"public_channel", "private_channel"})
		}
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
