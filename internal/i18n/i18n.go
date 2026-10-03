// Package i18n picks UI strings by language. The language is set once at
// startup, before any page runs.
package i18n

import "strings"

var en bool

// Set selects the language by BCP 47 tag: Chinese for zh*, English otherwise.
func Set(tag string) {
	en = !strings.HasPrefix(strings.ToLower(strings.TrimSpace(tag)), "zh")
}

// EN reports whether the language is English.
func EN() bool { return en }

// Tag returns the BCP 47 tag of the language.
func Tag() string { return T("zh-TW", "en") }

// T returns zh or en by the language.
func T(zh, en string) string {
	if EN() {
		return en
	}
	return zh
}
