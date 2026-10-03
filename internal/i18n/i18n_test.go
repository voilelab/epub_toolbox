package i18n

import "testing"

func TestSet(t *testing.T) {
	defer Set("zh-TW")
	for tag, want := range map[string]string{
		"zh-TW": "zh-TW", "zh": "zh-TW", "ZH-cn": "zh-TW",
		"en-US": "en", "ja": "en", "": "en",
	} {
		Set(tag)
		if got := Tag(); got != want {
			t.Errorf("Set(%q): Tag() = %q, want %q", tag, got, want)
		}
	}
}
