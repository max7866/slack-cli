package api

import (
	"fmt"
	"net/http"

	"github.com/max7866/slack-cli/internal/config"
	"github.com/slack-go/slack"
)

// cookieClient injects the xoxd session cookie and browser User-Agent on every request.
type cookieClient struct {
	inner     *http.Client
	cookie    string
	userAgent string
}

func (c *cookieClient) Do(req *http.Request) (*http.Response, error) {
	if c.cookie != "" {
		req.Header.Set("Cookie", fmt.Sprintf("d=%s", c.cookie))
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	return c.inner.Do(req)
}

// NewClient creates a Slack API client using xoxc token + xoxd cookie.
// If the workspace has a stored User-Agent, it's sent on every request
// to avoid SSO/SAML session invalidation.
func NewClient(ws *config.Workspace) *slack.Client {
	httpClient := &cookieClient{
		inner:     http.DefaultClient,
		cookie:    ws.Cookie,
		userAgent: ws.UserAgent,
	}
	return slack.New(ws.Token, slack.OptionHTTPClient(httpClient))
}
