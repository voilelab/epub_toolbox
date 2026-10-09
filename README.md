# 🧰 EPUB 工具箱

![](preview.png)

將純文字小說或圖片製作成 EPUB 電子書。以
[ToolGUI](https://github.com/voilelab/toolgui) 撰寫並編譯為 WebAssembly，
完全在瀏覽器中執行，檔案不會離開你的電腦。

線上版以 [Umami](https://umami.is/) 統計匿名的瀏覽與下載次數，不使用 cookie，也不會傳送檔案內容。

介面語言依瀏覽器語言自動選擇：中文（`zh*`）或英文；可用首頁的連結或網址參數
`?lang=en`、`?lang=zh-TW` 切換。

**線上使用：<https://voilelab.github.io/epub_toolbox/>**

遇到問題或有建議，歡迎[回報問題](https://github.com/voilelab/epub_toolbox/issues/new?template=bug.yml)或[建議功能](https://github.com/voilelab/epub_toolbox/issues/new?template=feature.yml)。

## 使用方式

### 📘 小說 TXT 轉 EPUB

將小說純文字檔切分成章節並製作成 EPUB。

1. 選擇小說的 TXT 檔。編碼（UTF-8、Big5、GB18030 等）會自動偵測，也可手動更改。
2. 用比對章節標題的正規表示式切分章節，並可用封鎖清單排除不該是標題的行。
   🧪 標題格式（如 `第N章`、`【N】`、`Chapter N`、楔子、番外）會依本書內容自動偵測並填入（實驗性），
   無法確定時使用預設值；可在「偵測到的標題格式（實驗性）」查看其他候選。
3. 填寫書籍資訊。
4. 在「樣式」分頁選擇排版模板：閱讀器預設、中文橫排（首行縮排）或中文直排（由右向左翻頁），
   再視需要微調首行縮排、段距、左右對齊、標題置中與直排。字體、字級與顏色交給閱讀器設定。
   進階使用者可在「自訂 CSS」加入規則，會接在產生的 CSS 之後。
5. 按下 `下載` 製作並儲存 EPUB。

### 🖼️ 圖片轉 EPUB（實驗性）

> 🧪 此功能仍在實驗階段，輸出結果可能不穩定。

將圖片打包成 EPUB，每頁一張圖。

1. 選擇圖片（PNG、JPEG、GIF 或 WebP）、其 zip 壓縮檔，或兩者混用。頁面依檔名的
   自然順序排列（`2.png` 在 `10.png` 之前）。
2. 填寫書籍資訊。
3. 按下 `下載` 製作並儲存 EPUB。

## 執行

需要 Go 1.27.2 以上（或設定 `GOTOOLCHAIN=auto`）。

```bash
# 瀏覽器（WebAssembly）
go tool toolgui-wasm serve -manifest cmd/epub_toolbox/web/manifest.json -assets cmd/epub_toolbox/web/assets -icon assets/favicon.ico -head cmd/epub_toolbox/web/head.html ./cmd/epub_toolbox

# 靜態網站輸出至 dist/
go tool toolgui-wasm build -manifest cmd/epub_toolbox/web/manifest.json -assets cmd/epub_toolbox/web/assets -icon assets/favicon.ico -head cmd/epub_toolbox/web/head.html -o dist ./cmd/epub_toolbox

# 本機伺服器 http://127.0.0.1:3000（英文介面加 -lang en）
go run ./cmd/epub_toolbox
```

測試與覆蓋率：

```bash
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out   # 或 -html=cover.out
```

只有 `release/*` 分支會部署至 GitHub Pages（需在 repo 設定中將 Pages 來源設為
GitHub Actions，並在 `github-pages` environment 允許 `release/*` 部署）。

發版：在 Actions 執行 **Release** workflow（從 `main`），輸入版本（如 `v1.2.3`）。
會將 `main` fast-forward 到 `release/1.x`、建立 tag 與 GitHub Release，並觸發部署。

## 專案結構

- `internal/epub`：EPUB 3 寫入器，僅使用標準函式庫
- `internal/novel`：編碼偵測、章節切分
- `internal/imgbook`：圖片與 zip 讀取、圖片書
- `cmd/epub_toolbox/web`：Web app manifest 與圖示，瀏覽器版與本機伺服器共用
- `cmd/epub_toolbox`：ToolGUI 介面；`main_wasm.go` 供瀏覽器使用，
  `main_server.go` 用於原生執行檔
