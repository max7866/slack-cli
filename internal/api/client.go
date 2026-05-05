package api

import (
	"fmt"
	"net/http"

	"github.com/max7866/slack-cli/internal/config"
	"github.com/slack-go/slack"
)

// cookieClient injects the xoxd session cookie on every request.
type cookieClient struct {
	inner  *http.Client
	cookie string
}

func (c *cookieClient) Do(req *http.Request) (*http.Response, error) {
	if c.cookie != "" {
		req.Header.Set("Cookie", fmt.Sprintf("d=%s", c.cookie))
	}
	return c.inner.Do(req)
}

// NewClient creates a Slack API client using xoxc token + xoxd cookie.
func NewClient(ws *config.Workspace) *slack.Client {
	httpClient := &cookieClient{
		inner:  http.DefaultClient,
		cookie: ws.Cookie,
	}
	return slack.New(ws.Token, slack.OptionHTTPClient(httpClient))
}
