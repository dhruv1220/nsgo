package updates

import (
	"strings"

	"golang.org/x/text/language"
)

// WinSparkleLanguage maps the catalog language an app shows to the name WinSparkle's win_sparkle_set_lang takes for a translation WinSparkle 0.9.4 ships, or to "en" for English, the language its own strings are written in. An empty result, for an app without catalogs or a language WinSparkle does not translate, leaves WinSparkle to follow the Windows UI language, as it does when no language is set: once one is set, WinSparkle shows English rather than that default when it lacks the translation.
//
// The names are in lower case, as zh_cn: wxWidgets lists the translations WinSparkle.dll holds by their resource names, which it lowers, and compares the language set with them case-sensitively, so zh_CN, as the translation's file is named, finds none and shows English.
func WinSparkleLanguage(tag string) string {
	t, err := language.Parse(strings.TrimSpace(tag))
	if err != nil || t.IsRoot() {
		return ""
	}
	// Script and Region infer the likely ones a tag leaves out: zh is Hans, zh-TW Hant and pt BR, as CLDR says.
	base, _ := t.Base()
	script, _ := t.Script()
	region, _ := t.Region()
	switch base.String() {
	case "en":
		return "en"
	case "zh":
		switch script.String() {
		case "Hans":
			return "zh_cn"
		case "Hant":
			return "zh_tw"
		}
		return ""
	case "pt":
		if region.String() == "PT" {
			return "pt_pt"
		}
		return "pt_br"
	case "sr":
		if script.String() == "Latn" {
			return "sr@latin"
		}
		return "sr"
	case "no":
		// Norwegian's written Bokmål, the translation WinSparkle has.
		return "nb"
	case "ca":
		return "ca_es"
	case "fy":
		return "fy_nl"
	case "gl":
		return "gl_es"
	}
	if _, ok := winSparkleTranslations[base.String()]; ok {
		return base.String()
	}
	return ""
}

// winSparkleTranslations are the translations WinSparkle 0.9.4 ships named by language alone (its translations directory); those named with a region or script are mapped above.
var winSparkleTranslations = map[string]struct{}{
	"af": {}, "ar": {}, "be": {}, "bg": {}, "bs": {}, "co": {}, "cs": {},
	"da": {}, "de": {}, "el": {}, "es": {}, "et": {}, "eu": {}, "fi": {},
	"fr": {}, "ga": {}, "he": {}, "hr": {}, "hu": {}, "hy": {}, "id": {},
	"is": {}, "it": {}, "ja": {}, "ka": {}, "kk": {}, "ko": {}, "lt": {},
	"nb": {}, "nl": {}, "pl": {}, "ru": {}, "sk": {}, "sv": {}, "th": {},
	"tr": {}, "ug": {}, "uk": {}, "uz": {},
}
