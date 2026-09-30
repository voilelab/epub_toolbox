package main

import (
	"testing"
	"time"

	"github.com/voilelab/toolgui/toolgui/tgframe"
)

// TestPagesRender draws every page once with an empty state.
func TestPagesRender(t *testing.T) {
	for _, page := range []string{"index", "novel", "regex", "images"} {
		t.Run(page, func(t *testing.T) {
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
		})
	}
}

func TestSafeFilename(t *testing.T) {
	for in, want := range map[string]string{"a/b:c": "a_b_c", "  ": "book", "書名": "書名"} {
		if got := safeFilename(in); got != want {
			t.Errorf("safeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}
