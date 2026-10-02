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
	tgcomp.Title(p.Main, "📘 小說 TXT 轉 EPUB")
	tgcomp.Text(p.Main, "將小說純文字檔切分成章節並製作成 EPUB。")

	file := tgcomp.FileUpload(p.Sidebar, "選擇 txt 檔", ".txt,text/plain", asciiID("novel_txt"))
	if file == nil {
		tgcomp.MessageInfo(p.Main, "👈 請選擇要處理的文字檔。")
		return nil
	}
	raw, err := file.Bytes()
	if err != nil {
		return err
	}
	fileKey := hashBytes(raw)

	encs := memo(p.State, "encodings", fileKey, func() []string { return novel.Encodings(raw) })
	encIdx := tgcomp.Select(p.Sidebar, "編碼", encs, (&tgcomp.SelectConf{
		Base: tgframe.Base{ID: "encoding_" + fileKey},
	}).SetDefault(0))
	if encIdx == nil {
		return nil
	}
	enc := encs[*encIdx]
	removeEmptyLines := tgcomp.Checkbox(p.Sidebar, "移除空行")

	text := memo(p.State, "text", fileKey+enc, func() string { return novel.Decode(raw, enc) })
	lines := memo(p.State, "lines", hashOf(fileKey, enc, removeEmptyLines), func() []string {
		return novel.Lines(text, removeEmptyLines)
	})

	tabChapters, tabMeta := tgcomp.Tab2(p.Main, "章節", "書籍資訊")

	chapters, ok := chaptersTab(p, tabChapters, lines, hashOf(fileKey, enc, removeEmptyLines))
	if !ok {
		return nil
	}

	tgcomp.Text(p.Sidebar, "編碼："+enc)
	tgcomp.Text(p.Sidebar, "行數："+strconv.Itoa(len(lines)))
	tgcomp.Text(p.Sidebar, "字數："+strconv.Itoa(utf8.RuneCountInString(text)))
	tgcomp.Text(p.Sidebar, "章節數："+strconv.Itoa(len(chapters.all)))

	defTitle := strings.TrimSuffix(file.Name, ".txt")
	defIntro := strings.TrimSpace(chapters.head.Content(longChapter))
	meta := metaForm(tabMeta, fileKey, defTitle, defIntro)

	exportEpub(p, meta.Title, func() ([]byte, error) {
		cover, err := meta.cover(p.Context)
		if err != nil {
			return nil, fmt.Errorf("封面：%w", err)
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
}

func chaptersTab(p *tgframe.Params, c *tgframe.Container, lines []string, linesKey string) (chapterSet, bool) {
	allow := nonEmptyLines(tgcomp.Textarea(c, "標題允許清單", &tgcomp.TextareaConf{Default: "第.*章.*"}))
	tgcomp.Caption(c, "符合任一正規表示式（從行首比對）的行即為章節標題。")
	block := nonEmptyLines(tgcomp.Textarea(c, "標題封鎖清單"))
	tgcomp.Caption(c, "包含任一行內容的行不會被視為標題。")

	if slices.ContainsFunc(allow, func(s string) bool { return slices.Contains(block, s) }) {
		tgcomp.MessageWarning(c, "封鎖清單與允許清單有相同的行。")
	}

	m, err := novel.NewMatcher(allow, block)
	if err != nil {
		tgcomp.MessageDanger(c, err.Error())
		return chapterSet{}, false
	}

	splitKey := hashOf(linesKey, strings.Join(allow, "\n"), strings.Join(block, "\n"))
	chs := memo(p.State, "chapters", splitKey, func() []novel.Chapter { return novel.Split(lines, m) })
	set := chapterSet{all: chs, head: chs[0]}

	if tgcomp.Checkbox(c, "移除空章節") {
		set.all = novel.RemoveEmpty(chs)
	}

	rows := make([][]string, len(set.all))
	for i, ch := range set.all {
		rows[i] = []string{strconv.Itoa(i), ch.Title, strconv.Itoa(len(ch.Lines)), snippet(ch.Content(3), 60)}
	}
	sel := tgcomp.DataFrame(c, []string{"編號", "標題", "行數", "內容"}, rows, &tgcomp.DataFrameConf{
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
	tgcomp.Caption(c, "選擇一列以預覽章節。")

	if slices.ContainsFunc(set.all, func(ch novel.Chapter) bool { return len(ch.Lines) > longChapter }) {
		tgcomp.MessageWarning(c, fmt.Sprintf("部分章節超過 %d 行，"+
			"標題正規表示式可能有誤。", longChapter))
	}

	if len(sel) > 0 && sel[0] < len(set.all) {
		ch := set.all[sel[0]]
		exp := tgcomp.Expand(c, "預覽："+ch.Title, true)
		if len(ch.Lines) > previewLines {
			tgcomp.Caption(exp, fmt.Sprintf("顯示前 %d 行，共 %d 行。", previewLines, len(ch.Lines)))
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
