package main

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestBugURL(t *testing.T) {
	u, err := url.Parse(bugURL(toolNovel, `invalid regex "(": x`))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("template") != "bug.yml" || q.Get("tool") != toolNovel || q.Get("error") != `invalid regex "(": x` {
		t.Errorf("query = %v", q)
	}
	// Markdown link targets must not contain ( ) or spaces.
	if s := u.RawQuery; strings.ContainsAny(s, "() ") {
		t.Errorf("raw query %q breaks Markdown links", s)
	}
	if q := mustQuery(t, bugURL("", "")); q.Has("tool") || q.Has("error") {
		t.Errorf("empty fields prefilled: %v", q)
	}
}

func mustQuery(t *testing.T, s string) url.Values {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query()
}

// TestBugTemplateTools keeps the prefilled tools in sync with the form's dropdown.
func TestBugTemplateTools(t *testing.T) {
	b, err := os.ReadFile("../../.github/ISSUE_TEMPLATE/bug.yml")
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{toolNovel, toolImages} {
		if !strings.Contains(string(b), "- "+tool+"\n") {
			t.Errorf("bug.yml has no option %q", tool)
		}
	}
}
