package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/epub_toolbox/internal/novel"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

const (
	// Chapters longer than this hint at a wrong title regex.
	longChapter = 500
	// Lines shown in the chapter preview.
	previewLines = 1000
	// Detected patterns at or above this score replace the default allowlist.
	suggestScore = 0.5
)

func NovelPage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, tr("📘 小說 TXT 轉 EPUB", "📘 Novel TXT to EPUB"))
	tgcomp.Text(p.Main, tr("將小說純文字檔切分成章節並製作成 EPUB。", "Split a plain-text novel into chapters and make an EPUB."))

	file := tgcomp.FileUpload(p.Sidebar, tr("選擇 txt 檔", "Choose a txt file"), ".txt,text/plain")
	if file == nil {
		tgcomp.MessageInfo(p.Main, tr("👈 請選擇要處理的文字檔。", "👈 Choose a text file to process."))
		return nil
	}
	raw, err := file.Bytes()
	if err != nil {
		return err
	}
	fileKey := hashBytes(raw)

	encs := memo(p.State, "encodings", fileKey, func() []string { return novel.Encodings(raw) })
	encIdx := tgcomp.Select(p.Sidebar, tr("編碼", "Encoding"), encs, (&tgcomp.SelectConf{
		Base: tgframe.Base{ID: "encoding_" + fileKey},
	}).SetDefault(0))
	if encIdx == nil {
		return nil
	}
	enc := encs[*encIdx]
	removeEmptyLines := tgcomp.Checkbox(p.Sidebar, tr("移除空行", "Remove empty lines"))

	text := memo(p.State, "text", fileKey+enc, func() string { return novel.Decode(raw, enc) })
	lines := memo(p.State, "lines", hashOf(fileKey, enc, removeEmptyLines), func() []string {
		return novel.Lines(text, removeEmptyLines)
	})

	// Detection reads blank lines, so it ignores removeEmptyLines.
	sugs := memo(p.State, "suggest", fileKey+enc, func() []novel.Suggestion {
		return novel.Suggest(novel.Lines(text, false))
	})

	tabChapters, tabMeta, tabStyle := tgcomp.Tab3(p.Main, tr("章節", "Chapters"), tr("書籍資訊", "Book Info"), tr("樣式", "Style"))

	chapters, ok := chaptersTab(p, tabChapters, lines, hashOf(fileKey, enc, removeEmptyLines), fileKey+enc, sugs)
	if !ok {
		return nil
	}

	tgcomp.Text(p.Sidebar, tr("編碼：", "Encoding: ")+enc)
	tgcomp.Text(p.Sidebar, tr("行數：", "Lines: ")+strconv.Itoa(len(lines)))
	tgcomp.Text(p.Sidebar, tr("字數：", "Characters: ")+strconv.Itoa(utf8.RuneCountInString(text)))
	tgcomp.Text(p.Sidebar, tr("章節數：", "Chapters: ")+strconv.Itoa(len(chapters.all)))

	defTitle, _ := cutExt(file.Name, ".txt")
	defIntro := strings.TrimSpace(chapters.head.Content(longChapter))
	meta := metaForm(tabMeta, fileKey, defTitle, defIntro)
	style := styleForm(tabStyle)

	exportEpub(p, "novel", meta.Title, func() ([]byte, error) {
		cover, err := meta.cover()
		if err != nil {
			return nil, fmt.Errorf(tr("封面：%w", "cover: %w"), err)
		}
		return novel.Build(novel.Meta{
			Title: meta.Title, Author: meta.Author, Intro: meta.Intro,
			Language: meta.Language, Cover: cover, Style: style,
			KeepHead: chapters.keepHead,
		}, chapters.all)
	})
	return nil
}

func styleForm(c *tgframe.Container) novel.Style {
	names := make([]string, len(novel.Templates))
	for i, t := range novel.Templates {
		names[i] = t.Name()
	}
	def := novel.TemplateHorizontal
	if i18n.EN() {
		def = novel.TemplateReader
	}
	idx := tgcomp.Select(c, tr("排版模板", "Layout template"), names, (&tgcomp.SelectConf{}).SetDefault(slices.Index(novel.Templates, def)))
	if idx == nil {
		return novel.Style{}
	}
	t := novel.Templates[*idx]
	desc := tgcomp.Column1(c) // filled once the options are known

	// Scoped by template so switching it resets the options.
	d := t.Style()
	sc := c.Scope("template_" + strconv.Itoa(*idx))
	box := func(label string, v bool) bool {
		return tgcomp.Checkbox(sc, label, &tgcomp.CheckboxConf{Default: v})
	}
	s := novel.Style{
		Indent:      box(tr("首行縮排兩字", "Two-character first-line indent"), d.Indent),
		NoGap:       box(tr("取消段距", "No paragraph gap"), d.NoGap),
		Justify:     box(tr("左右對齊", "Justify"), d.Justify),
		CenterTitle: box(tr("章節標題置中", "Center chapter titles"), d.CenterTitle),
		Vertical:    box(tr("直排（向左翻頁）", "Vertical (pages turn left)"), d.Vertical),
	}
	tgcomp.Caption(c, tr("字體、字級與顏色交給閱讀器設定。", "Font, size and colors are left to the reader app."))

	adv := tgcomp.Expand(c, tr("進階：自訂 CSS", "Advanced: custom CSS"), false)
	s.Extra = tgcomp.Textarea(adv, tr("自訂 CSS", "Custom CSS"), &tgcomp.TextareaConf{
		Height: 6,
	})
	tgcomp.Caption(adv, tr("例如 p { line-height: 1.8; }。加在產生的 CSS 之後，可覆寫上面的設定。寫死字級或顏色可能讓閱讀器設定或夜間模式失效。",
		"E.g. p { line-height: 1.8; }. Appended after the generated CSS, so it overrides the options above. Fixed sizes or colors may break reader settings or night mode."))

	s.Extra = strings.TrimSpace(s.Extra)
	if s == d {
		tgcomp.Caption(desc, t.Desc())
	} else {
		tgcomp.Caption(desc, fmt.Sprintf(tr("已自訂（以「%s」為基礎）。", "Customized (based on %s)."), t.Name()))
	}

	exp := tgcomp.Expand(c, "CSS", false)
	tgcomp.Code(exp, s.CSS(), &tgcomp.CodeConf{Language: "css"})
	return s
}

type chapterSet struct {
	all  []novel.Chapter
	head novel.Chapter // before RemoveEmpty, for the default intro

	keepHead bool
}

func chaptersTab(p *tgframe.Params, c *tgframe.Container, lines []string, linesKey, textKey string, sugs []novel.Suggestion) (chapterSet, bool) {
	defAllow := tr("第.*章.*", "Chapter.*")
	detected := len(sugs) > 0 && sugs[0].Score >= suggestScore
	if detected {
		defAllow = sugs[0].Pattern
	}
	allow := nonEmptyLines(tgcomp.Textarea(c, tr("標題允許清單", "Title allowlist"),
		&tgcomp.TextareaConf{Default: defAllow, ResetKey: textKey}))
	tgcomp.Caption(c, tr("符合任一正規表示式（從行首比對）的行即為章節標題。",
		"Lines matching any regex (from the line start) are chapter titles."))
	suggestions(c, sugs, detected)
	block := nonEmptyLines(tgcomp.Textarea(c, tr("標題封鎖清單", "Title blocklist")))
	tgcomp.Caption(c, tr("包含任一行內容的行不會被視為標題。", "Lines containing any of these lines are never titles."))

	if slices.ContainsFunc(allow, func(s string) bool { return slices.Contains(block, s) }) {
		tgcomp.MessageWarning(c, tr("封鎖清單與允許清單有相同的行。", "The blocklist and allowlist share a line."))
	}

	m, err := novel.NewMatcher(allow, block)
	if err != nil {
		showError(c, toolNovel, err)
		return chapterSet{}, false
	}

	splitKey := hashOf(linesKey, strings.Join(allow, "\n"), strings.Join(block, "\n"))
	chs := memo(p.State, "chapters", splitKey, func() []novel.Chapter { return novel.Split(lines, m) })
	set := chapterSet{all: chs, head: chs[0]}

	if tgcomp.Checkbox(c, tr("移除空章節", "Remove empty chapters")) {
		set.all = novel.RemoveEmpty(chs)
	}

	// The default intro holds only longChapter lines of the head, so keep a longer head by default.
	longHead := len(set.head.Lines) > longChapter
	set.keepHead = tgcomp.Checkbox(c, tr("將開頭放入書中", "Include the opening in the book"), &tgcomp.CheckboxConf{
		Base: tgframe.Base{ID: "keep_head_" + splitKey}, Default: longHead,
	})
	if !set.keepHead {
		tgcomp.Caption(c, tr("開頭只會作為預設簡介。", "The opening is used only as the default introduction."))
		if longHead {
			tgcomp.MessageWarning(c, fmt.Sprintf(tr("開頭有 %d 行，預設簡介只取前 %d 行，其餘不會出現在書中。",
				"The opening has %d lines; the default introduction takes only the first %d, the rest is left out of the book."),
				len(set.head.Lines), longChapter))
		}
	}

	rows := make([][]string, len(set.all))
	for i, ch := range set.all {
		rows[i] = []string{strconv.Itoa(i), ch.Title, strconv.Itoa(len(ch.Lines)), snippet(ch.Content(3), 60)}
	}
	sel := tgcomp.DataFrame(c, []string{tr("編號", "No."), tr("標題", "Title"), tr("行數", "Lines"), tr("內容", "Content")}, rows, &tgcomp.DataFrameConf{
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
	tgcomp.Caption(c, tr("選擇一列以預覽章節。", "Select a row to preview the chapter."))

	if slices.ContainsFunc(set.all, func(ch novel.Chapter) bool { return len(ch.Lines) > longChapter }) {
		tgcomp.MessageWarning(c, fmt.Sprintf(tr("部分章節超過 %d 行，標題正規表示式可能有誤。",
			"Some chapters exceed %d lines; the title regex may be wrong."), longChapter))
	}

	if len(sel) > 0 && sel[0] < len(set.all) {
		ch := set.all[sel[0]]
		exp := tgcomp.Expand(c, tr("預覽：", "Preview: ")+ch.Title, true)
		if len(ch.Lines) > previewLines {
			tgcomp.Caption(exp, fmt.Sprintf(tr("顯示前 %d 行，共 %d 行。", "Showing the first %d of %d lines."), previewLines, len(ch.Lines)))
		}
		tgcomp.Code(exp, ch.Content(previewLines), &tgcomp.CodeConf{
			Base: tgframe.Base{ID: "chapter_preview"}, Language: "text", // wrapped by web/assets/style.css
		})
	}
	return set, true
}

// suggestions reports the detected title patterns.
func suggestions(c *tgframe.Container, sugs []novel.Suggestion, detected bool) {
	if detected {
		tgcomp.Caption(c, fmt.Sprintf(tr("🧪 已依本書內容自動偵測標題格式（信心 %.0f%%，實驗性功能，請確認章節切分）。",
			"🧪 Title format detected from this book (confidence %.0f%%; experimental, check the chapters)."), sugs[0].Score*100))
	} else {
		tgcomp.Caption(c, tr("無法確定本書的標題格式，使用預設值。", "Could not tell this book's title format; using the default."))
	}
	sugs = slices.DeleteFunc(slices.Clone(sugs), func(s novel.Suggestion) bool { return s.Score < suggestScore })
	if len(sugs) == 0 {
		return
	}
	exp := tgcomp.Expand(c, tr("偵測到的標題格式（實驗性）", "Detected title formats (Experimental)"), false)
	rows := make([][]string, len(sugs))
	for i, s := range sugs {
		rows[i] = []string{fmt.Sprintf("%.0f%%", s.Score*100), strconv.Itoa(s.Count), s.Pattern, strings.Join(s.Samples, " / ")}
	}
	tgcomp.DataFrame(exp, []string{tr("信心", "Confidence"), tr("行數", "Lines"), tr("正規表示式", "Regex"), tr("範例", "Samples")}, rows, &tgcomp.DataFrameConf{
		Base: tgframe.Base{ID: "suggestions"},
	})
	tgcomp.Caption(exp, tr("信心由手調規則估算：標題是否短、無句末標點、編號是否連續、章節長度是否合理。可複製到允許清單使用。",
		"Confidence comes from hand-tuned rules: short lines, no closing punctuation, consecutive numbers, plausible chapter lengths. Copy a regex into the allowlist to use it."))
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
