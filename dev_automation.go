//go:build nsgodev && nsgoautomation

package nsgo

import (
	"github.com/nsgo-dev/nsgo/internal/devtools"
	// The agent nsgo dev's automation drives the app through (nsgotest's automation.go), which sets devtools.Automation.
	_ "github.com/nsgo-dev/nsgo/nsgotest"
)

// startAutomation has the agent answer the automation over DevTools' channel, when nsgo dev gave the app a token (startDevClient); without one, it follows nothing.
func startAutomation(server string) {
	if devtools.Token() == "" || devtools.Automation == nil {
		devtools.Automation = nil
		return
	}
	openDevChannel(server)
}
