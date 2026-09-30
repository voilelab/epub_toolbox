package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/voilelab/epub_toolbox/internal/fetch"
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
	coverURL  string
}

// key identifies the inputs, for invalidating a prepared EPUB.
func (m bookMeta) key() string {
	cover := m.coverURL
	if m.coverFile != nil {
		cover = fmt.Sprintf("file:%s:%d", m.coverFile.Name, m.coverFile.Size)
	}
	return hashOf(m.Title, m.Author, m.Intro, m.Language, cover)
}

func (m bookMeta) hasCover() bool {
	return m.coverFile != nil || m.coverURL != ""
}

// cover reads the uploaded cover or downloads it from the URL.
func (m bookMeta) cover(ctx context.Context) ([]byte, error) {
	if m.coverFile != nil {
		return m.coverFile.Bytes()
	}
	if m.coverURL != "" {
		return fetch.Image(ctx, m.coverURL)
	}
	return nil, nil
}

// metaForm draws the metadata inputs. idSuffix resets them for a new input file.
func metaForm(c *tgframe.Container, idSuffix, defTitle, defIntro string) bookMeta {
	m := bookMeta{
		Title: tgcomp.Textbox(c, "Title", &tgcomp.TextboxConf{
			Base: tgframe.Base{ID: "title_" + idSuffix}, Default: defTitle,
		}),
		Author: tgcomp.Textbox(c, "Author"),
		Intro: tgcomp.Textarea(c, "Introduction", &tgcomp.TextareaConf{
			Base: tgframe.Base{ID: "intro_" + idSuffix}, Default: defIntro, Height: 8,
		}),
		Language: tgcomp.Textbox(c, "Language", &tgcomp.TextboxConf{
			Default: "zh-TW", Placeholder: "BCP 47 tag, e.g. zh-TW, en",
		}),
	}
	m.coverFile = tgcomp.FileUpload(c, "Book cover", ".png,.jpg,.jpeg,.gif,.webp")
	if m.coverFile == nil {
		m.coverURL = strings.TrimSpace(tgcomp.Textbox(c, "Book cover URL"))
	}

	m.Title = strings.TrimSpace(m.Title)
	m.Author = strings.TrimSpace(m.Author)
	m.Intro = strings.TrimSpace(m.Intro)
	m.Language = strings.TrimSpace(m.Language)
	return m
}

// preparedEpub is the last EPUB built, tagged with its inputs.
type preparedEpub struct {
	key  string
	data []byte
}

// exportEpub draws the Prepare button and, once built for the current inputs
// (key), the download button.
func exportEpub(p *tgframe.Params, key, title string, build func() ([]byte, error)) {
	c := p.Main
	tgcomp.Divider(c)

	if tgcomp.Button(c, "Prepare EPUB") {
		status := tgcomp.Status(c, "Preparing EPUB...", &tgcomp.StatusConf{Expanded: true})
		data, err := build()
		if err != nil {
			status.Fail("Preparing EPUB failed")
			tgcomp.MessageDanger(c, err.Error())
			return
		}
		p.State.SetFuncCache("epub", preparedEpub{key, data})
		status.Complete(fmt.Sprintf("Preparing EPUB complete! (%.1f KiB)", float64(len(data))/1024))
	}

	prepared, ok := p.State.GetFuncCache[preparedEpub]("epub")
	if !ok || prepared.key != key {
		return
	}
	filename := safeFilename(title) + ".epub"
	tgcomp.DownloadFile(c, "Download "+filename, prepared.data, &tgcomp.DownloadFileConf{
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
