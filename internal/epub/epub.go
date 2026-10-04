// Package epub writes EPUB 3 books with only the standard library.
package epub

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"time"

	"github.com/voilelab/epub_toolbox/internal/i18n"
)

// ErrEmpty is returned when a book has nothing to put in its spine.
var ErrEmpty = errors.New("epub: book has no content")

// ErrImageType is returned for image data that is not PNG, JPEG, GIF or WebP.
var ErrImageType = errors.New("epub: unsupported image type")

// Book is an EPUB book being assembled. Zero values are usable.
type Book struct {
	Title       string
	Author      string
	Description string
	Language    string    // BCP 47 tag, "und" if empty
	ID          string    // random urn:uuid if empty
	Modified    time.Time // now if zero
	CSS         string    // shared by every page
	WritingMode string    // e.g. "vertical-rl"; a "-rl" mode pages right to left

	cover    *resource
	images   []resource
	sections []section
}

type resource struct {
	href, mime string
	data       []byte
}

type section struct {
	title, href, body string
}

// SetCover sets the cover image.
func (b *Book) SetCover(data []byte) error {
	mime, ext, ok := ImageType(data)
	if !ok {
		return ErrImageType
	}
	b.cover = &resource{href: "images/cover" + ext, mime: mime, data: data}
	return nil
}

// AddImage adds an image and returns the href a section body uses for it.
func (b *Book) AddImage(data []byte) (string, error) {
	mime, ext, ok := ImageType(data)
	if !ok {
		return "", ErrImageType
	}
	href := fmt.Sprintf("images/img_%05d%s", len(b.images)+1, ext)
	b.images = append(b.images, resource{href: href, mime: mime, data: data})
	return href, nil
}

// AddSection appends a page to the spine and the table of contents.
// body is XHTML markup placed inside <body>.
func (b *Book) AddSection(title, body string) {
	href := fmt.Sprintf("sec_%05d.xhtml", len(b.sections)+1)
	b.sections = append(b.sections, section{title: title, href: href, body: body})
}

// Bytes returns the book as an EPUB file.
func (b *Book) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := b.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write writes the book as an EPUB file to w.
func (b *Book) Write(w io.Writer) error {
	if len(b.sections) == 0 && b.cover == nil {
		return ErrEmpty
	}

	m := *b
	if m.Title == "" {
		m.Title = i18n.T("未命名", "Untitled")
	}
	if m.Language == "" {
		m.Language = "und"
	}
	if m.ID == "" {
		m.ID = newUUID()
	}
	if m.Modified.IsZero() {
		m.Modified = time.Now()
	}

	zw := zip.NewWriter(w)

	// mimetype: first entry, stored, no extra field.
	fw, err := zw.CreateHeader(&zip.FileHeader{Name: "mimetype", Method: zip.Store})
	if err != nil {
		return err
	}
	if _, err := io.WriteString(fw, "application/epub+zip"); err != nil {
		return err
	}

	put := func(name string, data []byte, method uint16) error {
		fw, err := zw.CreateHeader(&zip.FileHeader{
			Name: name, Method: method, Modified: m.Modified,
		})
		if err != nil {
			return err
		}
		_, err = fw.Write(data)
		return err
	}

	files := []struct {
		name string
		data string
	}{
		{"META-INF/container.xml", containerXML},
		{"EPUB/content.opf", m.opf()},
		{"EPUB/nav.xhtml", m.nav()},
		{"EPUB/toc.ncx", m.ncx()},
		{"EPUB/style.css", m.CSS},
	}
	if m.cover != nil {
		files = append(files, struct{ name, data string }{
			"EPUB/cover.xhtml",
			m.page(i18n.T("封面", "Cover"), fmt.Sprintf(`<div class="cover"><img src="%s" alt="%s"/></div>`,
				m.cover.href, Escape(i18n.T("封面", "Cover")))),
		})
	}
	for _, s := range m.sections {
		files = append(files, struct{ name, data string }{"EPUB/" + s.href, m.page(s.title, s.body)})
	}
	for _, f := range files {
		if err := put(f.name, []byte(f.data), zip.Deflate); err != nil {
			return err
		}
	}

	// Images are already compressed.
	for _, r := range m.resources() {
		if err := put("EPUB/"+r.href, r.data, zip.Store); err != nil {
			return err
		}
	}
	return zw.Close()
}

func (b *Book) resources() []resource {
	if b.cover == nil {
		return b.images
	}
	return append([]resource{*b.cover}, b.images...)
}

const containerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>
`

func (b *Book) opf() string {
	var meta, items, spine strings.Builder

	if b.Author != "" {
		fmt.Fprintf(&meta, "    <dc:creator>%s</dc:creator>\n", Escape(b.Author))
	}
	if b.Description != "" {
		fmt.Fprintf(&meta, "    <dc:description>%s</dc:description>\n", Escape(b.Description))
	}
	if b.cover != nil {
		meta.WriteString("    <meta name=\"cover\" content=\"cover-image\"/>\n")
		fmt.Fprintf(&items, "    <item id=\"cover-image\" href=\"%s\" media-type=\"%s\" properties=\"cover-image\"/>\n",
			b.cover.href, b.cover.mime)
		items.WriteString("    <item id=\"cover\" href=\"cover.xhtml\" media-type=\"application/xhtml+xml\"/>\n")
		spine.WriteString("    <itemref idref=\"cover\"/>\n")
	}
	if b.WritingMode != "" {
		// For Kindle; the CSS sets the actual writing mode.
		fmt.Fprintf(&meta, "    <meta name=\"primary-writing-mode\" content=\"%s\"/>\n", Escape(b.WritingMode))
	}
	spineAttr := ""
	if strings.HasSuffix(b.WritingMode, "-rl") {
		spineAttr = ` page-progression-direction="rtl"`
	}
	for i, r := range b.images {
		fmt.Fprintf(&items, "    <item id=\"img%d\" href=\"%s\" media-type=\"%s\"/>\n", i+1, r.href, r.mime)
	}
	for i, s := range b.sections {
		fmt.Fprintf(&items, "    <item id=\"sec%d\" href=\"%s\" media-type=\"application/xhtml+xml\"/>\n", i+1, s.href)
		fmt.Fprintf(&spine, "    <itemref idref=\"sec%d\"/>\n", i+1)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="book-id" xml:lang="%[1]s">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="book-id">%[2]s</dc:identifier>
    <dc:title>%[3]s</dc:title>
    <dc:language>%[1]s</dc:language>
%[4]s    <meta property="dcterms:modified">%[5]s</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="css" href="style.css" media-type="text/css"/>
%[6]s  </manifest>
  <spine toc="ncx"%[8]s>
%[7]s  </spine>
</package>
`, Escape(b.Language), Escape(b.ID), Escape(b.Title), meta.String(),
		b.Modified.UTC().Format("2006-01-02T15:04:05Z"), items.String(), spine.String(), spineAttr)
}

func (b *Book) nav() string {
	var li strings.Builder
	for _, s := range b.sections {
		fmt.Fprintf(&li, "      <li><a href=\"%s\">%s</a></li>\n", s.href, Escape(s.title))
	}
	// An empty <ol> is invalid, so point at the cover instead.
	if len(b.sections) == 0 {
		li.WriteString("      <li><a href=\"cover.xhtml\">Cover</a></li>\n")
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" lang="%[1]s" xml:lang="%[1]s">
<head><title>%[2]s</title></head>
<body>
  <nav epub:type="toc" id="toc">
    <h1>%[2]s</h1>
    <ol>
%[3]s    </ol>
  </nav>
</body>
</html>
`, Escape(b.Language), Escape(b.Title), li.String())
}

func (b *Book) ncx() string {
	var pts strings.Builder
	for i, s := range b.sections {
		fmt.Fprintf(&pts, "    <navPoint id=\"np%d\" playOrder=\"%d\"><navLabel><text>%s</text></navLabel><content src=\"%s\"/></navPoint>\n",
			i+1, i+1, Escape(s.title), s.href)
	}
	if len(b.sections) == 0 {
		pts.WriteString("    <navPoint id=\"np1\" playOrder=\"1\"><navLabel><text>Cover</text></navLabel><content src=\"cover.xhtml\"/></navPoint>\n")
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head><meta name="dtb:uid" content="%s"/></head>
  <docTitle><text>%s</text></docTitle>
  <navMap>
%s  </navMap>
</ncx>
`, Escape(b.ID), Escape(b.Title), pts.String())
}

func (b *Book) page(title, body string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" lang="%[1]s" xml:lang="%[1]s">
<head>
  <title>%[2]s</title>
  <link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
%[3]s
</body>
</html>
`, Escape(b.Language), Escape(title), body)
}

// Escape escapes s for XML text or attributes and drops characters XML 1.0
// forbids, such as NUL and other control codes.
func Escape(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return r
		case r < 0x20, r == 0xFFFE, r == 0xFFFF, r >= 0xD800 && r <= 0xDFFF:
			return -1
		}
		return r
	}, s)
	return html.EscapeString(s)
}

// Paragraphs renders each line as an escaped <p>.
func Paragraphs(lines []string) string {
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString("<p>")
		sb.WriteString(Escape(l))
		sb.WriteString("</p>\n")
	}
	return sb.String()
}

// ImageType sniffs PNG, JPEG, GIF and WebP by magic bytes.
func ImageType(b []byte) (mime, ext string, ok bool) {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "image/png", ".png", true
	case bytes.HasPrefix(b, []byte("\xff\xd8\xff")):
		return "image/jpeg", ".jpg", true
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return "image/gif", ".gif", true
	case len(b) >= 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "image/webp", ".webp", true
	}
	return "", "", false
}

func newUUID() string {
	var u [16]byte
	_, _ = rand.Read(u[:])
	u[6] = u[6]&0x0f | 0x40
	u[8] = u[8]&0x3f | 0x80
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}
