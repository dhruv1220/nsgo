package updates

import "testing"

func TestWinSparkleLanguage(t *testing.T) {
	for _, test := range []struct {
		app, want string
	}{
		// Lower case: wxWidgets finds zh_CN in WinSparkle.dll as zh_cn only.
		{"zh-Hans", "zh_cn"},
		{"zh", "zh_cn"},
		{"zh-CN", "zh_cn"},
		{"zh-Hant", "zh_tw"},
		{"zh-TW", "zh_tw"},
		{"zh-HK", "zh_tw"},
		{"zh-Hant-CN", "zh_tw"},
		{"pt", "pt_br"},
		{"pt-BR", "pt_br"},
		{"pt-PT", "pt_pt"},
		{"sr", "sr"},
		{"sr-Latn", "sr@latin"},
		{"ca", "ca_es"},
		{"de-CH", "de"},
		{"ja", "ja"},
		{"nb", "nb"},
		{"no", "nb"},
		{"he", "he"},
		{"iw", "he"},
		// English is WinSparkle's own, so an English app on a Chinese Windows says English.
		{"en", "en"},
		{"en-GB", "en"},
		// WinSparkle has no Esperanto or Vietnamese: it follows the Windows UI language.
		{"eo", ""},
		{"vi", ""},
		{"", ""},
		{"not a tag", ""},
	} {
		if got := WinSparkleLanguage(test.app); got != test.want {
			t.Errorf("WinSparkleLanguage(%q) = %q, want %q", test.app, got, test.want)
		}
	}
}
