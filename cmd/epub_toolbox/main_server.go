//go:build !(js && wasm)

package main

import (
	"embed"
	"encoding/json"
	"flag"
	"io/fs"
	"log"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgexec"
)

// web is shared with the wasm build, which takes it as build flags.
//
//go:embed web
var web embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:3000", "listen address")
	lang := flag.String("lang", "zh-TW", "UI language: zh-TW or en")
	flag.Parse()
	i18n.Set(*lang)

	e := tgexec.NewWebExecutor(newApp())
	if err := setWebConfig(e); err != nil {
		log.Fatal(err)
	}

	log.Printf("Serving on http://%s", *addr)
	if err := e.StartService(*addr); err != nil {
		log.Fatal(err)
	}
}

// setWebConfig serves web/manifest.json and web/assets, and links the stylesheet.
func setWebConfig(e *tgexec.WebExecutor) error {
	bs, err := web.ReadFile("web/manifest.json")
	if err != nil {
		return err
	}
	manifest := &tgexec.Manifest{}
	if err := json.Unmarshal(bs, manifest); err != nil {
		return err
	}
	e.SetManifest(manifest)

	assets, err := fs.Sub(web, "web/assets")
	if err != nil {
		return err
	}
	e.SetAssets(assets)
	// Analytics in head.html are for the deployed wasm build only.
	e.SetHeadHTML(`<link rel="stylesheet" href="assets/style.css" />`)
	return nil
}
