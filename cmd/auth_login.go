package cmd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

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

		// Normalize
		workspace = strings.TrimPrefix(workspace, "https://")
		workspace = strings.TrimPrefix(workspace, "http://")
		workspace = strings.TrimSuffix(workspace, "/")

		token, cookie, err := extractViaLocalServer(workspace)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		cfg := &config.Config{Token: token, Cookie: cookie}

		fmt.Println("Validating credentials...")
		client := api.NewClient(cfg)
		resp, err := client.AuthTest()
		if err != nil {
			return fmt.Errorf("auth failed — tokens may be invalid: %w", err)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("\nAuthenticated as %s in %s\n", resp.User, resp.Team)
		fmt.Println("Config saved to ~/.slack-cli/config.json")
		return nil
	},
}

func extractViaLocalServer(workspace string) (string, string, error) {
	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	tokenCh := make(chan [2]string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()

	// This page runs in the browser after login — it extracts tokens and sends them back
	mux.HandleFunc("/extract", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>slack-cli — Extracting tokens...</title></head>
<body style="font-family: -apple-system, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px;">
<h2>slack-cli — Token Extraction</h2>
<div id="status">
<p>Extracting your Slack tokens...</p>
<p style="color: #666;">This will fetch your workspace page and pull the tokens automatically.</p>
</div>
<script>
async function extract() {
    var status = document.getElementById('status');
    try {
        // Fetch the workspace page — browser sends cookies automatically
        var resp = await fetch('https://%s/', { credentials: 'include' });
        var html = await resp.text();

        // Extract xoxc token from page source
        var tokenMatch = html.match(/"api_token"\s*:\s*"(xoxc-[^"]+)"/);
        if (!tokenMatch) {
            tokenMatch = html.match(/(xoxc-[a-zA-Z0-9-]+)/);
        }

        // Get the d cookie
        var cookies = document.cookie;
        // Can't read httpOnly cookies from JS — we'll get it from the fetch response
        // Instead, ask the user's workspace page which includes it in boot_data

        if (!tokenMatch) {
            status.innerHTML = '<p style="color:red;">Could not find xoxc token. Make sure you are logged into <strong>%s</strong> in this browser.</p>' +
                '<p><a href="https://%s/" target="_blank">Click here to log in</a>, then come back and <a href="/extract">try again</a>.</p>';
            return;
        }

        var token = tokenMatch[1];

        // Send token to local server — cookie will be extracted server-side
        status.innerHTML = '<p>Found token! Sending to slack-cli...</p>';

        var result = await fetch('/callback?token=' + encodeURIComponent(token));
        if (result.ok) {
            status.innerHTML = '<h3 style="color:green;">Done! You can close this tab.</h3>' +
                '<p>slack-cli has your credentials. Check your terminal.</p>';
        } else {
            var errText = await result.text();
            status.innerHTML = '<p style="color:red;">Error: ' + errText + '</p>';
        }
    } catch(e) {
        status.innerHTML = '<p style="color:red;">Error: ' + e.message + '</p>' +
            '<p>Make sure you are logged into <a href="https://%s/">%s</a> in this browser, then <a href="/extract">try again</a>.</p>';
    }
}
extract();
</script>
</body>
</html>`, workspace, workspace, workspace, workspace, workspace)
	})

	// Receive the token, then ask user to paste cookie manually (httpOnly cookies can't be read by JS)
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" || !strings.HasPrefix(token, "xoxc-") {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		tokenCh <- [2]string{token, ""}
	})

	// Cookie extraction page — gives user a simple way to copy the d cookie
	mux.HandleFunc("/cookie", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>slack-cli — Cookie</title></head>
<body style="font-family: -apple-system, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px;">
<h2>Almost done!</h2>
<p>The xoxc token was captured. Now we need the session cookie.</p>
<p>Open DevTools (Cmd+Option+I) → Application → Cookies → <strong>%s</strong> → copy the <code>d</code> value (starts with <code>xoxd-</code>).</p>
<p>Or run this in the DevTools Console on your Slack tab:</p>
<pre style="background:#f5f5f5;padding:12px;border-radius:4px;overflow-x:auto;">
// Copy this whole line:
copy(document.cookie.split(';').find(c=>c.trim().startsWith('d=')).trim().slice(2))
</pre>
<p>Then paste it in your terminal.</p>
</body>
</html>`, workspace)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	extractURL := fmt.Sprintf("http://127.0.0.1:%d/extract", port)

	fmt.Println()
	fmt.Println("Opening your browser...")
	fmt.Println("If it doesn't open, visit this URL manually:")
	fmt.Printf("  %s\n\n", extractURL)

	openBrowser(extractURL)

	// Wait for the token
	fmt.Println("Waiting for token extraction...")
	var tokenPair [2]string
	select {
	case tokenPair = <-tokenCh:
	case err := <-errCh:
		return "", "", err
	case <-time.After(5 * time.Minute):
		return "", "", fmt.Errorf("timed out waiting for login")
	}

	server.Shutdown(context.Background())

	xoxcToken := tokenPair[0]

	// The d cookie is httpOnly so JS can't read it — ask user to paste it
	fmt.Println()
	fmt.Println("Token captured! Now we need the session cookie.")
	fmt.Println()
	fmt.Println("In your browser, open DevTools (Cmd+Option+I) on your Slack tab:")
	fmt.Println("  → Application → Cookies → app.slack.com → copy the 'd' cookie value")
	fmt.Println("  (starts with xoxd-...)")
	fmt.Println()

	openBrowser(fmt.Sprintf("http://127.0.0.1:%d/cookie", port))

	// Restart server briefly for the cookie help page
	server2 := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go server2.ListenAndServe()
	defer server2.Shutdown(context.Background())

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Paste the d cookie value: ")
	cookie, _ := reader.ReadString('\n')
	cookie = strings.TrimSpace(cookie)

	if !strings.HasPrefix(cookie, "xoxd-") {
		return "", "", fmt.Errorf("cookie should start with 'xoxd-'")
	}

	return xoxcToken, cookie, nil
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
