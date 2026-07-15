package cmd

import (
	"fmt"
	"strings"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send [#channel | @user | @a,@b,@c] [message]",
	Short: "Send a message to a channel, user, or group DM",
	Long: "Send a message to a channel, a single user, or a group DM.\n\n" +
		"Recipients may be a #channel, a single @user or email, or a\n" +
		"comma-separated list of people (up to 8) to open a group DM:\n\n" +
		"  slack-cli send #general \"hi team\"\n" +
		"  slack-cli send @ana \"hey\"\n" +
		"  slack-cli send @ana,@ben,carol@co.com \"lunch?\"",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		wsName, ws, err := loadWorkspace()
		if err != nil {
			return err
		}
		client := api.NewClient(ws)
		target := args[0]
		message := args[1]

		channelID, err := resolveTarget(client, wsName, target)
		if err != nil {
			return err
		}

		_, _, err = client.PostMessage(channelID, slack.MsgOptionText(message, false))
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		label := target
		if strings.HasPrefix(target, "#") || strings.HasPrefix(target, "@") {
			label = target
		}
		fmt.Printf("Message sent to %s\n", label)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)
}
