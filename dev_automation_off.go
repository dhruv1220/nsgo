//go:build nsgodev && !nsgoautomation

package nsgo

// startAutomation does nothing: only nsgo dev -automation's builds link the agent.
func startAutomation(string) {}
