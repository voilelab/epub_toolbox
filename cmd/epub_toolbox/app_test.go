package main

import (
	"testing"
	"time"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

// TestPagesRender draws every page once per language with an empty state.
func TestPagesRender(t *testing.T) {
	defer i18n.Set("zh-TW")
	for _, lang := range []string{"zh-TW", "en"} {
		i18n.Set(lang)
		for _, page := range []string{"index", "novel", "regex", "images"} {
			t.Run(lang+"/"+page, func(t *testing.T) { testPageRender(t, page) })
		}
	}
}

func testPageRender(t *testing.T, page string) {
	packs := make(chan any, 256)
	s, err := tgframe.NewSession(newApp(), page, tgframe.NewState(), func(pack any) error {
		packs <- pack
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if err := s.HandleRawEvent([]byte("{}")); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case pack := <-packs:
			if r, ok := pack.(*tgframe.ResultPack); ok {
				if !r.Success {
					t.Fatalf("run failed: %s", r.Error)
				}
				return
			}
		case <-time.After(5 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestSafeFilename(t *testing.T) {
	for in, want := range map[string]string{"a/b:c": "a_b_c", "  ": "book", "書名": "書名"} {
		if got := safeFilename(in); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
