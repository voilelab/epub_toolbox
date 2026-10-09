package main

import (
	"testing"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgtest"
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
	if err := tgtest.Open(t, newApp(), page).Err(); err != nil {
		t.Fatalf("run failed: %v", err)
	}
}

func TestSafeFilename(t *testing.T) {
	for in, want := range map[string]string{"a/b:c": "a_b_c", "  ": "book", "書名": "書名"} {
		if got := safeFilename(in); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCutExt(t *testing.T) {
	tests := []struct {
		name, ext, want string
		ok              bool
	}{
		{"book.txt", ".txt", "book", true},
		{"BOOK.TXT", ".txt", "BOOK", true},
		{"pics.Zip", ".zip", "pics", true},
		{"book", ".txt", "book", false},
		{"xt", ".txt", "xt", false},
	}
	for _, tt := range tests {
		if got, ok := cutExt(tt.name, tt.ext); got != tt.want || ok != tt.ok {
			t.Errorf("cutExt(%q, %q) = %q, %v", tt.name, tt.ext, got, ok)
		}
	}
}
