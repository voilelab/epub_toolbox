package main

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/voilelab/epub_toolbox/internal/imgbook"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

type pickedImages struct {
	imgs []imgbook.Image
	err  error
}

func ImagesPage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, tr("🖼️ 圖片轉 EPUB", "🖼️ Images to EPUB"))
	tgcomp.Text(p.Main, tr("將圖片打包成 EPUB，每頁一張圖。", "Pack images into an EPUB, one image per page."))

	files := tgcomp.MultiFileUpload(p.Sidebar, tr("選擇圖片或 zip 檔", "Choose images or zip files"),
		".zip,application/zip,.png,.jpg,.jpeg,.gif,.webp")
	tgcomp.Caption(p.Sidebar, tr("PNG、JPEG、GIF 或 WebP，或其 zip 壓縮檔；頁面依檔名排序（2 在 10 之前）。",
		"PNG, JPEG, GIF or WebP, or zip archives of them; pages are sorted by file name (2 before 10)."))
	if files == nil {
		tgcomp.MessageInfo(p.Main, tr("👈 請選擇圖片或圖片的 zip 壓縮檔。", "👈 Choose images or a zip archive of images."))
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

	z := memo(p.State, "images", fileKey, func() pickedImages {
		imgs, err := imgbook.Read(picked)
		return pickedImages{imgs, err}
	})
	if z.err != nil {
		tgcomp.MessageDanger(p.Main, z.err.Error())
		return nil
	}
	imgs := z.imgs
	tgcomp.Text(p.Sidebar, tr("圖片數：", "Images: ")+strconv.Itoa(len(imgs)))

	tabPreview, tabMeta := tgcomp.Tab2(p.Main, tr("圖片預覽", "Preview"), tr("書籍資訊", "Book Info"))

	names := make([]string, len(imgs))
	for i, img := range imgs {
		names[i] = fmt.Sprintf("%d. %s", i+1, img.Name)
	}
	if idx := tgcomp.Select(tabPreview, tr("檔名", "File"), names, (&tgcomp.SelectConf{
		Base: tgframe.Base{ID: "image_" + fileKey},
	}).SetDefault(0)); idx != nil {
		tgcomp.Image(tabPreview, imgs[*idx].Data, &tgcomp.ImageConf{Width: "360px"})
	}

	// A lone zip names the book.
	defTitle := ""
	if len(files) == 1 && strings.HasSuffix(files[0].Name, ".zip") {
		defTitle = strings.TrimSuffix(path.Base(files[0].Name), ".zip")
	}
	meta := metaForm(tabMeta, fileKey, defTitle, "")
	firstAsCover := false
	if !meta.hasCover() {
		firstAsCover = tgcomp.Checkbox(tabMeta, tr("以第一張圖片作為封面", "Use the first image as the cover"))
	}

	exportEpub(p, meta.Title, func() ([]byte, error) {
		cover, err := meta.cover(p.Context)
		if err != nil {
			return nil, fmt.Errorf(tr("封面：%w", "cover: %w"), err)
		}
		if firstAsCover {
			cover = imgs[0].Data
		}
		return imgbook.Build(imgbook.Meta{
			Title: meta.Title, Author: meta.Author, Intro: meta.Intro,
			Language: meta.Language, Cover: cover,
		}, imgs)
	})
	return nil
}
