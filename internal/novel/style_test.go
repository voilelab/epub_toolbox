package novel

import (
	"strings"
	"testing"
)

func TestStyleCSS(t *testing.T) {
	if got := (Style{}).CSS(); got != baseCSS {
		t.Errorf("zero Style CSS = %q, want base only", got)
	}
	tests := []struct {
		s    Style
		want []string
	}{
		{Style{Indent: true}, []string{"p { text-indent: 2em; }"}},
		{Style{NoGap: true, Justify: true}, []string{"p { margin: 0; text-align: justify; }"}},
		{Style{CenterTitle: true}, []string{"h1 { text-align: center; margin: 1em 0 2em; }"}},
		{Style{Vertical: true}, []string{"writing-mode: vertical-rl;", "h1 { margin: 0 1em 0 2em; }"}},
	}
	for _, tt := range tests {
		css := tt.s.CSS()
		for _, w := range tt.want {
			if !strings.Contains(css, w) {
				t.Errorf("%+v: CSS missing %q:\n%s", tt.s, w, css)
			}
		}
	}
}

// Templates must keep the CSS they shipped with.
func TestTemplateCSS(t *testing.T) {
	want := map[Template]string{
		TemplateReader: baseCSS,
		TemplateHorizontal: baseCSS + `p { text-indent: 2em; margin: 0; }
h1 { text-align: center; margin: 1em 0 2em; }
`,
		TemplateVertical: baseCSS + `html { -epub-writing-mode: vertical-rl; -webkit-writing-mode: vertical-rl; writing-mode: vertical-rl; }
p { text-indent: 2em; margin: 0; }
h1 { margin: 0 1em 0 2em; }
`,
	}
	for _, tp := range Templates {
		if got := tp.Style().CSS(); got != want[tp] {
			t.Errorf("%s CSS =\n%s\nwant\n%s", tp.Name(), got, want[tp])
		}
	}
}

func TestStyleExtra(t *testing.T) {
	s := TemplateHorizontal.Style()
	base := s.CSS()
	s.Extra = "  p { line-height: 1.8; }\n\n"
	if got, want := s.CSS(), base+"p { line-height: 1.8; }\n"; got != want {
		t.Errorf("CSS =\n%s\nwant\n%s", got, want)
	}
	s.Extra = " \n "
	if got := s.CSS(); got != base {
		t.Errorf("blank Extra changed CSS:\n%s", got)
	}
}
