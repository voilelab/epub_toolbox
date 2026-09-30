package novel

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/voilelab/epub_toolbox/internal/epub"
)

// Chapter is a title line and the lines below it.
type Chapter struct {
	Title string
	Lines []string
	Head  bool // content before the first title
}

// HeadTitle is the title of the head chapter.
const HeadTitle = "Head (Content before first title)"

// Content joins the first limit lines, or all lines if limit <= 0.
func (c Chapter) Content(limit int) string {
	lines := c.Lines
	if limit > 0 && len(lines) > limit {
		lines = lines[:limit]
	}
	return strings.Join(lines, "\n")
}

// Lines splits text into lines, trimming CR and optionally dropping blank ones.
func Lines(text string, removeEmpty bool) []string {
	raw := strings.Split(text, "\n")
	out := raw[:0]
	for _, l := range raw {
		l = strings.TrimRight(l, "\r")
		if removeEmpty && strings.TrimSpace(l) == "" {
			continue
		}
		out = append(out, l)
	}
	return out
}

// Matcher decides which lines are chapter titles.
type Matcher struct {
	allow []*regexp.Regexp
	block []string
}

// NewMatcher builds a Matcher. A line is a title if it matches one of allow
// from its start and contains none of block.
func NewMatcher(allow, block []string) (*Matcher, error) {
	m := &Matcher{block: block}
	for _, a := range allow {
		re, err := regexp.Compile("^(?:" + a + ")")
		if err != nil {
			return nil, fmt.Errorf("invalid regex %q: %w", a, err)
		}
		m.allow = append(m.allow, re)
	}
	return m, nil
}

// IsTitle reports whether line is a chapter title.
func (m *Matcher) IsTitle(line string) bool {
	for _, b := range m.block {
		if strings.Contains(line, b) {
			return false
		}
	}
	for _, re := range m.allow {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// Split splits lines into chapters. The first chapter is always the head.
func Split(lines []string, m *Matcher) []Chapter {
	chs := []Chapter{{Title: HeadTitle, Head: true}}
	for _, l := range lines {
		if m.IsTitle(l) {
			chs = append(chs, Chapter{Title: l})
			continue
		}
		last := &chs[len(chs)-1]
		last.Lines = append(last.Lines, l)
	}
	return chs
}

// RemoveEmpty drops chapters without lines.
func RemoveEmpty(chs []Chapter) []Chapter {
	var out []Chapter
	for _, c := range chs {
		if len(c.Lines) > 0 {
			out = append(out, c)
		}
	}
	return out
}

// Meta is the book metadata.
type Meta struct {
	Title, Author, Intro, Language string
	Cover                          []byte
}

// Build builds an EPUB from the non-head chapters.
func Build(meta Meta, chs []Chapter) ([]byte, error) {
	b := &epub.Book{
		Title:       meta.Title,
		Author:      meta.Author,
		Description: meta.Intro,
		Language:    meta.Language,
		CSS:         css,
	}
	if len(meta.Cover) > 0 {
		if err := b.SetCover(meta.Cover); err != nil {
			return nil, err
		}
	}
	if meta.Intro != "" {
		b.AddSection("Introduction", epub.Paragraphs(strings.Split(meta.Intro, "\n")))
	}
	for _, c := range chs {
		if c.Head {
			continue
		}
		b.AddSection(c.Title, "<h1>"+epub.Escape(c.Title)+"</h1>\n"+epub.Paragraphs(c.Lines))
	}
	return b.Bytes()
}

const css = `.cover { text-align: center; }
.cover img { max-width: 100%; max-height: 100vh; }
`
