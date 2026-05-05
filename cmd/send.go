package cmd

import (
	"fmt"
	"strings"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var sendCmd = &cobra.Command{
	Use:   "send [#channel or @user] [message]",
	Short: "Send a message to a channel or user",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		client := api.NewClient(cfg)
		target := args[0]
		message := args[1]

		channelID, err := resolveTarget(client, target)
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
