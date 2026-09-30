package imgbook

import (
	"archive/zip"
	"bytes"
	"slices"
	"testing"
)

const png = "\x89PNG\r\n\x1a\n"

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

func TestReadZip(t *testing.T) {
	data := makeZip(t, map[string]string{
		"p10.png":           png,
		"p2.png":            png,
		"p1.jpg":            "\xff\xd8\xff",
		"notes.txt":         "hi",
		"__MACOSX/._p1.jpg": png,
		".DS_Store":         png,
	})
	imgs, err := ReadZip(data)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, img := range imgs {
		names = append(names, img.Name)
	}
	if want := []string{"p1.jpg", "p2.png", "p10.png"}; !slices.Equal(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
}

func TestReadZipErrors(t *testing.T) {
	if _, err := ReadZip([]byte("not a zip")); err == nil {
		t.Error("want error for bad zip")
	}
	if _, err := ReadZip(makeZip(t, map[string]string{"a.txt": "x"})); err == nil {
		t.Error("want error for no images")
	}
}

func TestNaturalCompare(t *testing.T) {
	got := []string{"a10", "a2", "a02b", "b", "a1", "a"}
	slices.SortFunc(got, NaturalCompare)
	want := []string{"a", "a1", "a2", "a02b", "a10", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("sorted = %v, want %v", got, want)
	}
}

func TestBuild(t *testing.T) {
	data, err := Build(Meta{Title: "t", Cover: []byte(png)}, []Image{{"a.png", []byte(png)}})
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	for _, want := range []string{"EPUB/images/cover.png", "EPUB/images/img_00001.png", "EPUB/sec_00001.xhtml"} {
		if !slices.Contains(names, want) {
			t.Errorf("missing %s in %v", want, names)
		}
	}

	if _, err := Build(Meta{}, []Image{{"bad", []byte("x")}}); err == nil {
		t.Error("want error for non-image")
	}
}
