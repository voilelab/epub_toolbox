package main

import (
	"net/url"

	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

const issuesURL = "https://github.com/voilelab/epub_toolbox/issues"

// Dropdown options in .github/ISSUE_TEMPLATE/bug.yml.
const (
	toolNovel  = "小說 TXT 轉 EPUB / Novel TXT to EPUB"
	toolImages = "圖片轉 EPUB / Images to EPUB"
)

// bugURL opens the bug form prefilled with tool and errMsg.
func bugURL(tool, errMsg string) string {
	q := url.Values{"template": {"bug.yml"}}
	if tool != "" {
		q.Set("tool", tool)
	}
	if errMsg != "" {
		q.Set("error", errMsg)
	}
	return issuesURL + "/new?" + q.Encode()
}

// feedbackMarkdown links to the bug and feature forms.
func feedbackMarkdown() string {
	return tr("💬 意見回饋：[回報問題](", "💬 Feedback: [report a bug](") + bugURL("", "") +
		tr(")・[建議功能](", ") · [suggest a feature](") + issuesURL + "/new?template=feature.yml)"
}

// showError shows err with a link to report it.
// Markdown links open in a new tab, so the page state survives.
func showError(c *tgframe.Container, tool string, err error) {
	tgcomp.MessageDanger(c, err.Error())
	tgcomp.Markdown(c, tr("覺得這是 bug？[回報這個問題](", "Looks like a bug? [Report it](")+bugURL(tool, err.Error())+")")
}
