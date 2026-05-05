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

		fmt.Println()
		fmt.Println("Opening Slack in your browser...")
		openBrowser(fmt.Sprintf("https://%s/", workspace))

		fmt.Println()
		fmt.Println("Once you're signed in, grab the 'd' cookie:")
		fmt.Println("  1. On your Slack tab, press Cmd+Option+I (DevTools)")
		fmt.Println("  2. Click Application -> Cookies -> app.slack.com")
		fmt.Println("  3. Find the cookie named 'd' (starts with xoxd-)")
		fmt.Println("  4. Double-click its value, copy it")
		fmt.Println()

		fmt.Print("Paste the d cookie: ")
		cookie, _ := reader.ReadString('\n')
		cookie = strings.TrimSpace(cookie)

		if !strings.HasPrefix(cookie, "xoxd-") {
			return fmt.Errorf("cookie should start with 'xoxd-'")
		}

		fmt.Println("\nExtracting API token...")
		token, err := extractToken(workspace, cookie)
		if err != nil {
			return err
		}

		cfg := &config.Config{Token: token, Cookie: cookie}

		fmt.Println("Validating credentials...")
		client := api.NewClient(cfg)
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nAuthenticated as %s in %s\n", resp.User, resp.Team)
		fmt.Println("Config saved to ~/.slack-cli/config.json")
		return nil
	},
}

func extractToken(workspace string, cookie string) (string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/", workspace), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", fmt.Sprintf("d=%s", cookie))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch workspace page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	tokenPattern := regexp.MustCompile(`"api_token"\s*:\s*"(xoxc-[^"]+)"`)
	matches := tokenPattern.FindSubmatch(body)
	if len(matches) < 2 {
		broadPattern := regexp.MustCompile(`(xoxc-[a-zA-Z0-9-]+)`)
		matches = broadPattern.FindSubmatch(body)
		if len(matches) < 2 {
			return "", fmt.Errorf("could not find xoxc token — make sure you're logged in")
		}
	}

	fmt.Println("Token found!")
	return string(matches[1]), nil
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
