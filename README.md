# slack-cli

A command-line interface for Slack that uses your browser session tokens — no bot or Slack app setup required. Built with Go.

## How It Works

Uses the same authentication as the Slack web client: an `xoxc-` API token paired with an `xoxd-` session cookie. You paste the cookie once from your browser, and the CLI auto-extracts the API token.

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

### Quick Setup (recommended)

```bash
slack-cli auth login
```

This will:
1. Open Slack in your browser
2. Prompt you to paste the `d` cookie from DevTools (`Cmd+Option+I` -> Application -> Cookies -> `app.slack.com` -> `d`)
3. Prompt for your browser's User-Agent (paste from Console: `navigator.userAgent`)
4. Auto-extract the API token from your session
5. Save credentials to `~/.slack-cli/config.json`

> **SSO/Okta users:** The User-Agent step is important. Slack SSO workspaces detect when a session is used from a different client and will invalidate it. Providing your browser's exact User-Agent ensures the CLI looks like the same session.

### Manual Setup

If you already have both tokens:
```bash
slack-cli auth setup
```

### Verify

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
slack-cli users search ryan       # partial match on name, display name, or email
```

## Multi-Workspace Support

Add multiple workspaces by running `auth login` for each:

```bash
slack-cli auth login    # adds first workspace
slack-cli auth login    # adds second workspace
```

Manage workspaces:
```bash
slack-cli auth list                 # show all workspaces (* = default)
slack-cli auth switch mycompany     # change default workspace
```

Use `-w` to target a specific workspace:
```bash
slack-cli -w mycompany channels list
slack-cli -w mycompany send #general "Hello"
```

## Features

- **No bot or Slack app required** — uses your browser session
- **Auto token extraction** — paste one cookie, token is fetched automatically
- **Multi-workspace** — switch between multiple Slack workspaces
- **Read & send messages** — channels and DMs
- **List channels & users** — browse your workspace
- **Pure Go** — single binary, no external dependencies

## Security

- Credentials stored in `~/.slack-cli/config.json` with `0600` permissions
- Tokens are scoped to your user account — same access as your browser session
- No data is sent anywhere except Slack's API servers

## Note

This tool uses Slack's internal web client API, which is not officially supported for third-party use. Tokens may expire when you log out of the browser. Re-run `slack-cli auth login` to refresh them.
