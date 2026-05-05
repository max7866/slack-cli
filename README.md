# slack-cli

A command-line interface for Slack that uses your browser session tokens — no bot or Slack app setup required.

Inspired by [denniswebb/teams-cli](https://github.com/denniswebb/teams-cli).

## How It Works

Uses the same authentication as the Slack web client: an `xoxc-` API token paired with an `xoxd-` session cookie. These are extracted from your browser once, saved locally, and used for all API calls.

## Install

```bash
go install github.com/max7866/slack-cli@latest
```

Or build from source:
```bash
git clone https://github.com/max7866/slack-cli.git
cd slack-cli
go build -o slack-cli .
```

## Setup

### 1. Get Your Credentials

1. Open **Slack in your browser** and log in
2. Open **DevTools** (F12) -> **Application** -> **Cookies** -> `app.slack.com`
3. Copy the `d` cookie value (starts with `xoxd-...`)
4. For the token, check **Network** tab -> any API request -> look for `token` parameter starting with `xoxc-`

### 2. Configure

```bash
slack-cli auth setup
```

Paste your token and cookie when prompted. Credentials are saved to `~/.slack-cli/config.json` with restricted permissions (0600).

### 3. Verify

```bash
slack-cli auth test
```

## Usage

```bash
# List channels
slack-cli channels list
slack-cli channels list -a          # include archived

# Read messages
slack-cli messages read #general
slack-cli messages read #general -n 50
slack-cli messages read @username

# Send messages
slack-cli send #general "Hello from the CLI"
slack-cli send @username "Hey there"

# List users
slack-cli users list
slack-cli users info @username
```

## Security

- Credentials are stored in `~/.slack-cli/config.json` with `0600` permissions
- Tokens are scoped to your user account — same access as your browser session
- No data is sent anywhere except Slack's API servers

## Note

This tool uses Slack's internal web client API, which is not officially supported for third-party use. Tokens may expire when you log out of the browser. Re-run `slack-cli auth setup` to refresh them.
