//go:build js

package web

import (
	"io"
	"syscall/js"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/internal/driver"
)

// Run in a browser, which makes and fetches blob: URLs: nsgo test -target web.
func TestBlobFileReadsDroppedAndFetchedBlobs(t *testing.T) {
	if !js.Global().Get("document").Truthy() {
		t.Skip("no browser here: run with nsgo test -target web")
	}
	if driver.BlobFile == nil {
		t.Fatal("the web backend installs no driver.BlobFile")
	}
	when := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)
	file := js.Global().Get("File").New([]any{"dropped bytes"}, "a b#1.md", map[string]any{"type": "text/markdown", "lastModified": when.UnixMilli()})
	raw := droppedURL(file)
	if want := "#a%20b%231.md"; len(raw) < len(want) || raw[len(raw)-len(want):] != want {
		t.Fatalf("droppedURL = %q, want its name as the fragment", raw)
	}
	read := func(f driver.File) string {
		t.Helper()
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer r.Close()
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	f := driver.BlobFile(raw[:len(raw)-len("#a%20b%231.md")], "a b#1.md")
	if f.Name != "a b#1.md" || f.MIME != "text/markdown" || f.Size != int64(len("dropped bytes")) || !f.Modified.Equal(when) || f.Path != "" {
		t.Errorf("a dropped blob = %+v", f)
	}
	if got := read(f); got != "dropped bytes" {
		t.Errorf("a dropped blob reads %q", got)
	}

	// A blob: URL no drop made is fetched.
	blob := js.Global().Get("Blob").New([]any{"made by the page"}, map[string]any{"type": "text/plain"})
	made := js.Global().Get("URL").Call("createObjectURL", blob).String()
	f = driver.BlobFile(made, "notes.txt")
	if f.Name != "notes.txt" || f.MIME != "text/plain" || f.Size != 0 {
		t.Errorf("a fetched blob = %+v", f)
	}
	if got := read(f); got != "made by the page" {
		t.Errorf("a fetched blob reads %q", got)
	}
	js.Global().Get("URL").Call("revokeObjectURL", made)
	if _, err := driver.BlobFile(made, "notes.txt").Open(); err == nil {
		t.Error("a revoked blob: URL opened")
	}
}
