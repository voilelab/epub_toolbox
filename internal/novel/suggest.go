package novel

import (
	"math"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Suggestion is a detected title pattern for the allowlist.
type Suggestion struct {
	Pattern string   // allowlist regex
	Score   float64  // 0–1, hand-tuned
	Count   int      // matched lines
	Samples []string // first matched lines
}

const (
	// numClass matches Arabic, full-width and Chinese numerals.
	numClass = `[0-9０-９零〇一二三四五六七八九十百千萬万兩两]+`
	// spaceClass matches leading indentation, incl. the ideographic space.
	spaceClass = `[ \t　]*`

	maxTitleLen  = 40 // runes; longer lines are not shape candidates
	maxPrefixLen = 8  // runes before the number
	minCount     = 3
	maxSamples   = 5
	maxSuggest   = 3
)

// titleMarks may follow the number in a title shape.
const titleMarks = "章回節节卷部篇集話话幕】]）)．.、:："

// specials are unnumbered titles added to a Chinese pattern when present.
var specials = []string{"序章", "序幕", "楔子", "引子", "前言", "尾聲", "尾声", "終章", "终章", "後記", "后记", "番外"}

// Suggest infers title patterns from the book's lines, best first.
// Pass lines with blank ones kept: blank neighbors are a signal.
func Suggest(lines []string) []Suggestion {
	shapes := map[string]int{}
	for _, l := range lines {
		if k, ok := shapeOf(l); ok {
			shapes[k]++
		}
	}

	var out []Suggestion
	for k, n := range shapes {
		if n < minCount {
			continue
		}
		if s, ok := score(lines, k); ok {
			out = append(out, s)
		}
	}
	slices.SortFunc(out, func(a, b Suggestion) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Pattern, b.Pattern)
	})
	if len(out) > maxSuggest {
		out = out[:maxSuggest]
	}
	return out
}

// shape is a title shape: literal prefix, a number, then an optional mark.
type shape struct {
	prefix string
	mark   rune // 0 if none
}

// key encodes s as prefix + "\x00" + mark.
func (s shape) key() string { return s.prefix + "\x00" + string(s.mark) }

func parseKey(k string) shape {
	p, m, _ := strings.Cut(k, "\x00")
	r, _ := utf8.DecodeRuneInString(m)
	if m == "" {
		r = 0
	}
	return shape{p, r}
}

// shapeOf returns the shape key of a title-like line.
func shapeOf(line string) (string, bool) {
	t := trimSpace(line)
	if t == "" || utf8.RuneCountInString(t) > maxTitleLen {
		return "", false
	}
	rs := []rune(t)
	i := 0
	for i < len(rs) && !isNum(rs[i]) {
		if isEndPunct(rs[i]) {
			return "", false
		}
		i++
	}
	if i == len(rs) || i > maxPrefixLen {
		return "", false
	}
	prefix := string(rs[:i])
	for i < len(rs) && isNum(rs[i]) {
		i++
	}
	s := shape{prefix: prefix}
	if i < len(rs) && strings.ContainsRune(titleMarks, rs[i]) {
		s.mark = rs[i]
	}
	// A bare number must stand alone, or "一個人…" would be a title.
	if prefix == "" && s.mark == 0 && i < len(rs) && !isSpace(rs[i]) {
		return "", false
	}
	return s.key(), true
}

// pattern returns the allowlist regex of a shape.
func (s shape) pattern() string {
	var sb strings.Builder
	sb.WriteString(spaceClass)
	for _, f := range strings.Fields(s.prefix) {
		sb.WriteString(regexp.QuoteMeta(f))
		sb.WriteString(`[ 　]*`)
	}
	sb.WriteString(numClass)
	switch {
	case s.mark != 0:
		sb.WriteString(regexp.QuoteMeta(string(s.mark)))
	case s.prefix == "":
		sb.WriteString(`(?:[ \t　]|$)`)
	}
	return sb.String()
}

// score runs the shape's pattern over lines and rates the split.
func score(lines []string, k string) (Suggestion, bool) {
	sh := parseKey(k)
	main := sh.pattern()
	re := regexp.MustCompile("^(?:" + main + ")")

	var idx []int
	for i, l := range lines {
		if re.MatchString(l) {
			idx = append(idx, i)
		}
	}
	if len(idx) < minCount {
		return Suggestion{}, false
	}

	var short, clean, isolated float64
	nums := make([][]int, len(idx))
	for j, i := range idx {
		t := trimSpace(lines[i])
		if utf8.RuneCountInString(t) <= maxTitleLen {
			short++
		}
		if r, _ := utf8.DecodeLastRuneInString(t); !isEndPunct(r) {
			clean++
		}
		if blankAt(lines, i-1) || blankAt(lines, i+1) {
			isolated++
		}
		nums[j] = numbersIn(t, 2)
	}
	n := float64(len(idx))
	short, clean, isolated = short/n, clean/n, isolated/n

	// Blank lines only count if the book is not blank-separated throughout.
	if r := blankRate(lines); r > 0.3 {
		isolated = 0.5
	}

	z := -5.0 +
		2.0*sequential(nums) +
		1.5*clean +
		1.0*short +
		1.0*isolated +
		2.0*gapsOK(idx, len(lines)) +
		1.5*math.Min(1, math.Log10(n)/2)

	pat := main
	if sh.isCJK() {
		if sp := presentSpecials(lines); len(sp) > 0 {
			pat = spaceClass + "(?:" + strings.TrimPrefix(main, spaceClass) + "|" + strings.Join(sp, "|") + ")"
		}
	}

	s := Suggestion{Pattern: pat, Score: 1 / (1 + math.Exp(-z)), Count: len(idx)}
	for _, i := range idx[:min(len(idx), maxSamples)] {
		s.Samples = append(s.Samples, trimSpace(lines[i]))
	}
	return s, true
}

func (s shape) isCJK() bool {
	for _, r := range s.prefix + string(s.mark) {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return s.prefix == "" // a bare number shape, likely CJK too
}

// sequential is the share of consecutive titles numbered n, n+1 or restarting
// at 1, using whichever of the first two numbers runs best.
func sequential(nums [][]int) float64 {
	best := 0.0
	for pos := range 2 {
		ok, total := 0, 0
		for j := 1; j < len(nums); j++ {
			if len(nums[j-1]) <= pos || len(nums[j]) <= pos {
				continue
			}
			a, b := nums[j-1][pos], nums[j][pos]
			total++
			if b == a+1 || (b == 1 && a != 1) {
				ok++
			}
		}
		if total > 0 {
			best = max(best, float64(ok)/float64(total))
		}
	}
	return best
}

// gapsOK is the share of chapters with a plausible length: not a list of
// adjacent lines and not one swallowing a large part of the book.
func gapsOK(idx []int, total int) float64 {
	ends := append(idx[1:len(idx):len(idx)], total)
	ok := 0
	for j, i := range idx {
		g := ends[j] - i
		if g >= 3 && g <= max(2000, total/4) {
			ok++
		}
	}
	return float64(ok) / float64(len(idx))
}

// presentSpecials returns the specials used as standalone titles.
func presentSpecials(lines []string) []string {
	var out []string
	for _, w := range specials {
		for _, l := range lines {
			t := trimSpace(l)
			rest, ok := strings.CutPrefix(t, w)
			if !ok || utf8.RuneCountInString(t) > 20 {
				continue
			}
			if r, _ := utf8.DecodeRuneInString(rest); rest == "" || isSpace(r) || r == ':' || r == '：' {
				out = append(out, regexp.QuoteMeta(w))
				break
			}
		}
	}
	return out
}

// numbersIn parses up to n numbers in s.
func numbersIn(s string, n int) []int {
	var out []int
	rs := []rune(s)
	for i := 0; i < len(rs) && len(out) < n; {
		if !isNum(rs[i]) {
			i++
			continue
		}
		j := i
		for j < len(rs) && isNum(rs[j]) {
			j++
		}
		if v, ok := parseNum(rs[i:j]); ok {
			out = append(out, v)
		}
		i = j
	}
	return out
}

var cnDigits = map[rune]int{
	'零': 0, '〇': 0, '一': 1, '二': 2, '兩': 2, '两': 2, '三': 3, '四': 4,
	'五': 5, '六': 6, '七': 7, '八': 8, '九': 9,
}

var cnUnits = map[rune]int{'十': 10, '百': 100, '千': 1000}

// parseNum parses Arabic or Chinese numerals, e.g. "12", "１２", "一百零三", "一二".
func parseNum(rs []rune) (int, bool) {
	if d, ok := digit(rs[0]); ok {
		v := d
		for _, r := range rs[1:] {
			d, ok := digit(r)
			if !ok {
				return 0, false
			}
			v = v*10 + d
		}
		return v, true
	}

	total, section, cur, units := 0, 0, 0, false
	for _, r := range rs {
		switch {
		case r == '萬' || r == '万':
			total += (section + cur) * 10000
			section, cur, units = 0, 0, true
		case cnUnits[r] > 0:
			if cur == 0 {
				cur = 1 // 十 = 10
			}
			section += cur * cnUnits[r]
			cur, units = 0, true
		default:
			d, ok := cnDigits[r]
			if !ok {
				return 0, false
			}
			if !units {
				cur = cur*10 + d // positional, e.g. 一二 = 12
			} else {
				cur = d
			}
		}
	}
	return total + section + cur, true
}

func digit(r rune) (int, bool) {
	switch {
	case r >= '0' && r <= '9':
		return int(r - '0'), true
	case r >= '０' && r <= '９':
		return int(r - '０'), true
	}
	return 0, false
}

func isNum(r rune) bool {
	if _, ok := digit(r); ok {
		return true
	}
	_, ok := cnDigits[r]
	return ok || cnUnits[r] > 0 || r == '萬' || r == '万'
}

func isSpace(r rune) bool { return unicode.IsSpace(r) || r == '　' }

func isEndPunct(r rune) bool { return strings.ContainsRune("。！？!?，,；;…」』\"”", r) }

func trimSpace(s string) string { return strings.TrimFunc(s, isSpace) }

func blankAt(lines []string, i int) bool {
	return i < 0 || i >= len(lines) || trimSpace(lines[i]) == ""
}

func blankRate(lines []string) float64 {
	if len(lines) == 0 {
		return 0
	}
	n := 0
	for _, l := range lines {
		if trimSpace(l) == "" {
			n++
		}
	}
	return float64(n) / float64(len(lines))
}
