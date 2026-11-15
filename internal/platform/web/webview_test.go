//go:build js && wasm

package web

import (
	"syscall/js"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

// TestWebViewDropsTheBridgeOfAPageThatLeft loads HTML whose page goes to another, which has no bridge, and comes back as the app reloads it. An Evaluate made before the HTML's page handed its port waits for it; one the page had not answered as it left ends with an error, as does one made while the other page shows; and the HTML shown again answers again. The other page reaches no OnMessage. Run in a browser: nsgo test -target web.
func TestWebViewDropsTheBridgeOfAPageThatLeft(t *testing.T) {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		t.Skip("no document here: run with nsgo test -target web")
	}
	mount := doc.Call("createElement", "div")
	mount.Set("id", "nsgo-webview-test")
	setStyle(mount, map[string]string{"width": "200px", "height": "200px"})
	doc.Get("body").Call("appendChild", mount)
	ref := new(core.Ref)
	scope := make(chan *core.Scope, 1)
	var loads, starts int
	var answers, messages []string
	open(func(cx *core.Scope) core.Node {
		scope <- cx
		return core.WebView{Ref: ref, Source: core.WebHTML(`<script>window.nsgo.postMessage("html");</script>`, ""),
			MessageOrigins: []string{"*"},
			OnLoadStart:    func(core.WebNavigation) { starts++ },
			OnLoad:         func(core.WebNavigation) { loads++ },
			OnMessage:      func(m string) { messages = append(messages, m) },
			Style:          core.Style{Width: core.Pt(100), Height: core.Pt(100)}}
	}, Options{Mount: "#nsgo-webview-test"}, nil)
	cx := <-scope
	wait := func(what string, cond func() bool) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); !cond(); time.Sleep(10 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("%s: no answer after 5s (answers %q, %d loads)", what, answers, loads)
			}
		}
	}
	evaluate := func(script string) {
		cx.Post(func() {
			ref.Evaluate(script, func(json string, err error) {
				if err != nil {
					json = "error"
				}
				answers = append(answers, json)
			})
		})
	}
	answered := func(what, want string) {
		t.Helper()
		wait(what, func() bool { return len(answers) > 0 })
		if answers[0] != want {
			t.Fatalf("%s answered %q, want %q", what, answers[0], want)
		}
		answers = answers[1:]
	}

	// A load start ends what waits, so the Evaluate follows the HTML's.
	wait("the HTML's load start", func() bool { return starts == 1 })
	evaluate("1 + 1")
	answered("an Evaluate as the page loads", "2")
	wait("the HTML's load", func() bool { return loads == 1 })

	evaluate("new Promise(() => {})")
	time.Sleep(50 * time.Millisecond)
	cx.Post(func() {
		ref.InjectJavaScript(`location.href = "data:text/html,<script>window.nsgo && window.nsgo.postMessage('other');parent.postMessage('other', '*');</script>"`)
	})
	answered("an Evaluate the page left", "error")
	wait("the other page's load", func() bool { return loads == 2 })
	evaluate("3")
	answered("an Evaluate on the other page", "error")
	if starts != 2 {
		t.Errorf("%d load starts, want the source's and the page leaving", starts)
	}

	cx.Post(func() { ref.Reload() })
	wait("the HTML's load again", func() bool { return loads == 3 })
	evaluate("4")
	answered("an Evaluate on the HTML again", "4")
	if len(messages) != 2 || messages[0] != "html" || messages[1] != "html" {
		t.Errorf("OnMessage heard %q, want the HTML's page's own twice", messages)
	}
}
