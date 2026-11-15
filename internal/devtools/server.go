//go:build nsgodev

package devtools

import (
	"io"
	"net/http"
	"os"
)

// ServerToken is the run's token for nsgo dev's server, which the dev client sends with every request (Fetch): the tool gives each run of the app its own in NSGO_DEV_TOKEN, which is taken out of the environment so the programs the app starts never inherit it, or through -ldflags -X for targets that pass the app no environment; a web page reads it from its address's fragment.
var ServerToken = ""

func init() {
	if t := os.Getenv("NSGO_DEV_TOKEN"); t != "" {
		ServerToken = t
	}
	os.Unsetenv("NSGO_DEV_TOKEN")
	if ServerToken == "" {
		ServerToken = pageToken()
	}
}

// Fetch sends nsgo dev's server a request with ServerToken, and body as contentType when it has one.
func Fetch(method, url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-NSGo-Token", ServerToken)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return http.DefaultClient.Do(req)
}
