package main

import (
	"encoding/base64"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/voilelab/epub_toolbox/internal/epub"
	"github.com/voilelab/epub_toolbox/internal/imgbook"
	"github.com/voilelab/toolgui/toolgui/tgcomp"
	"github.com/voilelab/toolgui/toolgui/tgframe"
)

type zipImages struct {
	imgs []imgbook.Image
	err  error
}

func ImagesPage(p *tgframe.Params) error {
	tgcomp.Title(p.Main, "🖼️ Images-EPUB Builder")
	tgcomp.Text(p.Main, "Pack images into an EPUB, one image per page.")

	file := tgcomp.FileUpload(p.Sidebar, "Choose a zip of images", ".zip,application/zip")
	tgcomp.Caption(p.Sidebar, "PNG, JPEG, GIF or WebP; pages follow filename order (2 before 10).")
	if file == nil {
		tgcomp.MessageInfo(p.Main, "👈 Please select a zip file of images.")
		return nil
	}
	raw, err := file.Bytes()
	if err != nil {
		return err
	}
	fileKey := hashBytes(raw)

	z := memo(p.State, "images", fileKey, func() zipImages {
		imgs, err := imgbook.ReadZip(raw)
		return zipImages{imgs, err}
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
		tgcomp.Image(tabPreview, dataURI(imgs[*idx].Data), &tgcomp.ImageConf{Width: "360px"})
	}

	defTitle := strings.TrimSuffix(path.Base(file.Name), ".zip")
	meta := metaForm(tabMeta, fileKey, defTitle, "")
	firstAsCover := false
	if !meta.hasCover() {
		firstAsCover = tgcomp.Checkbox(tabMeta, "Use the first image as the cover")
	}

	key := hashOf(fileKey, meta.key(), firstAsCover)
	exportEpub(p, key, meta.Title, func() ([]byte, error) {
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

func dataURI(b []byte) string {
	mime, _, _ := epub.ImageType(b)
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b)
}
