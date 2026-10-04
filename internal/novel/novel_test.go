package novel

import (
	"archive/zip"
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

func TestDecode(t *testing.T) {
	text := strings.Repeat("第一章 開始\n從前從前，有一座山。山上有一座廟。\n", 20)
	big5, _ := traditionalchinese.Big5.NewEncoder().String(text)
	gbk, _ := simplifiedchinese.GB18030.NewEncoder().String(strings.Repeat("第一章 开始\n从前从前，有一座山。山上有一座庙。\n", 20))

	tests := []struct {
		name, data, want string
	}{
		{"utf8", text, "UTF-8"},
		{"utf8 bom", "\xef\xbb\xbf" + text, "UTF-8"},
		{"big5", big5, "Big5"},
		{"gb18030", gbk, "GB-18030"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encs := Encodings([]byte(tt.data))
			if len(encs) == 0 || encs[0] != tt.want {
				t.Fatalf("Encodings = %v, want %s first", encs, tt.want)
			}
			got := Decode([]byte(tt.data), encs[0])
			if !strings.HasPrefix(got, "第一章") {
				t.Errorf("Decode = %q", got[:20])
			}
		})
	}
}

func TestEncodingsFallback(t *testing.T) {
	encs := Encodings(nil)
	for _, f := range fallbacks {
		if !slices.Contains(encs, f) {
			t.Errorf("missing fallback %s in %v", f, encs)
		}
	}
}

func TestSplit(t *testing.T) {
	lines := Lines("序\r\n\n第一章 A\na1\n\n第二章 B（不是標題）\n第2章 C\nc1\nx第三章", true)
	m, err := NewMatcher([]string{"第.*章", `第\d+章`}, []string{"不是標題"})
	if err != nil {
		t.Fatal(err)
	}
	chs := Split(lines, m)

	var got []string
	for _, c := range chs {
		got = append(got, c.Title+"|"+c.Content(0))
	}
	want := []string{
		HeadTitle() + "|序",
		"第一章 A|a1\n第二章 B（不是標題）",
		"第2章 C|c1\nx第三章",
	}
	if !slices.Equal(got, want) {
		t.Errorf("Split =\n%q\nwant\n%q", got, want)
	}
	if !chs[0].Head || chs[1].Head {
		t.Error("head flag wrong")
	}
}

func TestNewMatcherInvalid(t *testing.T) {
	if _, err := NewMatcher([]string{"("}, nil); err == nil {
		t.Error("want error")
	}
}

func TestRemoveEmpty(t *testing.T) {
	chs := RemoveEmpty([]Chapter{{Title: "h", Head: true}, {Title: "a", Lines: []string{"x"}}, {Title: "b"}})
	if len(chs) != 1 || chs[0].Title != "a" {
		t.Errorf("RemoveEmpty = %+v", chs)
	}
}

func TestBuild(t *testing.T) {
	chs := []Chapter{
		{Title: HeadTitle(), Head: true, Lines: []string{"head"}},
		{Title: "第一章", Lines: []string{"a"}},
		{Title: "第二章", Lines: []string{"b"}},
	}
	data, err := Build(Meta{Title: "書", Intro: "簡介"}, chs)
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var secs int
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "EPUB/sec_") {
			secs++
		}
	}
	// intro + 2 chapters, head skipped
	if secs != 3 {
		t.Errorf("sections = %d, want 3", secs)
	}
}

func TestBuildStyle(t *testing.T) {
	chs := []Chapter{{Title: "第一章", Lines: []string{"a"}}}
	for _, tp := range Templates {
		s := tp.Style()
		data, err := Build(Meta{Title: "書", Style: s}, chs)
		if err != nil {
			t.Fatal(err)
		}
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatal(err)
		}
		files := map[string]string{}
		for _, f := range zr.File {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			files[f.Name] = string(b)
		}
		if files["EPUB/style.css"] != s.CSS() {
			t.Errorf("%s: style.css not from style", tp.Name())
		}
		vertical := strings.Contains(files["EPUB/style.css"], "writing-mode: vertical-rl")
		rtl := strings.Contains(files["EPUB/content.opf"], `page-progression-direction="rtl"`)
		if want := tp == TemplateVertical; vertical != want || rtl != want {
			t.Errorf("%s: vertical css = %v, rtl spine = %v, want %v", tp.Name(), vertical, rtl, want)
		}
	}
}
