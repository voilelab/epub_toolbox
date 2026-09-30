# 🧰 EPUB Toolbox

![](preview.png)

Build EPUB books from plain text novels or images. Written with
[ToolGUI](https://github.com/voilelab/toolgui) and compiled to WebAssembly, so
it runs entirely in your browser: files never leave your computer.

**Use it online: <https://voilelab.github.io/epub_toolbox/>**

## Usage

### 📘 Novel TXT-EPUB Builder

Split a novel's plain text file into chapters and build an EPUB.

1. Choose the TXT file containing the novel. The encoding (UTF-8, Big5,
   GB18030, ...) is detected automatically and can be changed.
2. Split it into chapters with regular expressions matching chapter titles,
   plus a block list for lines that must not be titles.
3. Fill in the book's metadata.
4. Click `Prepare EPUB`, then download the EPUB.

### 🖼️ Images-EPUB Builder

Pack images into an EPUB, one image per page.

1. Choose a zip file of images (PNG, JPEG, GIF or WebP). Pages follow natural
   filename order (`2.png` before `10.png`).
2. Fill in the book's metadata.
3. Click `Prepare EPUB`, then download the EPUB.

A cover URL only works when its host allows cross-origin reads; otherwise
download the image and upload it.

## Run

Requires Go 1.27.1 or later (or `GOTOOLCHAIN=auto`).

```bash
# Browser (WebAssembly)
go tool toolgui-wasm serve ./cmd/epub_toolbox

# Static site in dist/
go tool toolgui-wasm build -o dist ./cmd/epub_toolbox

# Local server at http://127.0.0.1:3000
go run ./cmd/epub_toolbox
```

Pushing to `main` deploys to GitHub Pages (set the Pages source to GitHub
Actions in the repo settings).

## Project layout

- `internal/epub`: EPUB 3 writer, standard library only
- `internal/novel`: encoding detection, chapter splitting
- `internal/imgbook`: zip reading, image book
- `internal/fetch`: cover download
- `cmd/epub_toolbox`: ToolGUI UI; `main_wasm.go` for the browser,
  `main_server.go` for a native binary
