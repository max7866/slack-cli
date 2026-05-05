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

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	mux := http.NewServeMux()

	// Instruction page — tells the user what to do
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>slack-cli — Login</title></head>
<body style="font-family:-apple-system,sans-serif;max-width:650px;margin:50px auto;padding:20px;line-height:1.6;">
<h2>slack-cli — Authenticate</h2>

<p><strong>Step 1:</strong> Make sure you're logged into
<a href="https://%s/" target="_blank">%s</a> in this browser.</p>

<p><strong>Step 2:</strong> Once logged in, open <strong>any Slack page</strong> in this browser,
then open DevTools (<code>Cmd+Option+I</code>) and go to the <strong>Console</strong> tab.</p>

<p><strong>Step 3:</strong> Paste this entire snippet into the Console and press Enter:</p>

<pre id="snippet" style="background:#1e1e1e;color:#d4d4d4;padding:16px;border-radius:8px;overflow-x:auto;font-size:13px;cursor:pointer;" onclick="copySnippet()">
<span style="color:#6a9955;">// slack-cli token extractor</span>
<span style="color:#c586c0;">var</span> t = <span style="color:#569cd6;">document</span>.body.innerHTML.<span style="color:#dcdcaa;">match</span>(<span style="color:#d16969;">/"api_token"\s*:\s*"(xoxc-[^"]+)"/</span>);
<span style="color:#c586c0;">var</span> c = <span style="color:#569cd6;">document</span>.cookie.<span style="color:#dcdcaa;">split</span>(<span style="color:#ce9178;">';'</span>).<span style="color:#dcdcaa;">find</span>(<span style="color:#569cd6;">x</span>=&gt;x.<span style="color:#dcdcaa;">trim</span>().<span style="color:#dcdcaa;">startsWith</span>(<span style="color:#ce9178;">'d='</span>));
<span style="color:#c586c0;">if</span>(t &amp;&amp; c) {
  <span style="color:#dcdcaa;">fetch</span>(<span style="color:#ce9178;">'%s?token='</span>+<span style="color:#dcdcaa;">encodeURIComponent</span>(t[<span style="color:#b5cea8;">1</span>])+<span style="color:#ce9178;">'&amp;cookie='</span>+<span style="color:#dcdcaa;">encodeURIComponent</span>(c.<span style="color:#dcdcaa;">trim</span>().<span style="color:#dcdcaa;">slice</span>(<span style="color:#b5cea8;">2</span>)));
  <span style="color:#ce9178;">'Sent! Check your terminal.'</span>;
} <span style="color:#c586c0;">else if</span>(!t) {
  <span style="color:#ce9178;">'ERROR: Could not find xoxc token. Are you on a Slack page?'</span>;
} <span style="color:#c586c0;">else</span> {
  <span style="color:#ce9178;">'ERROR: Could not find d cookie. Try the manual steps below.'</span>;
}
</pre>

<p style="color:#666;font-size:13px;">Click the snippet to copy it to clipboard.</p>

<div id="copied" style="display:none;color:green;font-weight:bold;">Copied!</div>

<hr style="margin:24px 0;border:none;border-top:1px solid #e0e0e0;">

<details>
<summary style="cursor:pointer;font-weight:bold;">Manual fallback (if the d cookie isn't found)</summary>
<p>The <code>d</code> cookie is sometimes httpOnly and can't be read by JavaScript.
In that case the snippet above will fail on the cookie part.</p>
<p>Go to DevTools → <strong>Application</strong> → <strong>Cookies</strong> → <code>app.slack.com</code> →
find the <code>d</code> cookie (starts with <code>xoxd-</code>) and copy its value.</p>
<p>Then run this in the Console:</p>
<pre style="background:#f5f5f5;padding:12px;border-radius:4px;font-size:13px;">
var t = document.body.innerHTML.match(/"api_token"\s*:\s*"(xoxc-[^"]+)"/);
fetch('%s?token='+encodeURIComponent(t[1])+'&cookie='+encodeURIComponent('PASTE_XOXD_COOKIE_HERE'));
</pre>
</details>

<script>
function copySnippet() {
  var text = "var t = document.body.innerHTML.match(/\"api_token\"\\s*:\\s*\"(xoxc-[^\"]+ )\"/); var c = document.cookie.split(';').find(x=>x.trim().startsWith('d=')); if(t && c) { fetch('%s?token='+encodeURIComponent(t[1])+'&cookie='+encodeURIComponent(c.trim().slice(2))); 'Sent! Check your terminal.'; } else if(!t) { 'ERROR: Could not find xoxc token. Are you on a Slack page?'; } else { 'ERROR: Could not find d cookie.'; }";
  navigator.clipboard.writeText(text);
  document.getElementById('copied').style.display = 'block';
  setTimeout(function(){ document.getElementById('copied').style.display = 'none'; }, 2000);
}
</script>

</body>
</html>`, workspace, workspace, callbackURL, callbackURL, callbackURL)
	})

	// Callback endpoint — receives token and cookie from the browser snippet
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Allow cross-origin from Slack domain
		w.Header().Set("Access-Control-Allow-Origin", "https://app.slack.com")
		w.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("https://%s", workspace))
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		token := r.URL.Query().Get("token")
		cookie := r.URL.Query().Get("cookie")

		if token == "" || !strings.HasPrefix(token, "xoxc-") {
			http.Error(w, "missing or invalid token", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok — check your terminal"))
		tokenCh <- [2]string{token, cookie}
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

	instructionURL := fmt.Sprintf("http://127.0.0.1:%d/", port)

	fmt.Println()
	fmt.Println("Opening your browser with instructions...")
	fmt.Println("If it doesn't open, visit:")
	fmt.Printf("  %s\n\n", instructionURL)

	openBrowser(instructionURL)

	fmt.Println("Waiting for tokens... (paste the snippet in your Slack tab's DevTools Console)")

	// Wait for tokens
	var tokenPair [2]string
	select {
	case tokenPair = <-tokenCh:
	case err := <-errCh:
		return "", "", err
	case <-time.After(5 * time.Minute):
		return "", "", fmt.Errorf("timed out waiting for tokens (5 min)")
	}

	server.Shutdown(context.Background())

	xoxcToken := tokenPair[0]
	xoxdCookie := tokenPair[1]

	// If cookie was captured, we're done
	if strings.HasPrefix(xoxdCookie, "xoxd-") {
		fmt.Println("\nTokens captured!")
		return xoxcToken, xoxdCookie, nil
	}

	// Cookie wasn't found by JS (httpOnly) — ask user to paste it
	fmt.Println()
	fmt.Println("Token captured! But the d cookie is httpOnly and couldn't be read by JavaScript.")
	fmt.Println()
	fmt.Println("In DevTools on your Slack tab:")
	fmt.Println("  Application -> Cookies -> app.slack.com -> copy the 'd' value (starts with xoxd-)")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Paste the d cookie value: ")
	cookie, _ := reader.ReadString('\n')
	xoxdCookie = strings.TrimSpace(cookie)

	if !strings.HasPrefix(xoxdCookie, "xoxd-") {
		return "", "", fmt.Errorf("cookie should start with 'xoxd-'")
	}

	return xoxcToken, xoxdCookie, nil
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
