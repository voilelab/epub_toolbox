package main

import (
	"fmt"
	"path"
	"strconv"

	"github.com/voilelab/epub_toolbox/internal/imgbook"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

func ImagesPage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, tr("🖼️ 圖片轉 EPUB（實驗性）", "🖼️ Images to EPUB (Experimental)"))
	tgcomp.MessageWarning(p.Main, tr("🧪 此功能仍在實驗階段，輸出結果可能不穩定，請自行確認產生的 EPUB。",
		"🧪 This feature is experimental. Output may be unreliable; check the generated EPUB yourself."))
	tgcomp.Text(p.Main, tr("將圖片打包成 EPUB，每頁一張圖。", "Pack images into an EPUB, one image per page."))

	bar := tgcomp.Toolbar(p.Main)
	files := tgcomp.MultiFileUpload(bar, tr("選擇圖片或 zip 檔", "Choose images or zip files"),
		".zip,application/zip,.png,.jpg,.jpeg,.gif,.webp")
	if files == nil {
		tgcomp.MessageInfo(p.Main, tr("👆 請選擇圖片（PNG、JPEG、GIF 或 WebP）或其 zip 壓縮檔。",
			"👆 Choose images (PNG, JPEG, GIF or WebP) or zip archives of them."))
		return nil
	}
	picked := make([]imgbook.Image, len(files))
	hashes := make([]any, len(files))
	for i, f := range files {
		b, err := f.Bytes()
		if err != nil {
			return err
		}
		picked[i] = imgbook.Image{Name: f.Name, Data: b}
		hashes[i] = f.Name + ":" + hashBytes(b)
	}
	fileKey := hashOf(hashes...)

	imgs, err := p.State.Memo("images", fileKey, func() ([]imgbook.Image, error) {
		imgs, err := imgbook.Read(picked)
		if err == nil {
			tgcomp.Toast(p.Main, fmt.Sprintf(tr("已讀取 %d 張圖片", "Read %d images"), len(imgs)), &tgcomp.ToastConf{Icon: "🖼️"})
		}
		return imgs, err
	})
	if err != nil {
		showError(p.Main, toolImages, err)
		return nil
	}
	stats := tgcomp.Toolbar(p.Main, &tgcomp.ToolbarConf{Base: tgframe.Base{ID: "stats"}})
	tgcomp.Metric(stats, tr("圖片數", "Images"), strconv.Itoa(len(imgs)))

	tabPreview, tabMeta := tgcomp.Tab2(p.Main, tr("圖片預覽", "Preview"), tr("書籍資訊", "Book Info"))

	names := make([]string, len(imgs))
	for i, img := range imgs {
		names[i] = fmt.Sprintf("%d. %s", i+1, img.Name)
	}
	if idx := tgcomp.Select(tabPreview, tr("檔名", "File"), names, (&tgcomp.SelectConf{
		Base: tgframe.Base{ID: "image_" + fileKey},
	}).SetDefault(0)); idx != nil {
		tgcomp.Caption(tabPreview, tr("頁面依檔名排序（2 在 10 之前）。", "Pages are sorted by file name (2 before 10)."))
		tgcomp.Image(tabPreview, imgs[*idx].Data, &tgcomp.ImageConf{Width: "360px"})
	}

	// A lone zip names the book.
	defTitle := ""
	if len(files) == 1 {
		if name, ok := cutExt(path.Base(files[0].Name), ".zip"); ok {
			defTitle = name
		}
	}
	meta := metaForm(tabMeta, fileKey, defTitle, "")
	firstAsCover := false
	if meta.coverFile == nil {
		firstAsCover = tgcomp.Checkbox(tabMeta, tr("以第一張圖片作為封面", "Use the first image as the cover"))
	}

	exportEpub(stats, "images", meta.Title, func() ([]byte, error) {
		b, err := meta.book()
		if err != nil {
			return nil, err
		}
		if firstAsCover {
			if err := b.SetCover(imgs[0].Data); err != nil {
				return nil, err
			}
		}
		return imgbook.Build(b, imgs)
	})
	return nil
}
