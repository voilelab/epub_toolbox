package novel

import (
	"strings"

	"github.com/voilelab/epub_toolbox/internal/i18n"
)

// Style is the body layout. The zero value leaves layout to the reader app.
// The options never set font, size or colors so reader settings and night mode still work.
type Style struct {
	Indent      bool // two-character first-line indent
	NoGap       bool // no space between paragraphs
	Justify     bool
	CenterTitle bool
	Vertical    bool   // vertical right to left, pages turn left
	Extra       string // custom CSS appended last, so it wins
}

// Template is a preset Style.
type Template int

const (
	TemplateReader Template = iota
	TemplateHorizontal
	TemplateVertical
)

// Templates lists every Template in UI order.
var Templates = []Template{TemplateReader, TemplateHorizontal, TemplateVertical}

// Style is the preset.
func (t Template) Style() Style {
	switch t {
	case TemplateHorizontal:
		return Style{Indent: true, NoGap: true, CenterTitle: true}
	case TemplateVertical:
		return Style{Indent: true, NoGap: true, Vertical: true}
	}
	return Style{}
}

// Name is the UI label.
func (t Template) Name() string {
	switch t {
	case TemplateHorizontal:
		return i18n.T("中文橫排", "Chinese, horizontal")
	case TemplateVertical:
		return i18n.T("中文直排", "Chinese, vertical")
	}
	return i18n.T("閱讀器預設", "Reader default")
}

// Desc describes the template for the UI.
func (t Template) Desc() string {
	switch t {
	case TemplateHorizontal:
		return i18n.T("段落首行縮排兩字、無段距，章節標題置中。",
			"Two-character first-line indent, no paragraph gap, centered chapter titles.")
	case TemplateVertical:
		return i18n.T("由上而下、由右而左直排，向左翻頁；段落首行縮排兩字。閱讀器需支援直排。",
			"Top to bottom, right to left, pages turn left; two-character indent. The reader app must support vertical text.")
	}
	return i18n.T("不設定內文樣式，由閱讀器決定。", "No body styling; the reader app decides.")
}

// WritingMode is the epub.Book WritingMode for s.
func (s Style) WritingMode() string {
	if s.Vertical {
		return "vertical-rl"
	}
	return ""
}

// CSS is the stylesheet for s.
func (s Style) CSS() string {
	var sb strings.Builder
	sb.WriteString(baseCSS)
	if s.Vertical {
		// Apple Books only honors writing-mode on <html>.
		sb.WriteString("html { -epub-writing-mode: vertical-rl; -webkit-writing-mode: vertical-rl; writing-mode: vertical-rl; }\n")
	}

	var p []string
	if s.Indent {
		p = append(p, "text-indent: 2em;")
	}
	if s.NoGap {
		p = append(p, "margin: 0;")
	}
	if s.Justify {
		p = append(p, "text-align: justify;")
	}
	rule(&sb, "p", p)

	var h1 []string
	if s.CenterTitle {
		h1 = append(h1, "text-align: center;")
	}
	switch {
	case s.Vertical:
		// Right is before the title in vertical-rl.
		h1 = append(h1, "margin: 0 1em 0 2em;")
	case s.CenterTitle:
		h1 = append(h1, "margin: 1em 0 2em;")
	}
	rule(&sb, "h1", h1)

	if extra := strings.TrimSpace(s.Extra); extra != "" {
		sb.WriteString(extra + "\n")
	}
	return sb.String()
}

func rule(sb *strings.Builder, sel string, decls []string) {
	if len(decls) > 0 {
		sb.WriteString(sel + " { " + strings.Join(decls, " ") + " }\n")
	}
}

const baseCSS = `.cover { text-align: center; }
.cover img { max-width: 100%; max-height: 100vh; }
`
