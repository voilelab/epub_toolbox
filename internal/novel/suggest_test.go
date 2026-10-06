package novel

import (
	"fmt"
	"strings"
	"testing"
)

var cnNums = []string{"一", "二", "三", "四", "五", "六", "七", "八", "九", "十", "十一", "十二"}

// book builds n chapters of body lines with title(i), i from 1.
func book(n int, blank bool, title func(i int) string) []string {
	var out []string
	for i := 1; i <= n; i++ {
		out = append(out, title(i))
		if blank {
			out = append(out, "")
		}
		for j := range 8 {
			out = append(out, fmt.Sprintf("　　他走了第%d步，心想還有很遠。", j))
		}
		if blank {
			out = append(out, "")
		}
	}
	return out
}

func matchAll(t *testing.T, pattern string, yes, no []string) {
	t.Helper()
	m, err := NewMatcher([]string{pattern}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range yes {
		if !m.IsTitle(l) {
			t.Errorf("%q: %q not a title", pattern, l)
		}
	}
	for _, l := range no {
		if m.IsTitle(l) {
			t.Errorf("%q: %q is a title", pattern, l)
		}
	}
}

func TestSuggest(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		yes, no []string
	}{
		{
			name:  "chinese chapters",
			lines: book(12, true, func(i int) string { return "第" + cnNums[i-1] + "章 風起" }),
			yes:   []string{"第十二章 風起", "　　第三章"},
			// Only numerals seen in the book are listed.
			no: []string{"第三天他終於讀完那一章", "他說第一章很好看", "第3章"},
		},
		{
			name: "prologue and prose",
			lines: append(append([]string{"楔子", "很久以前。", "第三天他終於讀完那一章。"},
				book(6, false, func(i int) string { return fmt.Sprintf("第%d回　大戰", i) })...),
				"番外 後日談", "完。"),
			yes: []string{"第6回　大戰", "楔子", "番外 後日談"},
			no:  []string{"第三天他終於讀完那一章。", "很久以前。"},
		},
		{
			name:  "brackets",
			lines: book(10, true, func(i int) string { return "【" + cnNums[i-1] + "】" }),
			yes:   []string{"【十】"},
			no:    []string{"十個人"},
		},
		{
			name: "volumes restart",
			lines: append(book(4, true, func(i int) string { return "第一卷 第" + cnNums[i-1] + "章" }),
				book(4, true, func(i int) string { return "第二卷 第" + cnNums[i-1] + "章" })...),
			yes: []string{"第二卷 第四章"},
		},
		{
			name:  "english",
			lines: book(5, true, func(i int) string { return fmt.Sprintf("Chapter %d: Wind", i) }),
			yes:   []string{"Chapter 12: X", "Chapter  3:"},
			no:    []string{"Chapters follow."},
		},
		{
			name:  "bare numbers",
			lines: book(5, true, func(i int) string { return fmt.Sprint(i) }),
			yes:   []string{"7", "8 風起"},
			no:    []string{"1984年的夏天"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Suggest(tt.lines)
			if len(got) == 0 {
				t.Fatal("no suggestion")
			}
			t.Logf("%+v", got)
			best := got[0]
			if best.Score < 0.5 {
				t.Errorf("score = %.2f, want >= 0.5 (%+v)", best.Score, got)
			}
			matchAll(t, best.Pattern, tt.yes, tt.no)
		})
	}
}

func TestSuggestRejects(t *testing.T) {
	// A numbered list inside one chapter is not a title pattern.
	lines := []string{"第一章", "內容。"}
	for i := 1; i <= 5; i++ {
		lines = append(lines, fmt.Sprintf("%d. 第%d點要注意。", i, i))
	}
	lines = append(lines, Lines(strings.Repeat("內容。\n", 50), false)...)
	for _, s := range Suggest(lines) {
		if s.Score >= 0.5 {
			t.Errorf("confident on %+v", s)
		}
	}
	if got := Suggest([]string{"第一章", "a", "第二章", "b"}); len(got) != 0 {
		t.Errorf("too few titles: %+v", got)
	}
}

func TestSuggestProse(t *testing.T) {
	// Numbered sentences restart in every chapter, like titles of volumes.
	lines := book(12, true, func(i int) string { return "第" + cnNums[i-1] + "章" })
	for _, s := range Suggest(lines) {
		if strings.Contains(s.Pattern, "他走了") {
			t.Errorf("prose suggested: %+v", s)
		}
	}
}

func TestNumClass(t *testing.T) {
	tests := map[string]string{
		"12":    "[0-9]+",
		"１二十":   "[０-９二十]+",
		"三一十百3": "[0-9一三十百]+",
	}
	for in, want := range tests {
		if got := numClass(in); got != want {
			t.Errorf("numClass(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestSuggestSamples(t *testing.T) {
	got := Suggest(book(8, true, func(i int) string { return fmt.Sprintf("　第%d章", i) }))
	if len(got) == 0 || got[0].Count != 8 || len(got[0].Samples) != maxSamples || got[0].Samples[0] != "第1章" {
		t.Errorf("Suggest = %+v", got)
	}
}

func TestParseNum(t *testing.T) {
	tests := map[string]int{
		"12": 12, "１２": 12, "十": 10, "十一": 11, "二十": 20, "一百零三": 103,
		"三千五百": 3500, "一萬二千": 12000, "一二": 12, "〇": 0, "兩百": 200,
	}
	for in, want := range tests {
		if got, ok := parseNum([]rune(in)); !ok || got != want {
			t.Errorf("parseNum(%q) = %d, %v; want %d", in, got, ok, want)
		}
	}
}
