package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/spf13/cobra"
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Slack via browser (guided token extraction)",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter your Slack workspace URL (e.g., mycompany.slack.com): ")
		workspace, _ := reader.ReadString('\n')
		workspace = strings.TrimSpace(workspace)
		workspace = strings.TrimPrefix(workspace, "https://")
		workspace = strings.TrimPrefix(workspace, "http://")
		workspace = strings.TrimSuffix(workspace, "/")

		profileName := strings.Split(workspace, ".")[0]

		fmt.Println()
		fmt.Println("Opening Slack in your browser...")
		openBrowser(fmt.Sprintf("https://%s/", workspace))

		fmt.Println()
		fmt.Println("Once you're signed in, we need two things from DevTools (Cmd+Option+I):")
		fmt.Println()
		fmt.Println("1. The 'd' cookie:")
		fmt.Println("   Application -> Cookies -> app.slack.com -> copy the 'd' value (starts with xoxd-)")
		fmt.Println()

		fmt.Print("Paste the d cookie: ")
		cookie, _ := reader.ReadString('\n')
		cookie = strings.TrimSpace(cookie)

		if !strings.HasPrefix(cookie, "xoxd-") {
			return fmt.Errorf("cookie should start with 'xoxd-'")
		}

		fmt.Println()
		fmt.Println("2. Your browser's User-Agent (needed for SSO/Okta workspaces):")
		fmt.Println("   In the Console tab, type:  navigator.userAgent")
		fmt.Println("   Copy the output string.")
		fmt.Println()
		fmt.Println("   Or press Enter to skip (uses a default — may cause issues with SSO).")
		fmt.Println()

		fmt.Print("Paste User-Agent (or Enter to skip): ")
		userAgent, _ := reader.ReadString('\n')
		userAgent = strings.TrimSpace(userAgent)
		// Strip quotes if they pasted the JS string with quotes
		userAgent = strings.Trim(userAgent, "\"'")

		if userAgent == "" {
			userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"
		}

		fmt.Println("\nExtracting API token...")
		token, err := extractToken(workspace, cookie, userAgent)
		if err != nil {
			return err
		}

		ws := &config.Workspace{Token: token, Cookie: cookie, UserAgent: userAgent}

		fmt.Println("Validating credentials...")
		client := api.NewClient(ws)
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if err := config.SaveWorkspace(profileName, ws); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nAuthenticated as %s in %s\n", resp.User, resp.Team)
		fmt.Printf("Workspace '%s' saved to ~/.slack-cli/config.json\n", profileName)
		return nil
	},
}

func extractToken(workspace string, cookie string, userAgent string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/", workspace), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", fmt.Sprintf("d=%s", cookie))
	req.Header.Set("User-Agent", userAgent)

	// Follow redirects but preserve the cookie
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Set("Cookie", fmt.Sprintf("d=%s", cookie))
			req.Header.Set("User-Agent", userAgent)
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch workspace page: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Response: %d from %s\n", resp.StatusCode, resp.Request.URL)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Try multiple patterns to find the token
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`"api_token"\s*:\s*"(xoxc-[^"]+)"`),
		regexp.MustCompile(`"token"\s*:\s*"(xoxc-[^"]+)"`),
		regexp.MustCompile(`(xoxc-[a-zA-Z0-9-]+)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindSubmatch(body)
		if len(matches) >= 2 {
			fmt.Println("Token found!")
			return string(matches[1]), nil
		}
	}

	// Debug: show what page we got
	title := regexp.MustCompile(`<title>([^<]+)</title>`)
	titleMatch := title.FindSubmatch(body)
	pageTitle := "unknown"
	if len(titleMatch) >= 2 {
		pageTitle = string(titleMatch[1])
	}
	return "", fmt.Errorf("could not find xoxc token — page title: '%s', status: %d, url: %s\nMake sure you're logged in and the cookie is fresh", pageTitle, resp.StatusCode, resp.Request.URL)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("open", url)
	}
	cmd.Start()
}

func init() {
	authCmd.AddCommand(authLoginCmd)
}
