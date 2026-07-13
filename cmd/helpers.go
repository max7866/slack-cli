package cmd

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

// getAllConversations fetches every conversation of the given types, following
// Slack's pagination cursor until it is exhausted.
//
// Slack caps each conversations.list page (typically at ~200) regardless of the
// requested Limit and returns a next_cursor to fetch the rest. The previous code
// made a single call and dropped everything past the first page — that was the
// root cause of channels missing from `channels list` and being unresolvable by
// `messages read` even though `send` could reach them.
func getAllConversations(client *slack.Client, types []string, excludeArchived bool) ([]slack.Channel, error) {
	var all []slack.Channel
	cursor := ""
	for {
		params := &slack.GetConversationsParameters{
			Types:           types,
			Limit:           1000, // Slack still paginates below this; we follow the cursor.
			ExcludeArchived: excludeArchived,
			Cursor:          cursor,
		}
		channels, next, err := client.GetConversations(params)
		if err != nil {
			var rl *slack.RateLimitedError
			if errors.As(err, &rl) {
				time.Sleep(rl.RetryAfter)
				continue
			}
			return nil, err
		}
		all = append(all, channels...)
		if next == "" {
			break
		}
		cursor = next
	}
	return all, nil
}

// resolveChannelName resolves a channel reference (with or without a leading
// '#') to a channel ID by paginating through the full conversation list.
func resolveChannelName(client *slack.Client, name string) (string, error) {
	name = strings.TrimPrefix(name, "#")
	channels, err := getAllConversations(client, []string{"public_channel", "private_channel"}, false)
	if err != nil {
		return "", fmt.Errorf("failed to list channels: %w", err)
	}
	for _, ch := range channels {
		if ch.Name == name {
			return ch.ID, nil
		}
	}
	return "", fmt.Errorf("channel #%s not found", name)
}

// isChannelID reports whether s looks like a raw Slack conversation ID
// (e.g. C0123ABCD for public, G… for private/group, D… for DMs) rather than a
// human-typed channel name.
func isChannelID(s string) bool {
	if len(s) < 8 {
		return false
	}
	switch s[0] {
	case 'C', 'G', 'D':
	default:
		return false
	}
	for _, r := range s[1:] {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// getAllUsers fetches every user in the workspace, following the pagination
// cursor. GetUsers requests a bounded page size so a single slow page can't make
// the command appear to hang on large workspaces.
func getAllUsers(client *slack.Client) ([]slack.User, error) {
	users, err := client.GetUsers(slack.GetUsersOptionLimit(200))
	if err != nil {
		return nil, err
	}
	return users, nil
}
