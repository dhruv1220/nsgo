//go:build ios && nsgoembed

package nsgo

import "github.com/nsgo-dev/nsgo/internal/backend"

// StartEmbedded runs main, the app's own, on a goroutine of its own until
// it calls Run, then gets NSGoView ready, once. Only the code `nsgo build
// -embed` adds to the app calls it, on the host's main thread.
func StartEmbedded(main func()) { backend.StartEmbedded(main) }
