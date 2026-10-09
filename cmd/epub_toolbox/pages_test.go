package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgframe"
	"github.com/voilelab/toolgui/toolgui/tgtest"
)

// openWith opens page and uploads name to the file input labelled label.
func openWith(t *testing.T, page, label, name string, data []byte) *tgtest.Page {
	t.Helper()
	p := tgtest.Open(t, newApp(), page)
	p.GetByLabel(label).Upload(name, data)
	return p
}

func openNovel(t *testing.T, name, text string) *tgtest.Page {
	t.Helper()
	return openWith(t, "novel", "選擇 txt 檔", name, []byte(text))
}

// hasAll reports each of want the page doesn't show.
func hasAll(t *testing.T, p *tgtest.Page, want ...string) {
	t.Helper()
	for _, w := range want {
		if !p.HasText(w) {
			t.Errorf("missing %q", w)
		}
	}
}

// download clicks the download button and returns the built file.
func download(t *testing.T, p *tgtest.Page) []byte {
	t.Helper()
	btns := p.FindByName("download_file_component")
	if len(btns) != 1 {
		t.Fatalf("%d download buttons", len(btns))
	}
	btns[0].Click()
	if err := p.Err(); err != nil {
		t.Fatal(err)
	}
	d := p.State().GetDownload(p.FindByName("download_file_component")[0].String("token"))
	if d == nil {
		t.Fatal("no download")
	}
	f, err := d.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	bs, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	return bs
}

// epubText joins every file in the EPUB.
func epubText(t *testing.T, data []byte) string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		bs, _ := io.ReadAll(rc)
		rc.Close()
		sb.WriteString(f.Name + "\n" + string(bs) + "\n")
	}
	return sb.String()
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const novelText = "前言\n\n第一章 開始\n從前有座山。\n\n第二章 結束\n山上有座廟。\n"

func TestNovelPage(t *testing.T) {
	p := openNovel(t, "我的書.TXT", novelText)
	hasAll(t, p, "章節數：3", "行數：8", "第一章 開始", "下載 我的書.epub")

	text := epubText(t, download(t, p))
	for _, want := range []string{"<dc:title>我的書</dc:title>", "第一章 開始", "從前有座山。", "山上有座廟。"} {
		if !strings.Contains(text, want) {
			t.Errorf("epub missing %q", want)
		}
	}
}

func TestNovelPageOptions(t *testing.T) {
	p := openNovel(t, "a.txt", novelText)
	p.GetByLabel("移除空行").Input(true)
	p.GetByLabel("移除空章節").Input(true)
	p.GetByLabel("左右對齊").Input(true)
	p.GetByLabel("封面").Upload("c.png", pngBytes(t))

	hasAll(t, p, "行數：5", "已自訂（以「中文橫排」為基礎）。")
	text := epubText(t, download(t, p))
	if !strings.Contains(text, "text-align: justify;") || !strings.Contains(text, "EPUB/cover.xhtml") {
		t.Error("epub misses style or cover")
	}

	// A new template resets the options.
	p.GetByLabel("排版模板").Select(0)
	if p.HasText("已自訂") {
		t.Error("options kept after switching template")
	}
}

func TestNovelPageRegexLink(t *testing.T) {
	p := tgtest.Open(t, newApp(), "novel")
	if l := p.FindByName("page_link_component"); len(l) != 1 || l[0].String("page") != "regex" {
		t.Error("no regex guide link before a file is chosen")
	}
}

func TestNovelPageCustomCSS(t *testing.T) {
	p := openNovel(t, "a.txt", novelText)
	p.GetByLabel("自訂 CSS").Input("  p { line-height: 1.8; }\n")
	if !strings.Contains(epubText(t, download(t, p)), "p { line-height: 1.8; }") {
		t.Error("custom CSS not in epub")
	}
}

func TestHomePageLinks(t *testing.T) {
	p := tgtest.Open(t, newApp(), "index")
	var pages []string
	for _, n := range p.FindByName("page_link_component") {
		pages = append(pages, n.String("page"))
	}
	if got := strings.Join(pages, ","); got != "novel,images" {
		t.Errorf("links = %s", got)
	}
}

func TestNovelPageDetect(t *testing.T) {
	var sb strings.Builder
	for _, n := range []string{"一", "二", "三", "四", "五"} {
		sb.WriteString("【" + n + "】\n\n內文。\n內文。\n內文。\n\n")
	}
	p := openNovel(t, "a.txt", sb.String())
	hasAll(t, p, "章節數：6", "已依本書內容自動偵測標題格式", "【五】")

	// Confidence sorts as a number but shows as a percentage.
	sugs := p.Get("dataframe_component_suggestions").Prop("rows").([]any)
	c := sugs[0].([]any)[0].(map[string]any)
	if v, _ := c["value"].(float64); c["display"] != fmt.Sprintf("%.0f%%", v*100) {
		t.Errorf("confidence cell = %v", c)
	}

	// A new file resets the allowlist to its own default.
	p.GetByLabel("選擇 txt 檔").Upload("b.txt", []byte(novelText))
	hasAll(t, p, "章節數：3", "無法確定本書的標題格式")
}

func TestNovelPageInvalidRegex(t *testing.T) {
	p := openNovel(t, "a.txt", novelText)
	p.GetByLabel("標題允許清單").Input("(")

	if len(p.FindByName("message_component")) == 0 {
		t.Error("no error message")
	}
	if len(p.FindByName("download_file_component")) != 0 {
		t.Error("download shown for a bad regex")
	}
}

func TestNovelPageOverlap(t *testing.T) {
	p := openNovel(t, "a.txt", novelText)
	p.GetByLabel("標題封鎖清單").Input("第.*章.*\r\n\n")
	hasAll(t, p, "封鎖清單與允許清單有相同的行。")
}

func TestNovelPageLong(t *testing.T) {
	// The head is previewed by default.
	head := strings.Repeat("序\n", previewLines+1)
	p := openNovel(t, "a.txt", head+"第一章\n內文\n")
	hasAll(t, p, "部分章節超過", "顯示前 1000 行")

	// A long head is kept by default; dropping it warns.
	if !strings.Contains(epubText(t, download(t, p)), "序") {
		t.Error("long head not kept")
	}
	p.GetByLabel("將開頭放入書中").Input(false)
	hasAll(t, p, "其餘不會出現在書中")
}

func TestNovelPageEmptyBook(t *testing.T) {
	p := openNovel(t, "a.txt", "沒有標題\n")
	p.GetByLabel("簡介").Input("")

	p.FindByName("download_file_component")[0].Click()
	if err := p.Err(); err == nil || !strings.Contains(err.Error(), "書中沒有任何內容") {
		t.Errorf("err = %v", err)
	}
}

func TestNovelPageEN(t *testing.T) {
	i18n.Set("en")
	defer i18n.Set("zh-TW")
	p := openWith(t, "novel", "Choose a txt file", "a.txt", []byte("Intro\nChapter 1\nHello\n"))
	hasAll(t, p, "Chapters: 2", "No body styling; the reader app decides.")
}

func TestImagesPage(t *testing.T) {
	img := pngBytes(t)
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	for _, name := range []string{"10.png", "2.png"} {
		w, _ := zw.Create(name)
		w.Write(img)
	}
	zw.Close()

	p := openWith(t, "images", "選擇圖片或 zip 檔", "漫畫.zip", zbuf.Bytes())
	p.GetByLabel("以第一張圖片作為封面").Input(true)
	hasAll(t, p, "圖片數：2", "1. 2.png", "下載 漫畫.epub")

	text := epubText(t, download(t, p))
	if !strings.Contains(text, "<dc:title>漫畫</dc:title>") || !strings.Contains(text, "EPUB/cover.xhtml") {
		t.Error("epub misses title or cover")
	}
}

func TestImagesPageFiles(t *testing.T) {
	img := pngBytes(t)
	p := tgtest.Open(t, newApp(), "images")
	p.GetByLabel("選擇圖片或 zip 檔").UploadFiles(tgtest.File{Name: "b.png", Body: img}, tgtest.File{Name: "a.png", Body: img})
	p.GetByLabel("封面").Upload("c.png", img)

	hasAll(t, p, "圖片數：2", "下載 book.epub")
	if p.HasText("以第一張圖片作為封面") {
		t.Error("first-as-cover offered with a cover")
	}
	download(t, p)
}

func TestImagesPageError(t *testing.T) {
	p := openWith(t, "images", "選擇圖片或 zip 檔", "a.zip", []byte("not a zip"))
	if len(p.FindByName("message_component")) == 0 || len(p.FindByName("download_file_component")) != 0 {
		t.Error("bad zip not reported")
	}
}

func TestNonEmptyLines(t *testing.T) {
	got := nonEmptyLines("a\r\n\nb\r\n\r\n")
	if strings.Join(got, ",") != "a,b" {
		t.Errorf("nonEmptyLines = %q", got)
	}
	if nonEmptyLines("") != nil {
		t.Error("want nil")
	}
}

func TestSnippet(t *testing.T) {
	for _, tt := range []struct {
		in   string
		n    int
		want string
	}{
		{"a\nb", 5, "a b"},
		{"一二三四", 2, "一二…"},
		{"abc", 3, "abc"},
	} {
		if got := snippet(tt.in, tt.n); got != tt.want {
			t.Errorf("snippet(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}

func TestHash(t *testing.T) {
	if hashOf("a", 1) == hashOf("a1") {
		t.Error("hashOf ignores part boundaries")
	}
	if len(hashBytes(nil)) != 16 || hashBytes([]byte("a")) == hashBytes([]byte("b")) {
		t.Error("bad hashBytes")
	}
}

func TestMemo(t *testing.T) {
	s := tgframe.NewState()
	t.Cleanup(s.Destroy)
	calls := 0
	f := func() int { calls++; return calls }
	if memo(s, "x", "k1", f) != 1 || memo(s, "x", "k1", f) != 1 {
		t.Error("not cached")
	}
	if memo(s, "x", "k2", f) != 2 {
		t.Error("not recomputed on a new key")
	}
}

func TestBookMetaNoCover(t *testing.T) {
	m := bookMeta{}
	if m.hasCover() {
		t.Error("hasCover")
	}
	if b, err := m.cover(); b != nil || err != nil {
		t.Errorf("cover = %v, %v", b, err)
	}
}
