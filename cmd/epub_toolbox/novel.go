package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/voilelab/epub_toolbox/internal/novel"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

const (
	// Chapters longer than this hint at a wrong title regex.
	longChapter = 500
	// Lines shown in the chapter preview.
	previewLines = 1000
)

func NovelPage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "📘 Novel TXT-EPUB Builder")
	tgcomp.Text(p.Main, "Split a novel's plain text file into chapters and build an EPUB.")

	file := tgcomp.FileUpload(p.Sidebar, "Choose a txt file", ".txt,text/plain")
	if file == nil {
		tgcomp.MessageInfo(p.Main, "👈 Please select a text file to process.")
		return nil
	}
	raw, err := file.Bytes()
	if err != nil {
		return err
	}
	fileKey := hashBytes(raw)

	encs := memo(p.State, "encodings", fileKey, func() []string { return novel.Encodings(raw) })
	encIdx := tgcomp.Select(p.Sidebar, "Encoding", encs, (&tgcomp.SelectConf{
		Base: tgframe.Base{ID: "encoding_" + fileKey},
	}).SetDefault(0))
	if encIdx == nil {
		return nil
	}
	enc := encs[*encIdx]
	removeEmptyLines := tgcomp.Checkbox(p.Sidebar, "Remove empty lines")

	text := memo(p.State, "text", fileKey+enc, func() string { return novel.Decode(raw, enc) })
	lines := memo(p.State, "lines", hashOf(fileKey, enc, removeEmptyLines), func() []string {
		return novel.Lines(text, removeEmptyLines)
	})

	tabChapters, tabMeta := tgcomp.Tab2(p.Main, "Chapters", "Meta Data")

	chapters, ok := chaptersTab(p, tabChapters, lines, hashOf(fileKey, enc, removeEmptyLines))
	if !ok {
		return nil
	}

	tgcomp.Text(p.Sidebar, "Encoding: "+enc)
	tgcomp.Text(p.Sidebar, "Lines: "+strconv.Itoa(len(lines)))
	tgcomp.Text(p.Sidebar, "Char count: "+strconv.Itoa(utf8.RuneCountInString(text)))
	tgcomp.Text(p.Sidebar, "Chapter count: "+strconv.Itoa(len(chapters.all)))

	defTitle := strings.TrimSuffix(file.Name, ".txt")
	defIntro := strings.TrimSpace(chapters.head.Content(longChapter))
	meta := metaForm(tabMeta, fileKey, defTitle, defIntro)

	exportEpub(p, hashOf(chapters.key, meta.key()), meta.Title, func() ([]byte, error) {
		cover, err := meta.cover(p.Context)
		if err != nil {
			return nil, fmt.Errorf("book cover: %w", err)
		}
		return novel.Build(novel.Meta{
			Title: meta.Title, Author: meta.Author, Intro: meta.Intro,
			Language: meta.Language, Cover: cover,
		}, chapters.all)
	})
	return nil
}

type chapterSet struct {
	all  []novel.Chapter
	head novel.Chapter // before RemoveEmpty, for the default intro
	key  string
}

func chaptersTab(p *tgframe.Params, c *tgframe.Container, lines []string, linesKey string) (chapterSet, bool) {
	allow := nonEmptyLines(tgcomp.Textarea(c, "Title allow list", &tgcomp.TextareaConf{Default: "第.*章.*"}))
	tgcomp.Caption(c, "Any line matching one of these regular expressions (from the line start) is a chapter title.")
	block := nonEmptyLines(tgcomp.Textarea(c, "Title block list"))
	tgcomp.Caption(c, "Any line containing one of these lines won't be a title.")

	if slices.ContainsFunc(allow, func(s string) bool { return slices.Contains(block, s) }) {
		tgcomp.MessageWarning(c, "Block list and allow list have a common line.")
	}

	m, err := novel.NewMatcher(allow, block)
	if err != nil {
		tgcomp.MessageDanger(c, err.Error())
		return chapterSet{}, false
	}

	splitKey := hashOf(linesKey, strings.Join(allow, "\n"), strings.Join(block, "\n"))
	chs := memo(p.State, "chapters", splitKey, func() []novel.Chapter { return novel.Split(lines, m) })
	set := chapterSet{all: chs, head: chs[0], key: splitKey}

	if tgcomp.Checkbox(c, "Remove empty chapters") {
		set.all = novel.RemoveEmpty(chs)
		set.key += "-nonempty"
	}

	rows := make([][]string, len(set.all))
	for i, ch := range set.all {
		rows[i] = []string{strconv.Itoa(i), ch.Title, strconv.Itoa(len(ch.Lines)), snippet(ch.Content(3), 60)}
	}
	sel := tgcomp.DataFrame(c, []string{"Index", "Title", "Lines", "Content"}, rows, &tgcomp.DataFrameConf{
		Base:             tgframe.Base{ID: "chapters"},
		PageSize:         10,
		Selection:        tgcomp.SelectionModeSingle,
		DefaultSelection: []int{0},
		ColumnConf: []tgcomp.DataFrameColumnConf{
			{Type: tgcomp.ColumnTypeNumber, Width: "5rem"},
			{},
			{Type: tgcomp.ColumnTypeNumber, Width: "5rem"},
			{},
		},
	})
	tgcomp.Caption(c, "Select a row to preview the chapter.")

	if slices.ContainsFunc(set.all, func(ch novel.Chapter) bool { return len(ch.Lines) > longChapter }) {
		tgcomp.MessageWarning(c, fmt.Sprintf("Some chapters have more than %d lines; "+
			"the title regex may be wrong.", longChapter))
	}

	if len(sel) > 0 && sel[0] < len(set.all) {
		ch := set.all[sel[0]]
		exp := tgcomp.Expand(c, "Preview: "+ch.Title, true)
		if len(ch.Lines) > previewLines {
			tgcomp.Caption(exp, fmt.Sprintf("First %d of %d lines.", previewLines, len(ch.Lines)))
		}
		tgcomp.Code(exp, ch.Content(previewLines), &tgcomp.CodeConf{Language: "text"})
	}
	return set, true
}

func nonEmptyLines(s string) []string {
	var out []string
	for l := range strings.SplitSeq(s, "\n") {
		if l = strings.TrimRight(l, "\r"); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func snippet(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
