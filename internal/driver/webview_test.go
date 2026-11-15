package driver

import "testing"

func TestWebOrigin(t *testing.T) {
	for raw, want := range map[string]string{
		"https://Example.COM/a?b#c":       "https://example.com",
		"https://example.com:443/":        "https://example.com",
		"http://example.com:8080/":        "http://example.com:8080",
		"http://user:pw@[::1]:80/x":       "http://[::1]",
		"app://bundle/index.html":         "app://bundle",
		"file:///Users/me/app/index.html": "file://",
		"blob:https://example.com/1234":   "https://example.com",
		"about:blank":                     "null",
		"about:srcdoc":                    "null",
		"data:text/html,<p>":              "null",
		"":                                "",
		"example.com":                     "null",
		"https://exa mple.com/":           "null",
	} {
		if got := WebOrigin(raw); got != want {
			t.Errorf("WebOrigin(%q) = %q, want %q", raw, got, want)
		}
	}
}
