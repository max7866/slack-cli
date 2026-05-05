package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/max7866/slack-cli/internal/api"
	"github.com/max7866/slack-cli/internal/config"
	"github.com/spf13/cobra"
	webview "github.com/webview/webview_go"
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to Slack via a native browser window",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Enter your Slack workspace URL (e.g., mycompany.slack.com): ")
		workspace, _ := reader.ReadString('\n')
		workspace = strings.TrimSpace(workspace)
		workspace = strings.TrimPrefix(workspace, "https://")
		workspace = strings.TrimPrefix(workspace, "http://")
		workspace = strings.TrimSuffix(workspace, "/")

		fmt.Println()
		fmt.Println("Opening Slack login window...")
		fmt.Println("Sign in normally. The window will close automatically once tokens are captured.")

		cookie, err := loginViaWebview(workspace)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Use the cookie to extract the xoxc token server-side
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

func loginViaWebview(workspace string) (string, error) {
	// Start a local server to receive the cookie from the webview
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	cookieCh := make(chan string, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var data struct {
			Cookie string `json:"cookie"`
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &data)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))

		if strings.HasPrefix(data.Cookie, "xoxd-") {
			cookieCh <- data.Cookie
		}
	})

	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go server.ListenAndServe()

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/cookie", port)

	// JavaScript that polls for the d cookie and sends it to our local server
	pollScript := fmt.Sprintf(`
		(function poll() {
			var cookies = document.cookie.split(';');
			for (var i = 0; i < cookies.length; i++) {
				var c = cookies[i].trim();
				if (c.startsWith('d=xoxd-')) {
					var val = c.substring(2);
					fetch('%s', {
						method: 'POST',
						headers: {'Content-Type': 'application/json'},
						body: JSON.stringify({cookie: val})
					});
					return;
				}
			}
			setTimeout(poll, 1000);
		})();
	`, callbackURL)

	// Open native webview window
	w := webview.New(false)
	defer w.Destroy()

	w.SetTitle("slack-cli — Sign in to Slack")
	w.SetSize(1024, 768, webview.HintNone)

	// Inject the polling script after every navigation
	w.Init(pollScript)

	w.Navigate(fmt.Sprintf("https://%s/", workspace))

	// Check for cookie in background, close webview when found
	go func() {
		<-cookieCh
		w.Dispatch(func() {
			w.Terminate()
		})
	}()

	// This blocks until the window is closed
	w.Run()

	server.Shutdown(nil)

	select {
	case cookie := <-cookieCh:
		return cookie, nil
	default:
		return "", fmt.Errorf("window closed before login completed")
	}
}

// extractToken fetches the Slack workspace page using the d cookie
// and extracts the xoxc API token from the HTML response.
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

func init() {
	authCmd.AddCommand(authLoginCmd)
}
