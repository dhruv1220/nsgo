package updates

import (
	"strings"

	"golang.org/x/text/language"
)

// WinSparkleLanguage maps an app catalog language to a translation bundled
// with WinSparkle. An empty result leaves WinSparkle's default language in
// place, which is its English UI for languages it does not translate.
func WinSparkleLanguage(tag string) string {
	t, err := language.Parse(strings.TrimSpace(tag))
	if err != nil || t.IsRoot() {
		return ""
	}
	base, _ := t.Base()
	script, _ := t.Script()
	region, _ := t.Region()
	switch base.String() {
	case "zh":
		if script.String() == "Hans" || region.String() == "CN" || region.String() == "SG" {
			return "zh_CN"
		}
		if script.String() == "Hant" || region.String() == "TW" || region.String() == "HK" || region.String() == "MO" {
			return "zh_TW"
		}
		return ""
	case "pt":
		if region.String() == "BR" {
			return "pt_BR"
		}
		if region.String() == "PT" {
			return "pt_PT"
		}
		return ""
	case "sr":
		if script.String() == "Latn" {
			return "sr@latin"
		}
		return "sr"
	case "ca":
		return "ca_ES"
	case "fy":
		return "fy_NL"
	case "gl":
		return "gl_ES"
	default:
		if _, ok := winSparkleTranslations[base.String()]; ok {
			return base.String()
		}
		return ""
	}
}

// winSparkleTranslations is the set of language-only translation files
// shipped by WinSparkle. Regional languages are handled above because their
// file names carry the region or script.
var winSparkleTranslations = map[string]struct{}{
	"af": {}, "ar": {}, "be": {}, "bg": {}, "bs": {}, "co": {}, "cs": {},
	"da": {}, "de": {}, "el": {}, "es": {}, "et": {}, "eu": {}, "fi": {},
	"fr": {}, "ga": {}, "he": {}, "hr": {}, "hu": {}, "hy": {}, "id": {},
	"is": {}, "it": {}, "ja": {}, "ka": {}, "kk": {}, "ko": {}, "lt": {},
	"nb": {}, "nl": {}, "pl": {}, "ru": {}, "sk": {}, "sv": {}, "th": {},
	"tr": {}, "ug": {}, "uk": {}, "uz": {},
}
