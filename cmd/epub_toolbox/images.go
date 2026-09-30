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
	tgcomp.Title(p.Main, "🖼️ Images-EPUB Builder")
	tgcomp.Text(p.Main, "Pack images into an EPUB, one image per page.")

	files := tgcomp.MultiFileUpload(p.Sidebar, "Choose images or zips",
		".zip,application/zip,.png,.jpg,.jpeg,.gif,.webp")
	tgcomp.Caption(p.Sidebar, "PNG, JPEG, GIF or WebP, or zips of them; "+
		"pages follow filename order (2 before 10).")
	if files == nil {
		tgcomp.MessageInfo(p.Main, "👈 Please select images or a zip file of images.")
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
	tgcomp.Text(p.Sidebar, "Image count: "+strconv.Itoa(len(imgs)))

	tabPreview, tabMeta := tgcomp.Tab2(p.Main, "Images Preview", "Book Meta")

	names := make([]string, len(imgs))
	for i, img := range imgs {
		names[i] = fmt.Sprintf("%d. %s", i+1, img.Name)
	}
	if idx := tgcomp.Select(tabPreview, "Filename", names, (&tgcomp.SelectConf{
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
		firstAsCover = tgcomp.Checkbox(tabMeta, "Use the first image as the cover")
	}

	exportEpub(p, meta.Title, func() ([]byte, error) {
		cover, err := meta.cover(p.Context)
		if err != nil {
			return nil, fmt.Errorf("book cover: %w", err)
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
