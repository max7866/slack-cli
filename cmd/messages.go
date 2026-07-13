package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/slack-go/slack"
	"github.com/spf13/cobra"
)

var messageCount int

var messagesCmd = &cobra.Command{
	Use:   "messages",
	Short: "Read messages",
}

var messagesReadCmd = &cobra.Command{
	Use:   "read [#channel or @user]",
	Short: "Read messages from a channel or DM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := config.Load(workspaceFlag)
		if err != nil {
			return err
		}
		client := api.NewClient(ws)
		target := args[0]

		channelID, err := resolveTarget(client, target)
		if err != nil {
			return err
		}

		params := slack.GetConversationHistoryParameters{
			ChannelID: channelID,
			Limit:     messageCount,
		}
		history, err := client.GetConversationHistory(&params)
		if err != nil {
			return fmt.Errorf("failed to get messages: %w", err)
		}

		// Build a user cache for display names
		userCache := make(map[string]string)

		// Print messages in chronological order (oldest first)
		for i := len(history.Messages) - 1; i >= 0; i-- {
			msg := history.Messages[i]
			username := resolveUsername(client, msg.User, userCache)
			ts := formatTimestamp(msg.Timestamp)
			fmt.Fprintf(os.Stdout, "[%s] %s: %s\n", ts, username, msg.Text)
		}
		return nil
	},
}

func resolveTarget(client *slack.Client, target string) (string, error) {
	if strings.HasPrefix(target, "@") {
		name := strings.TrimPrefix(target, "@")
		users, err := getAllUsers(client)
		if err != nil {
			return "", fmt.Errorf("failed to list users: %w", err)
		}
		for _, u := range users {
			if u.Name == name || u.RealName == name || u.Profile.DisplayName == name {
				ch, _, _, err := client.OpenConversation(&slack.OpenConversationParameters{
					Users: []string{u.ID},
				})
				if err != nil {
					return "", fmt.Errorf("failed to open DM: %w", err)
				}
				return ch.ID, nil
			}
		}
		return "", fmt.Errorf("user @%s not found", name)
	}

	// A raw conversation ID is passed through untouched.
	if isChannelID(target) {
		return target, nil
	}

	// Anything else is treated as a channel name (with or without a leading
	// '#'), resolved against the full paginated channel list — the same logic
	// used by `send`, so a channel you can send to can also be read from.
	return resolveChannelName(client, target)
}

func resolveUsername(client *slack.Client, userID string, cache map[string]string) string {
	if userID == "" {
		return "bot"
	}
	if name, ok := cache[userID]; ok {
		return name
	}
	info, err := client.GetUserInfo(userID)
	if err != nil {
		cache[userID] = userID
		return userID
	}
	name := info.RealName
	if name == "" {
		name = info.Name
	}
	cache[userID] = name
	return name
}

func formatTimestamp(ts string) string {
	parts := strings.Split(ts, ".")
	if len(parts) == 0 {
		return ts
	}
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return ts
	}
	t := time.Unix(sec, 0)
	return t.Format("Jan 02 15:04")
}

func init() {
	messagesReadCmd.Flags().IntVarP(&messageCount, "count", "n", 20, "Number of messages to show")
	messagesCmd.AddCommand(messagesReadCmd)
	rootCmd.AddCommand(messagesCmd)
}
