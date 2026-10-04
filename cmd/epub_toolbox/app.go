package main

import (
	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

var tr = i18n.T

// langSwitch shows a link that reloads the app in the other language.
var langSwitch bool

func newApp() *tgframe.App {
	app := tgframe.NewApp()
	app.SetTitle(tr("EPUB 工具箱", "EPUB Toolbox"))
	app.AddPageByConfig(&tgframe.PageConfig{Name: "index", Title: tr("EPUB 工具箱", "EPUB Toolbox"), Emoji: "🧰"}, HomePage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "novel", Title: tr("小說 TXT 轉 EPUB", "Novel TXT to EPUB"), Emoji: "📘"}, NovelPage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "regex", Title: tr("用正規表示式篩選章節標題", "Matching Chapter Titles with Regex"), Emoji: "📑"}, RegexPage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "images", Title: tr("圖片轉 EPUB（實驗性）", "Images to EPUB (Experimental)"), Emoji: "🖼️"}, ImagesPage)
	return app
}

func HomePage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, tr("🧰 EPUB 工具箱", "🧰 EPUB Toolbox"))
	if langSwitch {
		tgcomp.Link(p.Main, tr("🌐 English", "🌐 中文"), tr("?lang=en", "?lang=zh-TW"))
	}
	tgcomp.Markdown(p.Main, tr(`所有處理都在瀏覽器中完成，檔案不會離開你的電腦。線上版以 Umami 統計匿名的瀏覽與下載次數，不使用 cookie。

請從側邊導覽選擇工具。

### 📘 小說 TXT 轉 EPUB

將小說純文字檔切分成章節並製作成 EPUB。

1. 選擇小說的 TXT 檔。
2. 用比對章節標題的正規表示式切分章節。
3. 填寫書籍資訊。
4. 按下 **下載** 製作並儲存 EPUB。

### 🖼️ 圖片轉 EPUB（實驗性）

將圖片打包成 EPUB，每頁一張圖。此功能仍在實驗階段，輸出結果可能不穩定。

1. 選擇圖片（PNG、JPEG、GIF 或 WebP）或其 zip 壓縮檔；頁面依檔名排序。
2. 填寫書籍資訊。
3. 按下 **下載** 製作並儲存 EPUB。

原始碼：[voilelab/epub_toolbox](https://github.com/voilelab/epub_toolbox)，以 [ToolGUI](https://github.com/voilelab/toolgui) 製作。`,
		`Everything runs in your browser; your files never leave your computer. The online version counts anonymous visits and downloads with Umami, without cookies.

Pick a tool from the side navigation.

### 📘 Novel TXT to EPUB

Split a plain-text novel into chapters and make an EPUB.

1. Choose the novel's TXT file.
2. Split it into chapters with regexes that match chapter titles.
3. Fill in the book info.
4. Press **Download** to build and save the EPUB.

### 🖼️ Images to EPUB (Experimental)

Pack images into an EPUB, one image per page. This feature is experimental; output may be unreliable.

1. Choose images (PNG, JPEG, GIF or WebP) or zip archives of them; pages are sorted by file name.
2. Fill in the book info.
3. Press **Download** to build and save the EPUB.

Source: [voilelab/epub_toolbox](https://github.com/voilelab/epub_toolbox), made with [ToolGUI](https://github.com/voilelab/toolgui).`))
	return nil
}

func RegexPage(p *tgframe.Params) error {
	if i18n.EN() {
		return regexPageEN(p)
	}
	tgcomp.Title(p.Main, "📑 用正規表示式篩選章節標題")
	tgcomp.Markdown(p.Main, "三分鐘帶你大概了解怎麼用正規表示式篩選小說章節標題……大概吧？\n\n"+
		"網路上不難找到正規表示式的詳細教學，"+
		"但如果你只想知道哪些表示式會符合哪些章節標題，看這裡就好。\n\n"+
		"表示式從一行的**開頭**開始符合時，該行就是標題。"+
		"語法為 Go 的 [RE2](https://github.com/google/re2/wiki/Syntax)，"+
		"因此不支援 lookahead 與反向參照。")

	tgcomp.Subtitle(p.Main, "第.*章")
	tgcomp.Text(p.Main, "這個表示式會符合：")
	tgcomp.Code(p.Main, "第一章 開始\n第三十三章 你想不到吧\n第022章 結局", &tgcomp.CodeConf{Language: "text"})
	tgcomp.Text(p.Main, "也就是任何以「第」開頭、後面接著出現「章」的行。")

	tgcomp.Subtitle(p.Main, `第\d+章`)
	tgcomp.Text(p.Main, "這個表示式會符合：")
	tgcomp.Code(p.Main, "第1章 開始\n第2345章 你想不到吧\n第022章 結局", &tgcomp.CodeConf{Language: "text"})
	tgcomp.Text(p.Main, "但中間只要有非數字就不符合，所以這些不是標題：")
	tgcomp.Code(p.Main, "第一章 開始\n第三十三章 你想不到吧", &tgcomp.CodeConf{Language: "text"})
	return nil
}

func regexPageEN(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "📑 Matching Chapter Titles with Regex")
	tgcomp.Markdown(p.Main, "A three-minute rough guide to matching novel chapter titles with regexes… roughly.\n\n"+
		"Detailed regex tutorials are easy to find online, "+
		"but if you only want to know which expressions match which chapter titles, this is enough.\n\n"+
		"A line is a title when the expression matches from its **start**. "+
		"The syntax is Go's [RE2](https://github.com/google/re2/wiki/Syntax), "+
		"so lookahead and backreferences are not supported.")

	tgcomp.Subtitle(p.Main, "Chapter.*")
	tgcomp.Text(p.Main, "This expression matches:")
	tgcomp.Code(p.Main, "Chapter One: Beginning\nChapter 33 You Didn't See This Coming\nChapter022 The End", &tgcomp.CodeConf{Language: "text"})
	tgcomp.Text(p.Main, "That is, any line starting with \"Chapter\".")

	tgcomp.Subtitle(p.Main, `Chapter \d+`)
	tgcomp.Text(p.Main, "This expression matches:")
	tgcomp.Code(p.Main, "Chapter 1: Beginning\nChapter 2345 You Didn't See This Coming\nChapter 022 The End", &tgcomp.CodeConf{Language: "text"})
	tgcomp.Text(p.Main, "But it needs a space then digits, so these are not titles:")
	tgcomp.Code(p.Main, "Chapter One: Beginning\nChapter022 The End", &tgcomp.CodeConf{Language: "text"})
	return nil
}
