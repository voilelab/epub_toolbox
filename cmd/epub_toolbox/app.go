package main

import (
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

func newApp() *tgframe.App {
	app := tgframe.NewApp()
	app.SetTitle("EPUB 工具箱")
	app.AddPageByConfig(&tgframe.PageConfig{Name: "index", Title: "EPUB 工具箱", Emoji: "🧰"}, HomePage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "novel", Title: "小說 TXT 轉 EPUB", Emoji: "📘"}, NovelPage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "regex", Title: "用正規表示式篩選章節標題", Emoji: "📑"}, RegexPage)
	app.AddPageByConfig(&tgframe.PageConfig{Name: "images", Title: "圖片轉 EPUB", Emoji: "🖼️"}, ImagesPage)
	return app
}

func HomePage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "🧰 EPUB 工具箱")
	tgcomp.Markdown(p.Main, `所有處理都在瀏覽器中完成，檔案不會離開你的電腦。

請從側邊導覽選擇工具。

### 📘 小說 TXT 轉 EPUB

將小說純文字檔切分成章節並製作成 EPUB。

1. 選擇小說的 TXT 檔。
2. 用比對章節標題的正規表示式切分章節。
3. 填寫書籍資訊。
4. 按下 **下載** 製作並儲存 EPUB。

### 🖼️ 圖片轉 EPUB

將圖片打包成 EPUB，每頁一張圖。

1. 選擇圖片（PNG、JPEG、GIF 或 WebP）或其 zip 壓縮檔；頁面依檔名排序。
2. 填寫書籍資訊。
3. 按下 **下載** 製作並儲存 EPUB。

原始碼：[voilelab/epub_toolbox](https://github.com/voilelab/epub_toolbox)，以 [ToolGUI](https://github.com/voilelab/toolgui) 製作。`)
	return nil
}

func RegexPage(p *tgframe.Params) error {
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
