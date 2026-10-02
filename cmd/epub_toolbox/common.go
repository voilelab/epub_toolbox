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

// metaForm draws the metadata inputs. resetKey resets them for a new input file.
func metaForm(c *tgframe.Container, resetKey, defTitle, defIntro string) bookMeta {
	m := bookMeta{
		Title: tgcomp.Textbox(c, "書名", &tgcomp.TextboxConf{
			Default: defTitle, ResetKey: resetKey,
		}),
		Author: tgcomp.Textbox(c, "作者"),
		Intro: tgcomp.Textarea(c, "簡介", &tgcomp.TextareaConf{
			Default: defIntro, ResetKey: resetKey, Height: 8,
		}),
		Language: tgcomp.Textbox(c, "語言", &tgcomp.TextboxConf{
			Default: "zh-TW", Placeholder: "BCP 47 標籤，例如 zh-TW、en",
		}),
	}
	m.coverFile = tgcomp.FileUpload(c, "封面", ".png,.jpg,.jpeg,.gif,.webp")
	if m.coverFile == nil {
		m.coverURL = strings.TrimSpace(tgcomp.Textbox(c, "封面網址"))
	}

	m.Title = strings.TrimSpace(m.Title)
	m.Author = strings.TrimSpace(m.Author)
	m.Intro = strings.TrimSpace(m.Intro)
	m.Language = strings.TrimSpace(m.Language)
	return m
}

// exportEpub draws the download button; build runs only on click.
func exportEpub(p *tgframe.Params, title string, build func() ([]byte, error)) {
	tgcomp.Divider(p.Main)
	filename := safeFilename(title) + ".epub"
	tgcomp.DownloadFileFunc(p.Main, "下載 "+filename, build, &tgcomp.DownloadFileConf{
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
