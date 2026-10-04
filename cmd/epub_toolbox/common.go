package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

// memoEntry holds the latest value of a memo slot.
type memoEntry[T any] struct {
	key string
	v   T
}

// memo returns f() cached in slot, recomputing when key changes.
// Only the latest value is kept, so memory stays bounded.
func memo[T any](s *tgframe.State, slot, key string, f func() T) T {
	if e, ok := s.GetFuncCache[memoEntry[T]](slot); ok && e.key == key {
		return e.v
	}
	v := f()
	s.SetFuncCache(slot, memoEntry[T]{key, v})
	return v
}

func hashOf(parts ...any) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprintf(h, "%v\x00", p)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

func hashBytes(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])[:16]
}

// bookMeta is what metaForm collects.
type bookMeta struct {
	Title, Author, Intro, Language string

	coverFile *tgcomp.FileObject
}

func (m bookMeta) hasCover() bool {
	return m.coverFile != nil
}

// cover reads the uploaded cover, or returns nil if none.
func (m bookMeta) cover() ([]byte, error) {
	if m.coverFile == nil {
		return nil, nil
	}
	return m.coverFile.Bytes()
}

// metaForm draws the metadata inputs. resetKey resets them for a new input file.
func metaForm(c *tgframe.Container, resetKey, defTitle, defIntro string) bookMeta {
	m := bookMeta{
		Title: tgcomp.Textbox(c, tr("書名", "Title"), &tgcomp.TextboxConf{
			Default: defTitle, ResetKey: resetKey,
		}),
		Author: tgcomp.Textbox(c, tr("作者", "Author")),
		Intro: tgcomp.Textarea(c, tr("簡介", "Introduction"), &tgcomp.TextareaConf{
			Default: defIntro, ResetKey: resetKey, Height: 8,
		}),
		Language: tgcomp.Textbox(c, tr("語言", "Language"), &tgcomp.TextboxConf{
			Default: i18n.Tag(), Placeholder: tr("BCP 47 標籤，例如 zh-TW、en", "BCP 47 tag, e.g. en, zh-TW"),
		}),
	}
	m.coverFile = tgcomp.FileUpload(c, tr("封面", "Cover"), ".png,.jpg,.jpeg,.gif,.webp")

	m.Title = strings.TrimSpace(m.Title)
	m.Author = strings.TrimSpace(m.Author)
	m.Intro = strings.TrimSpace(m.Intro)
	m.Language = strings.TrimSpace(m.Language)
	return m
}

// track reports an analytics event; a no-op unless main sets it.
var track = func(event string, data map[string]any) {}

// exportEpub draws the download button; build runs only on click.
func exportEpub(p *tgframe.Params, tool, title string, build func() ([]byte, error)) {
	tgcomp.Divider(p.Main)
	filename := safeFilename(title) + ".epub"
	tgcomp.DownloadFileFunc(p.Main, tr("下載 ", "Download ")+filename, func() ([]byte, error) {
		track("download", map[string]any{"tool": tool})
		return build()
	}, &tgcomp.DownloadFileConf{
		Filename: filename,
		MIME:     "application/epub+zip",
	})
}

// safeFilename replaces characters most file systems reject.
func safeFilename(s string) string {
	s = strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return '_'
		}
		return r
	}, s)
	if s = strings.TrimSpace(s); s == "" {
		return "book"
	}
	return s
}

// cutExt removes ext from name, ignoring case, and reports whether it was there.
func cutExt(name, ext string) (string, bool) {
	if len(name) >= len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
		return name[:len(name)-len(ext)], true
	}
	return name, false
}
