// Package imgbook packs images into an EPUB.
package imgbook

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/voilelab/epub_toolbox/internal/epub"
)

// MaxImageSize caps one decompressed image, against zip bombs.
const MaxImageSize = 64 << 20

// Image is one page.
type Image struct {
	Name string
	Data []byte
}

// Read returns the images among files, expanding zips, in natural filename
// order. Other files and macOS metadata are skipped.
func Read(files []Image) ([]Image, error) {
	var imgs []Image
	for _, f := range files {
		if bytes.HasPrefix(f.Data, []byte("PK\x03\x04")) {
			zimgs, err := readZip(f.Data)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", f.Name, err)
			}
			imgs = append(imgs, zimgs...)
			continue
		}
		if skip(f.Name) {
			continue
		}
		if len(f.Data) > MaxImageSize {
			return nil, fmt.Errorf("%s：超過 %d MiB", f.Name, MaxImageSize>>20)
		}
		if _, _, ok := epub.ImageType(f.Data); ok {
			imgs = append(imgs, f)
		}
	}
	if len(imgs) == 0 {
		return nil, errors.New("找不到 PNG、JPEG、GIF 或 WebP 圖片")
	}

	slices.SortStableFunc(imgs, func(a, b Image) int { return NaturalCompare(a.Name, b.Name) })
	return imgs, nil
}

func readZip(data []byte) ([]Image, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("讀取 zip：%w", err)
	}

	var imgs []Image
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || skip(f.Name) {
			continue
		}
		if f.UncompressedSize64 > MaxImageSize {
			return nil, fmt.Errorf("%s：超過 %d MiB", f.Name, MaxImageSize>>20)
		}
		b, err := readFile(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		if _, _, ok := epub.ImageType(b); ok {
			imgs = append(imgs, Image{Name: f.Name, Data: b})
		}
	}
	return imgs, nil
}

func skip(name string) bool {
	return strings.HasPrefix(name, "__MACOSX/") || strings.HasPrefix(path.Base(name), ".")
}

func readFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, MaxImageSize+1))
}

// NaturalCompare compares strings so that "2" sorts before "10".
func NaturalCompare(a, b string) int {
	for a != "" && b != "" {
		da, db := digits(a), digits(b)
		if da > 0 && db > 0 {
			na := strings.TrimLeft(a[:da], "0")
			nb := strings.TrimLeft(b[:db], "0")
			if c := len(na) - len(nb); c != 0 {
				return c
			}
			if c := strings.Compare(na, nb); c != 0 {
				return c
			}
			a, b = a[da:], b[db:]
			continue
		}
		if a[0] != b[0] {
			return int(a[0]) - int(b[0])
		}
		a, b = a[1:], b[1:]
	}
	return len(a) - len(b)
}

func digits(s string) int {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// Meta is the book metadata.
type Meta struct {
	Title, Author, Intro, Language string
	Cover                          []byte
}

// Build builds an EPUB with one page per image.
func Build(meta Meta, imgs []Image) ([]byte, error) {
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
		b.AddSection("簡介", epub.Paragraphs(strings.Split(meta.Intro, "\n")))
	}
	for i, img := range imgs {
		href, err := b.AddImage(img.Data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", img.Name, err)
		}
		title := fmt.Sprintf("第 %d 頁", i+1)
		b.AddSection(title, fmt.Sprintf(`<div class="page"><img src="%s" alt="%s"/></div>`, href, title))
	}
	return b.Bytes()
}

const css = `body { margin: 0; padding: 0; }
.page, .cover { text-align: center; }
.page img, .cover img { max-width: 100%; max-height: 100vh; }
`
