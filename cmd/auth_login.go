package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/spf13/cobra"
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Slack via browser (auto-extracts tokens)",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter your Slack workspace URL (e.g., mycompany.slack.com): ")
		workspace, _ := reader.ReadString('\n')
		workspace = strings.TrimSpace(workspace)

		// Normalize the URL
		if !strings.Contains(workspace, "://") {
			workspace = "https://" + workspace
		}
		if !strings.HasSuffix(workspace, "/") {
			workspace = workspace + "/"
		}

		fmt.Println()
		fmt.Println("Opening browser for Slack sign-in...")
		fmt.Println("Please log in normally. The window will close automatically once tokens are captured.")
		fmt.Println()

		token, cookie, err := extractTokensFromBrowser(workspace)
		if err != nil {
			return fmt.Errorf("failed to extract tokens: %w", err)
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

func extractTokensFromBrowser(workspaceURL string) (token string, cookie string, err error) {
	// Launch Chrome with a visible window (not headless)
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.WindowSize(1024, 768),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for the entire login flow
	ctx, cancel = context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Navigate to the workspace
	if err := chromedp.Run(ctx, chromedp.Navigate(workspaceURL)); err != nil {
		return "", "", fmt.Errorf("failed to open browser: %w", err)
	}

	// Poll until we find the xoxd cookie and xoxc token
	tokenPattern := regexp.MustCompile(`"api_token"\s*:\s*"(xoxc-[^"]+)"`)

	for {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("timed out waiting for login (5 min). Try again or use 'auth setup' for manual entry")
		default:
		}

		// Check for the xoxd cookie
		var cookies []*network.Cookie
		if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		})); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		xoxdCookie := ""
		for _, c := range cookies {
			if c.Name == "d" && strings.HasPrefix(c.Value, "xoxd-") {
				xoxdCookie = c.Value
				break
			}
		}

		if xoxdCookie == "" {
			time.Sleep(2 * time.Second)
			continue
		}

		// Cookie found — now extract the xoxc token from the page
		var pageHTML string
		if err := chromedp.Run(ctx, chromedp.InnerHTML("html", &pageHTML)); err != nil {
			time.Sleep(2 * time.Second)
			continue
		}

		matches := tokenPattern.FindStringSubmatch(pageHTML)
		if len(matches) < 2 {
			// Token might not be on this page yet — try evaluating JS
			var jsToken string
			err := chromedp.Run(ctx, chromedp.Evaluate(`
				(function() {
					try {
						// Try boot_data
						if (window.boot_data && window.boot_data.api_token) {
							return window.boot_data.api_token;
						}
						// Try localStorage
						for (var i = 0; i < localStorage.length; i++) {
							var key = localStorage.key(i);
							var val = localStorage.getItem(key);
							if (val && val.indexOf('xoxc-') !== -1) {
								var match = val.match(/(xoxc-[a-zA-Z0-9-]+)/);
								if (match) return match[1];
							}
						}
						// Try page source
						var bodyMatch = document.body.innerHTML.match(/"api_token"\s*:\s*"(xoxc-[^"]+)"/);
						if (bodyMatch) return bodyMatch[1];
						return "";
					} catch(e) { return ""; }
				})()
			`, &jsToken))
			if err == nil && strings.HasPrefix(jsToken, "xoxc-") {
				fmt.Println("Tokens captured successfully!")
				return jsToken, xoxdCookie, nil
			}
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Println("Tokens captured successfully!")
		return matches[1], xoxdCookie, nil
	}
}

func init() {
	authCmd.AddCommand(authLoginCmd)
}
