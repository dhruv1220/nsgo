//go:build nsgodev && !(js && wasm && !nsgominiprogram)

package devtools

// pageToken is "": no web page runs the app.
func pageToken() string { return "" }
