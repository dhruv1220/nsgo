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
		{"Файл", 'ф'},
		// NSGo's labels mark no access key: '&' shows as itself.
		{"E&xit", 'e'},
		{"&Help", 'h'},
		{"(Beta) Tools", 'b'},
		{"123", '1'},
		{"---", 0},
	} {
		if got := menuMnemonic(test.label); got != test.want {
			t.Errorf("menuMnemonic(%q) = %q, want %q", test.label, got, test.want)
		}
	}
}
