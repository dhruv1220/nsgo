//go:build windows

package windows

import "testing"

func TestMenuMnemonic(t *testing.T) {
	for _, test := range []struct {
		label string
		want  rune
	}{
		{"File", 'f'},
		{"Édit", 'é'},
		{"E&xit", 'x'},
		{"Rock && Roll", 'r'},
		{"A && Beta", 'a'},
		{"123", '1'},
		{"---", 0},
	} {
		if got := menuMnemonic(test.label); got != test.want {
			t.Errorf("menuMnemonic(%q) = %q, want %q", test.label, got, test.want)
		}
	}
}
