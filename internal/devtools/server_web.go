//go:build nsgodev && js && wasm && !nsgominiprogram

package devtools

import (
	"net/url"
	"strings"
	"syscall/js"
)

// pageToken is the token the page's address carries in its fragment, as #nsgo-token=, where nsgo dev's address for it puts it.
func pageToken() string {
	q, _ := url.ParseQuery(strings.TrimPrefix(js.Global().Get("location").Get("hash").String(), "#"))
	return q.Get("nsgo-token")
}
