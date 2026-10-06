package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"strings"
	"testing"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

const (
	novelFileID  = "fileupload_component_選擇 txt 檔"
	imagesFileID = "fileupload_component_選擇圖片或 zip 檔"
	allowID      = "textarea_component_標題允許清單"
	blockID      = "textarea_component_標題封鎖清單"
	coverFileID  = "fileupload_component_封面"
)

// pageRun is the components one run drew, as JSON.
type pageRun []map[string]any

func runPage(t *testing.T, page string, s *tgframe.State) pageRun {
	t.Helper()
	var out pageRun
	err := newApp().Run(page, s, func(p tgframe.NotifyPack) {
		bs, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		var m struct {
			Component map[string]any `json:"component"`
		}
		if err := json.Unmarshal(bs, &m); err != nil {
			t.Fatal(err)
		}
		if m.Component != nil {
			out = append(out, m.Component)
		}
	})
	if err != nil {
		t.Fatalf("run %s: %v", page, err)
	}
	return out
}

// has reports whether any component carries s in its JSON.
func (r pageRun) has(s string) bool {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
	return strings.Contains(sb.String(), s)
}

func (r pageRun) find(name string) map[string]any {
	for _, c := range r {
		if c["name"] == name {
			return c
		}
	}
	return nil
}

func upload(t *testing.T, s *tgframe.State, id, name string, data []byte) {
	t.Helper()
	s.Set(id, map[string]any{"name": name})
	if _, err := s.WriteFile(id, name, bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
}

func uploadMany(t *testing.T, s *tgframe.State, id string, files map[string][]byte, order []string) {
	t.Helper()
	objs := make([]map[string]any, len(order))
	for i, name := range order {
		objs[i] = map[string]any{"name": name}
		if _, err := s.WriteFile(tgframe.FileKey(id, i), name, bytes.NewReader(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	s.Set(id, objs)
}

// download clicks the download button and returns the built file.
func download(t *testing.T, page string, s *tgframe.State) []byte {
	t.Helper()
	btn := runPage(t, page, s).find("download_file_component")
	if btn == nil {
		t.Fatal("no download button")
	}
	s.SetClickID(btn["id"].(string))
	btn = runPage(t, page, s).find("download_file_component")
	s.SetClickID("")
	d := s.GetDownload(btn["token"].(string))
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

func newState(t *testing.T) *tgframe.State {
	s := tgframe.NewState()
	t.Cleanup(s.Destroy)
	return s
}

const novelText = "前言\n\n第一章 開始\n從前有座山。\n\n第二章 結束\n山上有座廟。\n"

func TestNovelPage(t *testing.T) {
	s := newState(t)
	upload(t, s, novelFileID, "我的書.TXT", []byte(novelText))

	r := runPage(t, "novel", s)
	for _, want := range []string{"章節數：3", "行數：8", "第一章 開始", "下載 我的書.epub"} {
		if !r.has(want) {
			t.Errorf("missing %q", want)
		}
	}

	text := epubText(t, download(t, "novel", s))
	for _, want := range []string{"<dc:title>我的書</dc:title>", "第一章 開始", "從前有座山。", "山上有座廟。"} {
		if !strings.Contains(text, want) {
			t.Errorf("epub missing %q", want)
		}
	}
}

func TestNovelPageOptions(t *testing.T) {
	s := newState(t)
	upload(t, s, novelFileID, "a.txt", []byte(novelText))
	s.Set("checkbox_component_移除空行", true)
	s.Set("checkbox_component_移除空章節", true)
	s.Set("checkbox_component_style_justify_1", true)
	upload(t, s, coverFileID, "c.png", pngBytes(t))

	r := runPage(t, "novel", s)
	if !r.has("行數：5") {
		t.Error("empty lines not removed")
	}
	if !r.has("已自訂（以「中文橫排」為基礎）。") {
		t.Error("customized style not shown")
	}
	text := epubText(t, download(t, "novel", s))
	if !strings.Contains(text, "text-align: justify;") || !strings.Contains(text, "EPUB/cover.xhtml") {
		t.Error("epub misses style or cover")
	}
}

func TestNovelPageDetect(t *testing.T) {
	var sb strings.Builder
	for _, n := range []string{"一", "二", "三", "四", "五"} {
		sb.WriteString("【" + n + "】\n\n內文。\n內文。\n內文。\n\n")
	}
	s := newState(t)
	upload(t, s, novelFileID, "a.txt", []byte(sb.String()))

	r := runPage(t, "novel", s)
	for _, want := range []string{"章節數：6", "已依本書內容自動偵測標題格式", "【五】"} {
		if !r.has(want) {
			t.Errorf("missing %q", want)
		}
	}

	// A new file resets the allowlist to its own default.
	upload(t, s, novelFileID, "b.txt", []byte(novelText))
	r = runPage(t, "novel", s)
	if !r.has("章節數：3") || !r.has("無法確定本書的標題格式") {
		t.Error("allowlist not reset for a new file")
	}
}

func TestNovelPageInvalidRegex(t *testing.T) {
	s := newState(t)
	upload(t, s, novelFileID, "a.txt", []byte(novelText))
	s.Set(allowID, "(")

	r := runPage(t, "novel", s)
	if r.find("message_component") == nil {
		t.Error("no error message")
	}
	if r.find("download_file_component") != nil {
		t.Error("download shown for a bad regex")
	}
}

func TestNovelPageOverlap(t *testing.T) {
	s := newState(t)
	upload(t, s, novelFileID, "a.txt", []byte(novelText))
	s.Set(blockID, "第.*章.*\r\n\n")

	if !runPage(t, "novel", s).has("封鎖清單與允許清單有相同的行。") {
		t.Error("no overlap warning")
	}
}

func TestNovelPageLong(t *testing.T) {
	// The head is previewed by default.
	head := strings.Repeat("序\n", previewLines+1)
	s := newState(t)
	upload(t, s, novelFileID, "a.txt", []byte(head+"第一章\n內文\n"))

	r := runPage(t, "novel", s)
	if !r.has("部分章節超過") {
		t.Error("no long chapter warning")
	}
	if !r.has("顯示前 1000 行") {
		t.Error("preview not truncated")
	}
	// A long head is kept by default; dropping it warns.
	if !strings.Contains(epubText(t, download(t, "novel", s)), "序") {
		t.Error("long head not kept")
	}
	for _, c := range r {
		if id, _ := c["id"].(string); strings.HasPrefix(id, "checkbox_component_keep_head_") {
			s.Set(id, false)
		}
	}
	if !runPage(t, "novel", s).has("其餘不會出現在書中") {
		t.Error("no dropped head warning")
	}
}

func TestNovelPageEmptyBook(t *testing.T) {
	s := newState(t)
	upload(t, s, novelFileID, "a.txt", []byte("沒有標題\n"))
	s.Set("textarea_component_簡介", "")

	btn := runPage(t, "novel", s).find("download_file_component")
	s.SetClickID(btn["id"].(string))
	err := newApp().Run("novel", s, func(tgframe.NotifyPack) {})
	if err == nil || !strings.Contains(err.Error(), "書中沒有任何內容") {
		t.Errorf("err = %v", err)
	}
}

func TestNovelPageEN(t *testing.T) {
	i18n.Set("en")
	defer i18n.Set("zh-TW")
	s := newState(t)
	upload(t, s, "fileupload_component_Choose a txt file", "a.txt", []byte("Intro\nChapter 1\nHello\n"))

	r := runPage(t, "novel", s)
	if !r.has("Chapters: 2") || !r.has("No body styling; the reader app decides.") {
		t.Error("English page not drawn")
	}
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

	s := newState(t)
	uploadMany(t, s, imagesFileID, map[string][]byte{"漫畫.zip": zbuf.Bytes()}, []string{"漫畫.zip"})
	s.Set("checkbox_component_以第一張圖片作為封面", true)

	r := runPage(t, "images", s)
	for _, want := range []string{"圖片數：2", "1. 2.png", "下載 漫畫.epub"} {
		if !r.has(want) {
			t.Errorf("missing %q", want)
		}
	}
	text := epubText(t, download(t, "images", s))
	if !strings.Contains(text, "<dc:title>漫畫</dc:title>") || !strings.Contains(text, "EPUB/cover.xhtml") {
		t.Error("epub misses title or cover")
	}
}

func TestImagesPageFiles(t *testing.T) {
	img := pngBytes(t)
	s := newState(t)
	uploadMany(t, s, imagesFileID, map[string][]byte{"a.png": img, "b.png": img}, []string{"b.png", "a.png"})
	upload(t, s, coverFileID, "c.png", img)

	r := runPage(t, "images", s)
	if !r.has("圖片數：2") || !r.has("下載 book.epub") {
		t.Error("images page not drawn")
	}
	if r.has("以第一張圖片作為封面") {
		t.Error("first-as-cover offered with a cover")
	}
	download(t, "images", s)
}

func TestImagesPageError(t *testing.T) {
	s := newState(t)
	uploadMany(t, s, imagesFileID, map[string][]byte{"a.zip": []byte("not a zip")}, []string{"a.zip"})

	r := runPage(t, "images", s)
	if r.find("message_component") == nil || r.find("download_file_component") != nil {
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
	s := newState(t)
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
