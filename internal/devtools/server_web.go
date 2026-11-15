//go:build nsgodev && js && wasm && !nsgominiprogram

package devtools

import (
	"net/url"
	"strings"
	"syscall/js"
)

// pageToken is the token nsgo dev's address for the page carries in its fragment, as #nsgo-token=, or, once the page's script has moved it out of the address, the tab's sessionStorage holds as nsgo-token.
func pageToken() string {
	q, _ := url.ParseQuery(strings.TrimPrefix(js.Global().Get("location").Get("hash").String(), "#"))
	if t := q.Get("nsgo-token"); t != "" {
		return t
	}
	// A browser that keeps no storage for the page throws on reading sessionStorage, which a call, unlike a read, turns into a panic.
	defer func() { recover() }()
	storage := js.Global().Get("Reflect").Call("get", js.Global(), "sessionStorage")
	if t := storage.Call("getItem", "nsgo-token"); t.Type() == js.TypeString {
		return t.String()
	}
	return ""
}
