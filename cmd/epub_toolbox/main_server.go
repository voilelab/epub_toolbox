//go:build !(js && wasm)

package main

import (
	"flag"
	"log"

	"github.com/voilelab/epub_toolbox/internal/i18n"
	"github.com/voilelab/toolgui/toolgui/tgexec"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:3000", "listen address")
	lang := flag.String("lang", "zh-TW", "UI language: zh-TW or en")
	flag.Parse()
	i18n.Set(*lang)

	log.Printf("Serving on http://%s", *addr)
	if err := tgexec.NewWebExecutor(newApp()).StartService(*addr); err != nil {
		log.Fatal(err)
	}
}
