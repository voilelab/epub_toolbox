package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

var pngData = []byte("\x89PNG\r\n\x1a\n rest")

func readZip(t *testing.T, data []byte) (*zip.Reader, map[string]string) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		files[f.Name] = string(b)
	}
	return zr, files
}

func TestWrite(t *testing.T) {
	b := &Book{
		Title:       "A & B",
		Author:      "<me>",
		Description: "intro",
		Language:    "zh-TW",
		ID:          "id-1",
		Modified:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	if err := b.SetCover(pngData); err != nil {
		t.Fatal(err)
	}
	href, err := b.AddImage(pngData)
	if err != nil {
		t.Fatal(err)
	}
	b.AddSection("第一章 <1>", Paragraphs([]string{"a < b", "\x00bad\x1b"}))
	b.AddSection("img", `<img src="`+href+`" alt=""/>`)

	data, err := b.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	zr, files := readZip(t, data)

	first := zr.File[0]
	if first.Name != "mimetype" || first.Method != zip.Store || len(first.Extra) != 0 {
		t.Errorf("bad mimetype entry: %+v", first.FileHeader)
	}
	if files["mimetype"] != "application/epub+zip" {
		t.Errorf("mimetype = %q", files["mimetype"])
	}

	for name, body := range files {
		if !strings.HasSuffix(name, ".xml") && !strings.HasSuffix(name, ".opf") &&
			!strings.HasSuffix(name, ".xhtml") && !strings.HasSuffix(name, ".ncx") {
			continue
		}
		d := xml.NewDecoder(strings.NewReader(body))
		d.Strict = true
		d.Entity = xml.HTMLEntity
		for {
			_, err := d.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("%s not well-formed: %v\n%s", name, err, body)
			}
		}
	}

	opf := files["EPUB/content.opf"]
	for _, want := range []string{
		"<dc:title>A &amp; B</dc:title>",
		"<dc:creator>&lt;me&gt;</dc:creator>",
		"<dc:language>zh-TW</dc:language>",
		`properties="cover-image"`,
		`href="images/img_00001.png" media-type="image/png"`,
		"2026-01-02T03:04:05Z",
		`<itemref idref="cover"/>`,
	} {
		if !strings.Contains(opf, want) {
			t.Errorf("opf missing %q", want)
		}
	}

	sec := files["EPUB/sec_00001.xhtml"]
	if !strings.Contains(sec, "<p>a &lt; b</p>") || !strings.Contains(sec, "<p>bad</p>") {
		t.Errorf("section body not escaped:\n%s", sec)
	}
	if _, ok := files["EPUB/images/cover.png"]; !ok {
		t.Error("cover image missing")
	}
}

func TestWriteEmpty(t *testing.T) {
	if _, err := (&Book{}).Bytes(); !errors.Is(err, ErrEmpty) {
		t.Errorf("err = %v, want ErrEmpty", err)
	}
}

func TestImageType(t *testing.T) {
	tests := []struct {
		data string
		ext  string
	}{
		{"\x89PNG\r\n\x1a\n", ".png"},
		{"\xff\xd8\xff\xe0", ".jpg"},
		{"GIF89a", ".gif"},
		{"RIFF\x00\x00\x00\x00WEBPVP8 ", ".webp"},
		{"hello", ""},
	}
	for _, tt := range tests {
		_, ext, ok := ImageType([]byte(tt.data))
		if ext != tt.ext || ok != (tt.ext != "") {
			t.Errorf("ImageType(%q) = %q, %v", tt.data, ext, ok)
		}
	}
	if err := (&Book{}).SetCover([]byte("nope")); !errors.Is(err, ErrImageType) {
		t.Errorf("SetCover err = %v", err)
	}
}
