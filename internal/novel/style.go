package novel

import "github.com/voilelab/epub_toolbox/internal/i18n"

// Style is a layout template for the book body.
type Style int

const (
	StyleReader     Style = iota // leave layout to the reader app
	StyleHorizontal              // Chinese, horizontal
	StyleVertical                // Chinese, vertical right to left
)

// Styles lists every Style in UI order.
var Styles = []Style{StyleReader, StyleHorizontal, StyleVertical}

// Name is the UI label.
func (s Style) Name() string {
	switch s {
	case StyleHorizontal:
		return i18n.T("中文橫排", "Chinese, horizontal")
	case StyleVertical:
		return i18n.T("中文直排", "Chinese, vertical")
	}
	return i18n.T("閱讀器預設", "Reader default")
}

// Desc describes the style for the UI.
func (s Style) Desc() string {
	switch s {
	case StyleHorizontal:
		return i18n.T("段落首行縮排兩字、無段距，章節標題置中。",
			"Two-character first-line indent, no paragraph gap, centered chapter titles.")
	case StyleVertical:
		return i18n.T("由上而下、由右而左直排，向左翻頁；段落首行縮排兩字。閱讀器需支援直排。",
			"Top to bottom, right to left, pages turn left; two-character indent. The reader app must support vertical text.")
	}
	return i18n.T("不設定內文樣式，由閱讀器決定。", "No body styling; the reader app decides.")
}

// WritingMode is the epub.Book WritingMode for s.
func (s Style) WritingMode() string {
	if s == StyleVertical {
		return "vertical-rl"
	}
	return ""
}

// CSS is the stylesheet for s.
// Font, size and colors are left to the reader so its settings and night mode still work.
func (s Style) CSS() string {
	switch s {
	case StyleHorizontal:
		return baseCSS + `p { text-indent: 2em; margin: 0; }
h1 { text-align: center; margin: 1em 0 2em; }
`
	case StyleVertical:
		// Apple Books only honors writing-mode on <html>.
		return baseCSS + `html { -epub-writing-mode: vertical-rl; -webkit-writing-mode: vertical-rl; writing-mode: vertical-rl; }
p { text-indent: 2em; margin: 0; }
h1 { margin: 0 1em 0 2em; }
`
	}
	return baseCSS
}

const baseCSS = `.cover { text-align: center; }
.cover img { max-width: 100%; max-height: 100vh; }
`
