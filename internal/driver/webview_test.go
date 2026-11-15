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

func TestWebSourceShows(t *testing.T) {
	for _, c := range []struct {
		source WebSource
		url    string
		want   bool
	}{
		{WebSource{HTML: "<p>"}, "about:blank", true},
		{WebSource{HTML: "<p>"}, "about:blank#top", true},
		{WebSource{HTML: "<p>"}, "about:srcdoc", true},
		{WebSource{HTML: "<p>"}, "data:text/html,<p>", false},
		{WebSource{HTML: "<p>"}, "blob:null/1234", false},
		{WebSource{HTML: "<p>"}, "", false},
		{WebSource{HTML: "<p>", BaseURL: "app://bundle/"}, "app://bundle/", true},
		{WebSource{HTML: "<p>", BaseURL: "app://bundle/"}, "about:blank", false},
		{WebSource{Src: "data:text/html,<p>hi</p>"}, "data:text/html,%3Cp%3Ehi%3C/p%3E", true},
		{WebSource{Src: "data:text/html,<p>hi</p>"}, "data:text/html,<p>evil</p>", false},
		{WebSource{Src: "data:text/html,<p>hi</p>"}, "about:blank", false},
		{WebSource{}, "about:blank", false},
	} {
		if got := c.source.Shows(c.url); got != c.want {
			t.Errorf("%+v.Shows(%q) = %v, want %v", c.source, c.url, got, c.want)
		}
	}
}
