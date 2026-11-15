package updates

import "testing"

func TestWinSparkleLanguage(t *testing.T) {
	for _, test := range []struct {
		app, want string
	}{
		{"zh-Hans", "zh_CN"},
		{"zh-CN", "zh_CN"},
		{"zh-Hant-TW", "zh_TW"},
		{"pt-BR", "pt_BR"},
		{"pt-PT", "pt_PT"},
		{"sr-Latn", "sr@latin"},
		{"de-CH", "de"},
		{"en-US", ""},
		{"eo", ""},
	} {
		if got := WinSparkleLanguage(test.app); got != test.want {
			t.Errorf("WinSparkleLanguage(%q) = %q, want %q", test.app, got, test.want)
		}
	}
}
