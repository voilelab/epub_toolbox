package main

import (
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

func newApp() *tgframe.App {
	app := tgframe.NewApp()
	app.SetTitle("EPUB Toolbox")
	app.AddPageByConfig(&tgframe.PageConfig{Name: "index", Title: "EPUB Toolbox", Emoji: "🧰"}, HomePage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "novel", Title: "Novel TXT-EPUB Builder", Emoji: "📘"}, NovelPage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "regex", Title: "Regex for Filter Novel Titles", Emoji: "📑"}, RegexPage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "images", Title: "Images-EPUB Builder", Emoji: "🖼️"}, ImagesPage)
	return app
}

func HomePage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "🧰 EPUB Toolbox")
	tgcomp.Markdown(p.Main, `All processing happens in your browser; files never leave your computer.

Pick a tool from the side navigation.

### 📘 Novel TXT-EPUB Builder

Split a novel's plain text file into chapters and build an EPUB.

1. Choose the TXT file containing the novel.
2. Split it into chapters with regular expressions matching chapter titles.
3. Fill in the book's metadata.
4. Click **Download** to build and save the EPUB.

### 🖼️ Images-EPUB Builder

Pack images into an EPUB, one image per page.

1. Choose images (PNG, JPEG, GIF or WebP) or zips of them; pages are sorted by filename.
2. Fill in the book's metadata.
3. Click **Download** to build and save the EPUB.

Source: [voilelab/epub_toolbox](https://github.com/voilelab/epub_toolbox), built with [ToolGUI](https://github.com/voilelab/toolgui).`)
	return nil
}

func RegexPage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "📑 Regex for Filter Novel Titles")
	tgcomp.Markdown(p.Main, "Three minutes to give you a rough idea of how to use regular "+
		"expressions to filter novel titles... roughly?\n\n"+
		"Detailed tutorials on regular expressions are easy to find on the internet, "+
		"but if you just need to know which expressions match which chapter titles, look here.\n\n"+
		"A line is a title when an expression matches from the **start** of the line. "+
		"The syntax is Go's [RE2](https://github.com/google/re2/wiki/Syntax), "+
		"so lookaheads and backreferences are not supported.")

	tgcomp.Subtitle(p.Main, "第.*章")
	tgcomp.Text(p.Main, "This expression matches:")
	tgcomp.Code(p.Main, "第一章 開始\n第三十三章 你想不到吧\n第022章 結局", &tgcomp.CodeConf{Language: "text"})
	tgcomp.Text(p.Main, "In other words, any line starting with 第 and followed later by 章.")

	tgcomp.Subtitle(p.Main, `第\d+章`)
	tgcomp.Text(p.Main, "This expression matches:")
	tgcomp.Code(p.Main, "第1章 開始\n第2345章 你想不到吧\n第022章 結局", &tgcomp.CodeConf{Language: "text"})
	tgcomp.Text(p.Main, "But any non-digit in between fails the match, so these are not titles:")
	tgcomp.Code(p.Main, "第一章 開始\n第三十三章 你想不到吧", &tgcomp.CodeConf{Language: "text"})
	return nil
}
