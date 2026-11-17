//go:build linux && !android

package linux

import "testing"

func TestAccessibleStatesUseGTKCollectTypes(t *testing.T) {
	for _, state := range []string{"selected", "checked", "pressed", "expanded", "invalid"} {
		if !stateValueIsInt(state) {
			t.Errorf("%s must use GTK's integer GValue", state)
		}
	}
	for _, state := range []string{"disabled", "hidden", "busy"} {
		if stateValueIsInt(state) {
			t.Errorf("%s must use a boolean GValue", state)
		}
	}
}
